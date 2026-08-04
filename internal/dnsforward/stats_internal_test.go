package dnsforward

import (
	"net"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testQueryLog is a simple [querylog.QueryLog] implementation for tests.
type testQueryLog struct {
	// QueryLog is embedded here simply to make testQueryLog
	// a [querylog.QueryLog] without actually implementing all methods.
	querylog.QueryLog

	lastParams *querylog.AddParams
}

// Add implements the [querylog.QueryLog] interface for *testQueryLog.
func (l *testQueryLog) Add(p *querylog.AddParams) {
	l.lastParams = p
}

// ShouldLog implements the [querylog.QueryLog] interface for *testQueryLog.
func (l *testQueryLog) ShouldLog(string, uint16, uint16, []string) bool {
	return true
}

// testStats is a simple [stats.Interface] implementation for tests.
type testStats struct {
	// Stats is embedded here simply to make testStats a [stats.Interface]
	// without actually implementing all methods.
	stats.Interface

	// mu protects lastEntry and upsEntries, since an optimistic refresh reports
	// from a background goroutine.
	mu sync.Mutex

	lastEntry *stats.Entry

	upsEntries []*stats.UpstreamEntry

	// onShouldCount, if not nil, decides the verdict of ShouldCount.
	onShouldCount func(host string, qt, cl uint16, ids []string) (ok bool)
}

// UpdateUpstream implements the [stats.Interface] interface for *testStats.
func (l *testStats) UpdateUpstream(e *stats.UpstreamEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.upsEntries = append(l.upsEntries, e)
}

// entry returns the last entry the statistics received, or nil.
func (l *testStats) entry() (e *stats.Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.lastEntry
}

// upstreamEntries returns a copy of the collected upstream entries.
func (l *testStats) upstreamEntries() (entries []*stats.UpstreamEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.upsEntries)
}

// Update implements the [stats.Interface] interface for *testStats.
func (l *testStats) Update(e *stats.Entry) {
	if e.Domain == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.lastEntry = e
}

// ShouldCount implements the [stats.Interface] interface for *testStats.
func (l *testStats) ShouldCount(host string, qt, cl uint16, ids []string) bool {
	if l.onShouldCount != nil {
		return l.onShouldCount(host, qt, cl, ids)
	}

	return true
}

func TestServer_ProcessQueryLogsAndStats(t *testing.T) {
	const domain = "example.com."

	testCases := []struct {
		name           string
		domain         string
		proto          proxy.Proto
		addr           netip.AddrPort
		clientID       string
		wantLogProto   querylog.ClientProto
		wantStatClient string
		wantCode       resultCode
		reason         filtering.Reason
		wantStatResult stats.Result
	}{{
		name:           "success_udp",
		domain:         domain,
		proto:          proxy.ProtoUDP,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_tls_clientid",
		domain:         domain,
		proto:          proxy.ProtoTLS,
		addr:           testClientAddrPort,
		clientID:       "cli42",
		wantLogProto:   querylog.ClientProtoDoT,
		wantStatClient: "cli42",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_tls",
		domain:         domain,
		proto:          proxy.ProtoTLS,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDoT,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_quic",
		domain:         domain,
		proto:          proxy.ProtoQUIC,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDoQ,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_https",
		domain:         domain,
		proto:          proxy.ProtoHTTPS,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDoH,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_dnscrypt",
		domain:         domain,
		proto:          proxy.ProtoDNSCrypt,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDNSCrypt,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_udp_filtered",
		domain:         domain,
		proto:          proxy.ProtoUDP,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredBlockList,
		wantStatResult: stats.RFiltered,
	}, {
		name:           "success_udp_sb",
		domain:         domain,
		proto:          proxy.ProtoUDP,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredSafeBrowsing,
		wantStatResult: stats.RSafeBrowsing,
	}, {
		name:           "success_udp_ss",
		domain:         domain,
		proto:          proxy.ProtoUDP,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredSafeSearch,
		wantStatResult: stats.RSafeSearch,
	}, {
		name:           "success_udp_pc",
		domain:         domain,
		proto:          proxy.ProtoUDP,
		addr:           testClientAddrPort,
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredParental,
		wantStatResult: stats.RParental,
	}, {
		name:           "success_udp_pc_empty_fqdn",
		domain:         ".",
		proto:          proxy.ProtoUDP,
		addr:           netip.MustParseAddrPort("4.3.2.1:1234"),
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "4.3.2.1",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredParental,
		wantStatResult: stats.RParental,
	}}

	ups, err := upstream.AddressToUpstream("1.1.1.1", nil)
	require.NoError(t, err)

	for _, tc := range testCases {
		ql := &testQueryLog{}
		st := &testStats{}
		srv := &Server{
			baseLogger: testLogger,
			logger:     testLogger,
			queryLog:   ql,
			stats:      st,
			anonymizer: aghnet.NewIPMut(nil),
		}
		t.Run(tc.name, func(t *testing.T) {
			req := &dns.Msg{
				Question: []dns.Question{{
					Name: tc.domain,
				}},
			}
			pctx := &proxy.DNSContext{
				Proto:    tc.proto,
				Req:      req,
				Res:      &dns.Msg{},
				Addr:     tc.addr,
				Upstream: ups,
			}
			dctx := &dnsContext{
				proxyCtx:  pctx,
				startTime: time.Now(),
				result: &filtering.Result{
					Reason: tc.reason,
				},
				clientID: tc.clientID,
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			code := srv.processQueryLogsAndStats(ctx, testLogger, dctx)
			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantLogProto, ql.lastParams.ClientProto)
			assert.Equal(t, tc.wantStatClient, st.lastEntry.Client)
			assert.Equal(t, tc.wantStatResult, st.lastEntry.Result)
		})
	}
}

