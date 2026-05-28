package querylog

import (
	"fmt"
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchCriTerm is a helper function for tests that constructs a search
// criterion of type [ctTerm].
func searchCriTerm(val string, isStrict bool) (c searchCriterion) {
	return searchCriterion{
		value:         val,
		criterionType: ctTerm,
		strict:        isStrict,
	}
}

// TestQueryLog tests adding and loading (with filtering) entries from disk and
// memory.
func TestQueryLog(t *testing.T) {
	hosts := []string{
		"example.org",
		"example.org",
		"test.example.org",
		"example.com",
		"",
	}

	entries := make([]*logEntry, len(hosts))
	for i, h := range hosts {
		n := byte(i + 1)
		entries[i] = &logEntry{
			QHost:  h,
			Answer: net.IPv4(192, 0, 2, n),
			IP:     net.IPv4(203, 0, 113, n),
		}
	}

	entry1, entry2, entry3, entry4 := entries[0], entries[1], entries[2], entries[3]

	entryRoot := entries[4]
	entryRootWant := &logEntry{QHost: ".", Answer: entryRoot.Answer, IP: entryRoot.IP}

	l, err := newQueryLog(Config{
		Logger:      slogutil.NewDiscardLogger(),
		Enabled:     true,
		FileEnabled: true,
		RotationIvl: timeutil.Day,
		MemSize:     100,
		BaseDir:     t.TempDir(),
	})
	require.NoError(t, err)

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	// Add disk entries.
	addEntry(l, entry1.QHost, entry1.Answer, entry1.IP)
	// Write to disk (first file).
	require.NoError(t, l.flushLogBuffer(ctx))

	// Start writing to the second file.
	require.NoError(t, l.rotate(ctx))

	// Add disk entries.
	addEntry(l, entry2.QHost, entry2.Answer, entry2.IP)
	// Write to disk.
	require.NoError(t, l.flushLogBuffer(ctx))

	// Add memory entries.
	addEntry(l, entry3.QHost, entry3.Answer, entry3.IP)
	addEntry(l, entry4.QHost, entry4.Answer, entry4.IP)
	addEntry(l, entryRoot.QHost, entryRoot.Answer, entryRoot.IP)

	testCases := []struct {
		name string
		sCr  []searchCriterion
		want []*logEntry
	}{{
		name: "all",
		sCr:  []searchCriterion{},
		want: []*logEntry{entryRootWant, entry4, entry3, entry2, entry1},
	}, {
		name: "by_domain_strict",
		sCr:  []searchCriterion{searchCriTerm("TEST.example.org", true)},
		want: []*logEntry{entry3},
	}, {
		name: "by_domain_non-strict",
		sCr:  []searchCriterion{searchCriTerm("example.ORG", false)},
		want: []*logEntry{entry3, entry2, entry1},
	}, {
		name: "by_client_ip_strict",
		sCr:  []searchCriterion{searchCriTerm(entry2.IP.String(), true)},
		want: []*logEntry{entry2},
	}, {
		name: "by_client_ip_non-strict",
		sCr:  []searchCriterion{searchCriTerm("203.0.113", false)},
		want: []*logEntry{entryRootWant, entry4, entry3, entry2, entry1},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := newSearchParams()
			params.searchCriteria = tc.sCr

			entries, _ = l.search(ctx, params)
			require.Len(t, entries, len(tc.want))

			for i, want := range tc.want {
				assertLogEntry(t, entries[i], want.QHost, want.Answer, want.IP)
			}
		})
	}
}

func TestQueryLogOffsetLimit(t *testing.T) {
	l, err := newQueryLog(Config{
		Logger:      slogutil.NewDiscardLogger(),
		Enabled:     true,
		RotationIvl: timeutil.Day,
		MemSize:     100,
		BaseDir:     t.TempDir(),
	})
	require.NoError(t, err)

	const (
		entNum           = 10
		firstPageDomain  = "first.example.org"
		secondPageDomain = "second.example.org"
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	// Add entries to the log.
	for range entNum {
		addEntry(l, secondPageDomain, net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))
	}
	// Write them to the first file.
	require.NoError(t, l.flushLogBuffer(ctx))

	// Add more to the in-memory part of log.
	for range entNum {
		addEntry(l, firstPageDomain, net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))
	}

	params := newSearchParams()

	testCases := []struct {
		name    string
		want    string
		wantLen int
		offset  int
		limit   int
	}{{
		name:    "page_1",
		want:    firstPageDomain,
		wantLen: 10,
		offset:  0,
		limit:   10,
	}, {
		name:    "page_2",
		want:    secondPageDomain,
		wantLen: 10,
		offset:  10,
		limit:   10,
	}, {
		name:    "page_2.5",
		want:    secondPageDomain,
		wantLen: 5,
		offset:  15,
		limit:   10,
	}, {
		name:    "page_3",
		want:    "",
		wantLen: 0,
		offset:  20,
		limit:   10,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params.offset = tc.offset
			params.limit = tc.limit
			entries, _ := l.search(ctx, params)
			require.Len(t, entries, tc.wantLen)

			if tc.wantLen > 0 {
				assert.Equal(t, entries[0].QHost, tc.want)
				assert.Equal(t, entries[tc.wantLen-1].QHost, tc.want)
			}
		})
	}
}

