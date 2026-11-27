package querylog

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryLog_Search_findClient(t *testing.T) {
	const knownClientID = "client-1"
	const knownClientName = "Known Client 1"
	const unknownClientID = "client-2"

	knownClient := &Client{
		Name: knownClientName,
	}

	findClientCalls := 0
	findClient := func(ids []string) (c *Client, _ error) {
		defer func() { findClientCalls++ }()

		if len(ids) == 0 {
			return nil, nil
		}

		if ids[0] == knownClientID {
			return knownClient, nil
		}

		return nil, nil
	}

	l, err := newQueryLog(Config{
		Logger:            testLogger,
		FindClient:        findClient,
		BaseDir:           t.TempDir(),
		RotationIvl:       timeutil.Day,
		MemSize:           100,
		Enabled:           true,
		FileEnabled:       true,
		AnonymizeClientIP: false,
	})
	require.NoError(t, err)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return l.Shutdown(ctx)
	})

	q := &dns.Msg{
		Question: []dns.Question{{
			Name: "example.com",
		}},
	}

	l.Add(&AddParams{
		Question: q,
		ClientID: knownClientID,
		ClientIP: net.IP{1, 2, 3, 4},
	})

	// Add the same thing again to test the cache.
	l.Add(&AddParams{
		Question: q,
		ClientID: knownClientID,
		ClientIP: net.IP{1, 2, 3, 4},
	})

	l.Add(&AddParams{
		Question: q,
		ClientID: unknownClientID,
		ClientIP: net.IP{1, 2, 3, 5},
	})

	sp := &searchParams{
		// Add some time to the "current" one to protect against
		// low-resolution timers on some Windows machines.
		//
		// TODO(a.garipov): Use some kind of timeSource interface
		// instead of relying on time.Now() in tests.
		olderThan: time.Now().Add(10 * time.Second),
		limit:     3,
	}
	entries, _ := l.search(ctx, sp)
	assert.Equal(t, 2, findClientCalls)

	require.Len(t, entries, 3)

	assert.Nil(t, entries[0].client)

	gotClient := entries[2].client
	require.NotNil(t, gotClient)

	assert.Equal(t, knownClientName, gotClient.Name)
}

// testQLogLine is a single query log file record used by the search tests.
type testQLogLine struct {
	time   time.Time
	host   string
	reason filtering.Reason
}

// TestQueryLog_Search_ignoredEntryCursor tests that the timestamp reported as
// the pagination cursor is set even for an entry that is skipped because its
// host name is ignored.  Otherwise the cursor is lost, the API returns an
// empty "oldest", and clients stop fetching pages, hiding the matching entries
// older than the ignored one.
func TestQueryLog_Search_ignoredEntryCursor(t *testing.T) {
	const (
		ignoredHost = "ignored.example"
		matchHost   = "match.example"

		// fillersNum is the number of entries newer than the ignored one, so
		// that the scan budget runs out right after the ignored one.
		fillersNum = 3

		// entryFormat is the query log record format.  The timestamps are set
		// explicitly, because the entries must be strictly ordered for the
		// pagination cursor to be unambiguous.
		entryFormat = `{"IP":"192.0.2.1","T":%q,"QH":%q,"QT":"A","QC":"IN",` +
			`"Result":{"Reason":%d},"Elapsed":0}` + "\n"
	)

	engine, err := aghnet.NewIgnoreEngine([]string{ignoredHost}, true)
	require.NoError(t, err)

	l, err := newQueryLog(Config{
		Logger:      testLogger,
		Ignored:     engine,
		BaseDir:     t.TempDir(),
		RotationIvl: timeutil.Day,
		MemSize:     1,
		Enabled:     true,
		FileEnabled: true,
	})
	require.NoError(t, err)

	// The oldest entry is the matching one, followed by the ignored one, so
	// that the only way to reach the match is through the cursor of the
	// ignored entry, which is the newest processed entry of the first page.
	// The fillers don't match the search criterion and are only there to
	// exhaust the scan budget.
	base := time.Date(2020, 2, 18, 19, 36, 35, 0, time.UTC)
	lines := []testQLogLine{{
		host:   matchHost,
		reason: filtering.FilteredBlockList,
		time:   base,
	}, {
		host:   ignoredHost,
		reason: filtering.FilteredBlockList,
		time:   base.Add(time.Second),
	}}
	for i := range fillersNum {
		lines = append(lines, testQLogLine{
			host:   "filler.example",
			reason: filtering.Rewritten,
			time:   base.Add(time.Duration(i+2) * time.Second),
		})
	}

	var b strings.Builder
	for _, line := range lines {
		_, err = fmt.Fprintf(
			&b,
			entryFormat,
			line.time.Format(time.RFC3339Nano),
			line.host,
			line.reason,
		)
		require.NoError(t, err)
	}

	require.NoError(t, os.WriteFile(l.logFile, []byte(b.String()), 0o600))

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	params := newSearchParams()
	params.searchCriteria = []searchCriterion{
		searchReason([]string{filtering.FilteredBlockList.String()}),
	}
	params.maxFileScanEntries = fillersNum + 1
	params.limit = 10

	entries, oldest := l.search(ctx, params)
	assert.Empty(t, entries)
	require.False(t, oldest.IsZero(), "cursor must be set for an ignored entry")

	// Use the returned cursor the way the API clients do to fetch the next
	// page and make sure that the matching entry is found there.
	next := newSearchParams()
	next.searchCriteria = params.searchCriteria
	next.olderThan = oldest
	next.limit = 10

	entries, _ = l.search(ctx, next)
	require.Len(t, entries, 1)
	assert.Equal(t, matchHost, entries[0].QHost)
}