// TestServer_stats_upstreamTimes covers the collection of upstream response
// times end to end, for the two ways it used to be wrong.
func TestServer_stats_upstreamTimes(t *testing.T) {
	const domain = "example.com."

	t.Run("retried_not_counted", func(t *testing.T) {
		// A retried exchange takes at least the whole upstream timeout even
		// though its successful attempt took a millisecond, so counting it as an
		// ordinary response makes the reported time bear no relation to the
		// round-trip time, see
		// https://github.com/AdguardTeam/AdGuardHome/issues/8457.
		const upsTimeout = 1 * time.Second

		for _, mode := range []UpstreamMode{
			UpstreamModeLoadBalance,
			UpstreamModeParallel,
			UpstreamModeFastestAddr,
		} {
			t.Run(string(mode), func(t *testing.T) {
				var dropFirst atomic.Bool
				dropFirst.Store(true)

				st := &testStats{}
				addr := newStatsTestServer(t, st, mode, domain, upsTimeout, &dropFirst, false)

				resp, err := dns.Exchange((&dns.Msg{}).SetQuestion(domain, dns.TypeA), addr)
				require.NoError(t, err)
				require.NotEmpty(t, resp.Answer)

				require.Eventually(t, func() (ok bool) {
					return st.entry() != nil
				}, testTimeout, testTimeout/20)

				assert.Empty(t, st.entry().UpstreamStats)
			})
		}
	})

	t.Run("optimistic_refresh_counted", func(t *testing.T) {
		// The refresh of an expired entry happens in the background, after the
		// client has already been answered from the cache, so it reaches no
		// request handler.  Leaving it out biases the average towards cache
		// misses, see https://github.com/AdguardTeam/AdGuardHome/issues/8435.
		var dropFirst atomic.Bool

		st := &testStats{}
		addr := newStatsTestServer(
			t, st, UpstreamModeLoadBalance, domain, 10*time.Second, &dropFirst, true,
		)

		req := (&dns.Msg{}).SetQuestion(domain, dns.TypeA)

		resp, err := dns.Exchange(req, addr)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Answer)

		// Nothing has been refreshed in the background yet.
		require.Empty(t, st.upstreamEntries())

		// Let the cached entry expire, then hit it again: the answer comes from
		// the cache and the entry is refreshed behind it.
		time.Sleep(testCacheTTL + 100*time.Millisecond)

		resp, err = dns.Exchange(req, addr)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Answer)

		assert.Eventually(t, func() (ok bool) {
			return len(st.upstreamEntries()) == 1
		}, 10*time.Second, 100*time.Millisecond)

		for _, e := range st.upstreamEntries() {
			assert.Equal(t, "example.com", e.Domain)
		}
	})
}

// testCacheTTL is the TTL of the answers of the upstream started by
// [newStatsTestServer], and the maximum TTL of its cache.
const testCacheTTL = 1 * time.Second

// newStatsTestServer starts a server whose only upstream answers domain, and
// drops the very first query while dropFirst is set.  If optimistic is true,
// the cache is enabled and refreshes expired entries in the background.
func newStatsTestServer(
	t *testing.T,
	st stats.Interface,
	mode UpstreamMode,
	domain string,
	upsTimeout time.Duration,
	dropFirst *atomic.Bool,
	optimistic bool,
) (addr string) {
	t.Helper()

	var reqNum atomic.Uint32
	upsAddr := aghtest.StartLocalhostUpstream(t, dns.HandlerFunc(func(
		w dns.ResponseWriter,
		r *dns.Msg,
	) {
		if reqNum.Add(1) == 1 && dropFirst.Load() {
			return
		}

		resp := (&dns.Msg{}).SetReply(r)
		resp.Answer = []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(testCacheTTL.Seconds()),
			},
			A: net.IP{1, 2, 3, 4},
		}}

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})).String()

	conf := Config{
		UpstreamDNS:      []string{upsAddr},
		UpstreamMode:     mode,
		EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
		ClientsContainer: EmptyClientsContainer{},
	}

	if optimistic {
		conf.CacheEnabled = true
		conf.CacheSize = 4096
		conf.CacheMaxTTL = uint32(testCacheTTL.Seconds())
		conf.CacheOptimistic = true
		conf.CacheOptimisticAnswerTTL = timeutil.Duration(time.Minute)
		conf.CacheOptimisticMaxAge = timeutil.Duration(time.Hour)
	}

	s := createTestServer(t, &filtering.Config{
		Logger:            testLogger,
		ProtectionEnabled: true,
		BlockingMode:      filtering.BlockingModeDefault,
	}, ServerConfig{
		UDPListenAddrs:  []*net.UDPAddr{{}},
		TCPListenAddrs:  []*net.TCPAddr{{}},
		TLSConf:         &TLSConfig{},
		Config:          conf,
		UpstreamTimeout: upsTimeout,
		ServePlainDNS:   true,
	}, testTLSConfigProvider)
	s.stats = st

	startDeferStop(t, s)

	return s.dnsProxy.Addr(proxy.ProtoUDP).String()
}