func TestQueryLogMaxFileScanEntries(t *testing.T) {
	l, err := newQueryLog(Config{
		Logger:      slogutil.NewDiscardLogger(),
		Enabled:     true,
		FileEnabled: true,
		RotationIvl: timeutil.Day,
		MemSize:     100,
		BaseDir:     t.TempDir(),
	})
	require.NoError(t, err)

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	const entNum = 10
	// Add entries to the log.
	for range entNum {
		addEntry(l, "example.org", net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))
	}
	// Write them to disk.
	require.NoError(t, l.flushLogBuffer(ctx))

	params := newSearchParams()
	for _, maxFileScanEntries := range []int{5, 0} {
		t.Run(fmt.Sprintf("limit_%d", maxFileScanEntries), func(t *testing.T) {
			params.maxFileScanEntries = maxFileScanEntries
			entries, _ := l.search(ctx, params)
			assert.Len(t, entries, entNum-maxFileScanEntries)
		})
	}
}

func TestQueryLogFileDisabled(t *testing.T) {
	l, err := newQueryLog(Config{
		Logger:      slogutil.NewDiscardLogger(),
		Enabled:     true,
		FileEnabled: false,
		RotationIvl: timeutil.Day,
		MemSize:     2,
		BaseDir:     t.TempDir(),
	})
	require.NoError(t, err)

	addEntry(l, "example1.org", net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))
	addEntry(l, "example2.org", net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))
	// The oldest entry is going to be removed from memory buffer.
	addEntry(l, "example3.org", net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))

	params := newSearchParams()
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	ll, _ := l.search(ctx, params)
	require.Len(t, ll, 2)

	assert.Equal(t, "example3.org", ll[0].QHost)
	assert.Equal(t, "example2.org", ll[1].QHost)
}

func TestQueryLogShouldLog(t *testing.T) {
	const (
		ignored1        = "ignor.ed"
		ignored2        = "ignored.to"
		ignoredWildcard = "*.ignored.com"
		ignoredRoot     = "|.^"
	)

	ignored := []string{
		ignored1,
		ignored2,
		ignoredWildcard,
		ignoredRoot,
	}

	engine, err := aghnet.NewIgnoreEngine(ignored, true)
	require.NoError(t, err)

	findClient := func(ids []string) (c *Client, err error) {
		log := ids[0] == "no_log"

		return &Client{IgnoreQueryLog: log}, nil
	}

	l, err := newQueryLog(Config{
		Ignored:     engine,
		Enabled:     true,
		RotationIvl: timeutil.Day,
		MemSize:     100,
		BaseDir:     t.TempDir(),
		FindClient:  findClient,
	})
	require.NoError(t, err)

	testCases := []struct {
		name    string
		host    string
		ids     []string
		wantLog bool
	}{{
		name:    "log",
		host:    "example.com",
		ids:     []string{"whatever"},
		wantLog: true,
	}, {
		name:    "no_log_ignored_1",
		host:    ignored1,
		ids:     []string{"whatever"},
		wantLog: false,
	}, {
		name:    "no_log_ignored_2",
		host:    ignored2,
		ids:     []string{"whatever"},
		wantLog: false,
	}, {
		name:    "no_log_ignored_wildcard",
		host:    "www.ignored.com",
		ids:     []string{"whatever"},
		wantLog: false,
	}, {
		name:    "no_log_ignored_root",
		host:    ".",
		ids:     []string{"whatever"},
		wantLog: false,
	}, {
		name:    "no_log_client_ignore",
		host:    "example.com",
		ids:     []string{"no_log"},
		wantLog: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := l.ShouldLog(tc.host, dns.TypeA, dns.ClassINET, tc.ids)

			assert.Equal(t, tc.wantLog, res)
		})
	}
}

func addEntry(l *queryLog, host string, answerStr, client net.IP) {
	q := dns.Msg{
		Question: []dns.Question{{
			Name:   host + ".",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	a := dns.Msg{
		Question: q.Question,
		Answer: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: answerStr,
		}},
	}

	res := filtering.Result{
		ServiceName: "SomeService",
		Rules: []*filtering.ResultRule{{
			FilterListID: 1,
			Text:         "SomeRule",
		}},
		Reason:     filtering.Rewritten,
		IsFiltered: true,
	}

	params := &AddParams{
		Question:   &q,
		Answer:     &a,
		OrigAnswer: &a,
		Result:     &res,
		Upstream:   "upstream",
		ClientIP:   client,
	}

	l.Add(params)
}

func assertLogEntry(tb testing.TB, entry *logEntry, host string, answer, client net.IP) {
	tb.Helper()

	require.NotNil(tb, entry)

	assert.Equal(tb, host, entry.QHost)
	assert.Equal(tb, client, entry.IP)
	assert.Equal(tb, "A", entry.QType)
	assert.Equal(tb, "IN", entry.QClass)

	msg := &dns.Msg{}
	require.NoError(tb, msg.Unpack(entry.Answer))
	require.Len(tb, msg.Answer, 1)

	a := testutil.RequireTypeAssert[*dns.A](tb, msg.Answer[0])
	assert.Equal(tb, answer, a.A.To16())
}
