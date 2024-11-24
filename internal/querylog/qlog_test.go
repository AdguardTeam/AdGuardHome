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

// TestQueryLog tests adding and loading (with filtering) entries from disk and
// memory.
func TestQueryLog(t *testing.T) {
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
	addEntry(l, "example.org", net.IPv4(1, 1, 1, 1), net.IPv4(2, 2, 2, 1))
	// Write to disk (first file).
	require.NoError(t, l.flushLogBuffer(ctx))

	// Start writing to the second file.
	require.NoError(t, l.rotate(ctx))

	// Add disk entries.
	addEntry(l, "example.org", net.IPv4(1, 1, 1, 2), net.IPv4(2, 2, 2, 2))
	// Write to disk.
	require.NoError(t, l.flushLogBuffer(ctx))

	// Add memory entries.
	addEntry(l, "test.example.org", net.IPv4(1, 1, 1, 3), net.IPv4(2, 2, 2, 3))
	addEntry(l, "example.com", net.IPv4(1, 1, 1, 4), net.IPv4(2, 2, 2, 4))
	addEntry(l, "", net.IPv4(1, 1, 1, 5), net.IPv4(2, 2, 2, 5))

	type tcAssertion struct {
		host   string
		answer net.IP
		client net.IP
		num    int
	}

	testCases := []struct {
		name string
		sCr  []searchCriterion
		want []tcAssertion
	}{{
		name: "all",
		sCr:  []searchCriterion{},
		want: []tcAssertion{
			{num: 0, host: ".", answer: net.IPv4(1, 1, 1, 5), client: net.IPv4(2, 2, 2, 5)},
			{num: 1, host: "example.com", answer: net.IPv4(1, 1, 1, 4), client: net.IPv4(2, 2, 2, 4)},
			{num: 2, host: "test.example.org", answer: net.IPv4(1, 1, 1, 3), client: net.IPv4(2, 2, 2, 3)},
			{num: 3, host: "example.org", answer: net.IPv4(1, 1, 1, 2), client: net.IPv4(2, 2, 2, 2)},
			{num: 4, host: "example.org", answer: net.IPv4(1, 1, 1, 1), client: net.IPv4(2, 2, 2, 1)},
		},
	}, {
		name: "by_domain_strict",
		sCr: []searchCriterion{{
			criterionType: ctTerm,
			strict:        true,
			value:         "TEST.example.org",
		}},
		want: []tcAssertion{{
			num: 0, host: "test.example.org", answer: net.IPv4(1, 1, 1, 3), client: net.IPv4(2, 2, 2, 3),
		}},
	}, {
		name: "by_domain_non-strict",
		sCr: []searchCriterion{{
			criterionType: ctTerm,
			strict:        false,
			value:         "example.ORG",
		}},
		want: []tcAssertion{
			{num: 0, host: "test.example.org", answer: net.IPv4(1, 1, 1, 3), client: net.IPv4(2, 2, 2, 3)},
			{num: 1, host: "example.org", answer: net.IPv4(1, 1, 1, 2), client: net.IPv4(2, 2, 2, 2)},
			{num: 2, host: "example.org", answer: net.IPv4(1, 1, 1, 1), client: net.IPv4(2, 2, 2, 1)},
		},
	}, {
		name: "by_client_ip_strict",
		sCr: []searchCriterion{{
			criterionType: ctTerm,
			strict:        true,
			value:         "2.2.2.2",
		}},
		want: []tcAssertion{{
			num: 0, host: "example.org", answer: net.IPv4(1, 1, 1, 2), client: net.IPv4(2, 2, 2, 2),
		}},
	}, {
		name: "by_client_ip_non-strict",
		sCr: []searchCriterion{{
			criterionType: ctTerm,
			strict:        false,
			value:         "2.2.2",
		}},
		want: []tcAssertion{
			{num: 0, host: ".", answer: net.IPv4(1, 1, 1, 5), client: net.IPv4(2, 2, 2, 5)},
			{num: 1, host: "example.com", answer: net.IPv4(1, 1, 1, 4), client: net.IPv4(2, 2, 2, 4)},
			{num: 2, host: "test.example.org", answer: net.IPv4(1, 1, 1, 3), client: net.IPv4(2, 2, 2, 3)},
			{num: 3, host: "example.org", answer: net.IPv4(1, 1, 1, 2), client: net.IPv4(2, 2, 2, 2)},
			{num: 4, host: "example.org", answer: net.IPv4(1, 1, 1, 1), client: net.IPv4(2, 2, 2, 1)},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := newSearchParams()
			params.searchCriteria = tc.sCr

			entries, _ := l.search(ctx, params)
			require.Len(t, entries, len(tc.want))

			for _, want := range tc.want {
				assertLogEntry(t, entries[want.num], want.host, want.answer, want.client)
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

	engine, err := aghnet.NewIgnoreEngine(ignored)
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

func assertLogEntry(t *testing.T, entry *logEntry, host string, answer, client net.IP) {
	t.Helper()

	require.NotNil(t, entry)

	assert.Equal(t, host, entry.QHost)
	assert.Equal(t, client, entry.IP)
	assert.Equal(t, "A", entry.QType)
	assert.Equal(t, "IN", entry.QClass)

	msg := &dns.Msg{}
	require.NoError(t, msg.Unpack(entry.Answer))
	require.Len(t, msg.Answer, 1)

	a := testutil.RequireTypeAssert[*dns.A](t, msg.Answer[0])
	assert.Equal(t, answer, a.A.To16())
}
