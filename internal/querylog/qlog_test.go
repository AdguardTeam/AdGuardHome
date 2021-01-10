package querylog

import (
	"math/rand"
	"net"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/AdguardTeam/dnsproxy/proxyutil"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Check adding and loading (with filtering) entries from disk and memory
func TestQueryLog(t *testing.T) {
	conf := Config{
		Enabled:     true,
		FileEnabled: true,
		Interval:    1,
		MemSize:     100,
	}
	conf.BaseDir = prepareTestDir()
	defer func() { _ = os.RemoveAll(conf.BaseDir) }()
	l := newQueryLog(conf)

	// add disk entries
	addEntry(l, "example.org", "1.1.1.1", "2.2.2.1")
	// write to disk (first file)
	_ = l.flushLogBuffer(true)
	// start writing to the second file
	_ = l.rotate()
	// add disk entries
	addEntry(l, "example.org", "1.1.1.2", "2.2.2.2")
	// write to disk
	_ = l.flushLogBuffer(true)
	// add memory entries
	addEntry(l, "test.example.org", "1.1.1.3", "2.2.2.3")
	addEntry(l, "example.com", "1.1.1.4", "2.2.2.4")

	// get all entries
	params := newSearchParams()
	entries, _ := l.search(params)
	assert.Equal(t, 4, len(entries))
	assertLogEntry(t, entries[0], "example.com", "1.1.1.4", "2.2.2.4")
	assertLogEntry(t, entries[1], "test.example.org", "1.1.1.3", "2.2.2.3")
	assertLogEntry(t, entries[2], "example.org", "1.1.1.2", "2.2.2.2")
	assertLogEntry(t, entries[3], "example.org", "1.1.1.1", "2.2.2.1")

	// search by domain (strict)
	params = newSearchParams()
	params.searchCriteria = append(params.searchCriteria, searchCriteria{
		criteriaType: ctDomainOrClient,
		strict:       true,
		value:        "TEST.example.org",
	})
	entries, _ = l.search(params)
	assert.Equal(t, 1, len(entries))
	assertLogEntry(t, entries[0], "test.example.org", "1.1.1.3", "2.2.2.3")

	// search by domain (not strict)
	params = newSearchParams()
	params.searchCriteria = append(params.searchCriteria, searchCriteria{
		criteriaType: ctDomainOrClient,
		strict:       false,
		value:        "example.ORG",
	})
	entries, _ = l.search(params)
	assert.Equal(t, 3, len(entries))
	assertLogEntry(t, entries[0], "test.example.org", "1.1.1.3", "2.2.2.3")
	assertLogEntry(t, entries[1], "example.org", "1.1.1.2", "2.2.2.2")
	assertLogEntry(t, entries[2], "example.org", "1.1.1.1", "2.2.2.1")

	// search by client IP (strict)
	params = newSearchParams()
	params.searchCriteria = append(params.searchCriteria, searchCriteria{
		criteriaType: ctDomainOrClient,
		strict:       true,
		value:        "2.2.2.2",
	})
	entries, _ = l.search(params)
	assert.Equal(t, 1, len(entries))
	assertLogEntry(t, entries[0], "example.org", "1.1.1.2", "2.2.2.2")

	// search by client IP (part of)
	params = newSearchParams()
	params.searchCriteria = append(params.searchCriteria, searchCriteria{
		criteriaType: ctDomainOrClient,
		strict:       false,
		value:        "2.2.2",
	})
	entries, _ = l.search(params)
	assert.Equal(t, 4, len(entries))
	assertLogEntry(t, entries[0], "example.com", "1.1.1.4", "2.2.2.4")
	assertLogEntry(t, entries[1], "test.example.org", "1.1.1.3", "2.2.2.3")
	assertLogEntry(t, entries[2], "example.org", "1.1.1.2", "2.2.2.2")
	assertLogEntry(t, entries[3], "example.org", "1.1.1.1", "2.2.2.1")
}

func TestQueryLogOffsetLimit(t *testing.T) {
	conf := Config{
		Enabled:  true,
		Interval: 1,
		MemSize:  100,
	}
	conf.BaseDir = prepareTestDir()
	defer func() { _ = os.RemoveAll(conf.BaseDir) }()
	l := newQueryLog(conf)

	// add 10 entries to the log
	for i := 0; i < 10; i++ {
		addEntry(l, "second.example.org", "1.1.1.1", "2.2.2.1")
	}
	// write them to disk (first file)
	_ = l.flushLogBuffer(true)
	// add 10 more entries to the log (memory)
	for i := 0; i < 10; i++ {
		addEntry(l, "first.example.org", "1.1.1.1", "2.2.2.1")
	}

	// First page
	params := newSearchParams()
	params.offset = 0
	params.limit = 10
	entries, _ := l.search(params)
	assert.Equal(t, 10, len(entries))
	assert.Equal(t, entries[0].QHost, "first.example.org")
	assert.Equal(t, entries[9].QHost, "first.example.org")

	// Second page
	params.offset = 10
	params.limit = 10
	entries, _ = l.search(params)
	assert.Equal(t, 10, len(entries))
	assert.Equal(t, entries[0].QHost, "second.example.org")
	assert.Equal(t, entries[9].QHost, "second.example.org")

	// Second and a half page
	params.offset = 15
	params.limit = 10
	entries, _ = l.search(params)
	assert.Equal(t, 5, len(entries))
	assert.Equal(t, entries[0].QHost, "second.example.org")
	assert.Equal(t, entries[4].QHost, "second.example.org")

	// Third page
	params.offset = 20
	params.limit = 10
	entries, _ = l.search(params)
	assert.Equal(t, 0, len(entries))
}

func TestQueryLogMaxFileScanEntries(t *testing.T) {
	conf := Config{
		Enabled:     true,
		FileEnabled: true,
		Interval:    1,
		MemSize:     100,
	}
	conf.BaseDir = prepareTestDir()
	defer func() { _ = os.RemoveAll(conf.BaseDir) }()
	l := newQueryLog(conf)

	// add 10 entries to the log
	for i := 0; i < 10; i++ {
		addEntry(l, "example.org", "1.1.1.1", "2.2.2.1")
	}
	// write them to disk (first file)
	_ = l.flushLogBuffer(true)

	params := newSearchParams()
	params.maxFileScanEntries = 5 // do not scan more than 5 records
	entries, _ := l.search(params)
	assert.Equal(t, 5, len(entries))

	params.maxFileScanEntries = 0 // disable the limit
	entries, _ = l.search(params)
	assert.Equal(t, 10, len(entries))
}

func TestQueryLogFileDisabled(t *testing.T) {
	conf := Config{
		Enabled:     true,
		FileEnabled: false,
		Interval:    1,
		MemSize:     2,
	}
	conf.BaseDir = prepareTestDir()
	defer func() { _ = os.RemoveAll(conf.BaseDir) }()
	l := newQueryLog(conf)

	addEntry(l, "example1.org", "1.1.1.1", "2.2.2.1")
	addEntry(l, "example2.org", "1.1.1.1", "2.2.2.1")
	addEntry(l, "example3.org", "1.1.1.1", "2.2.2.1")
	// the oldest entry is now removed from mem buffer

	params := newSearchParams()
	ll, _ := l.search(params)
	assert.Equal(t, 2, len(ll))
	assert.Equal(t, "example3.org", ll[0].QHost)
	assert.Equal(t, "example2.org", ll[1].QHost)
}

func addEntry(l *queryLog, host, answerStr, client string) {
	q := dns.Msg{}
	q.Question = append(q.Question, dns.Question{
		Name:   host + ".",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	a := dns.Msg{}
	a.Question = append(a.Question, q.Question[0])
	answer := new(dns.A)
	answer.Hdr = dns.RR_Header{
		Name:   q.Question[0].Name,
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
	}
	answer.A = net.ParseIP(answerStr)
	a.Answer = append(a.Answer, answer)
	res := dnsfilter.Result{
		IsFiltered:  true,
		Reason:      dnsfilter.Rewritten,
		ServiceName: "SomeService",
		Rules: []*dnsfilter.ResultRule{{
			FilterListID: 1,
			Text:         "SomeRule",
		}},
	}
	params := AddParams{
		Question:   &q,
		Answer:     &a,
		OrigAnswer: &a,
		Result:     &res,
		ClientIP:   net.ParseIP(client),
		Upstream:   "upstream",
	}
	l.Add(params)
}

func assertLogEntry(t *testing.T, entry *logEntry, host, answer, client string) bool {
	assert.Equal(t, host, entry.QHost)
	assert.Equal(t, client, entry.IP)
	assert.Equal(t, "A", entry.QType)
	assert.Equal(t, "IN", entry.QClass)

	msg := new(dns.Msg)
	assert.Nil(t, msg.Unpack(entry.Answer))
	assert.Equal(t, 1, len(msg.Answer))
	ip := proxyutil.GetIPFromDNSRecord(msg.Answer[0])
	assert.NotNil(t, ip)
	assert.Equal(t, answer, ip.String())
	return true
}

func testEntries() (entries []*logEntry) {
	rsrc := rand.NewSource(time.Now().UnixNano())
	rgen := rand.New(rsrc)

	entries = make([]*logEntry, 1000)
	for i := range entries {
		min := rgen.Intn(60)
		sec := rgen.Intn(60)
		entries[i] = &logEntry{
			Time: time.Date(2020, 1, 1, 0, min, sec, 0, time.UTC),
		}
	}

	return entries
}

// logEntriesByTimeDesc is a wrapper over []*logEntry for sorting.
//
// NOTE(a.garipov): Weirdly enough, on my machine this gets consistently
// outperformed by sort.Slice, see the benchmark below.  I'm leaving this
// implementation here, in tests, in case we want to make sure it outperforms on
// most machines, but for now this is unused in the actual code.
type logEntriesByTimeDesc []*logEntry

// Len implements the sort.Interface interface for logEntriesByTimeDesc.
func (les logEntriesByTimeDesc) Len() (n int) { return len(les) }

// Less implements the sort.Interface interface for logEntriesByTimeDesc.
func (les logEntriesByTimeDesc) Less(i, j int) (less bool) {
	return les[i].Time.After(les[j].Time)
}

// Swap implements the sort.Interface interface for logEntriesByTimeDesc.
func (les logEntriesByTimeDesc) Swap(i, j int) { les[i], les[j] = les[j], les[i] }

func BenchmarkLogEntry_sort(b *testing.B) {
	b.Run("methods", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			entries := testEntries()
			b.StartTimer()

			sort.Stable(logEntriesByTimeDesc(entries))
		}
	})

	b.Run("reflect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			entries := testEntries()
			b.StartTimer()

			sort.SliceStable(entries, func(i, j int) (less bool) {
				return entries[i].Time.After(entries[j].Time)
			})
		}
	})
}

func TestLogEntriesByTime_sort(t *testing.T) {
	entries := testEntries()
	sort.Sort(logEntriesByTimeDesc(entries))

	for i := 1; i < len(entries); i++ {
		assert.False(t, entries[i].Time.After(entries[i-1].Time),
			"%s %s", entries[i].Time, entries[i-1].Time)
	}
}
