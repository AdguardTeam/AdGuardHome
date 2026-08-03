package dnsforward

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsUpstream_Exchange(t *testing.T) {
	const domain = "example.com."

	req := (&dns.Msg{}).SetQuestion(domain, dns.TypeA)

	testCases := []struct {
		upsErr     error
		req        *dns.Msg
		name       string
		wantDomain string
	}{{
		upsErr:     nil,
		req:        req,
		name:       "success",
		wantDomain: "example.com",
	}, {
		upsErr:     errors.Error("network is unreachable"),
		req:        req,
		name:       "error",
		wantDomain: "",
	}, {
		upsErr:     nil,
		req:        &dns.Msg{},
		name:       "no_question",
		wantDomain: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertExchangeSample(t, tc.req, tc.upsErr, tc.wantDomain)
		})
	}
}

// assertExchangeSample performs a single exchange through a [statsUpstream]
// whose wrapped upstream fails with upsErr, if it is not nil, and asserts that
// the statistics received a sample for wantDomain, or none at all if it is
// empty.
func assertExchangeSample(t *testing.T, req *dns.Msg, upsErr error, wantDomain string) {
	t.Helper()

	st := &testStats{}
	ups := &statsUpstream{
		upstream: aghtest.NewUpstreamMock(func(req *dns.Msg) (resp *dns.Msg, err error) {
			if upsErr != nil {
				return nil, upsErr
			}

			return (&dns.Msg{}).SetReply(req), nil
		}),
		srv: &Server{upstreamStats: st},
	}

	_, err := ups.Exchange(req)
	if upsErr != nil {
		assert.ErrorIs(t, err, upsErr)
	} else {
		assert.NoError(t, err)
	}

	entries := st.upstreamEntries()
	if wantDomain == "" {
		assert.Empty(t, entries)

		return
	}

	require.Len(t, entries, 1)

	assert.Equal(t, wantDomain, entries[0].Domain)
	assert.Equal(t, ups.Address(), entries[0].Address)
}

// TestStatsUpstream_Exchange_timeoutRetry is a regression test for the upstream
// response times being much higher than the actual network latency, see
// https://github.com/AdguardTeam/AdGuardHome/issues/8457.
//
// A plain DNS upstream retries once when an attempt times out, e.g. when a UDP
// datagram is lost, and the retried exchange succeeds.  Its duration is then at
// least the whole upstream timeout, which defaults to ten seconds, so a single
// such sample outweighs a hundred normal ones several times over.
func TestStatsUpstream_Exchange_timeoutRetry(t *testing.T) {
	const domain = "example.com."
	const upsTimeout = 1 * time.Second

	// dropFirst tells the upstream to ignore the very first query, the way a
	// lossy network would.
	var dropFirst atomic.Bool

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
			Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   net.IP{1, 2, 3, 4},
		}}

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})).String()

	testCases := []struct {
		name      string
		dropFirst bool
		wantLen   int
	}{{
		name:      "normal_exchange_counted",
		dropFirst: false,
		wantLen:   1,
	}, {
		name:      "retried_exchange_not_counted",
		dropFirst: true,
		wantLen:   0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqNum.Store(0)
			dropFirst.Store(tc.dropFirst)

			ups, err := upstream.AddressToUpstream(upsAddr, &upstream.Options{
				Logger:  testLogger,
				Timeout: upsTimeout,
			})
			require.NoError(t, err)
			testutil.CleanupAndRequireSuccess(t, ups.Close)

			st := &testStats{}
			wrapped := &statsUpstream{
				upstream: ups,
				srv:      &Server{upstreamStats: st, logger: testLogger},
				timeout:  upsTimeout,
			}

			resp, err := wrapped.Exchange((&dns.Msg{}).SetQuestion(domain, dns.TypeA))
			require.NoError(t, err)
			require.NotEmpty(t, resp.Answer)

			assert.Len(t, st.upstreamEntries(), tc.wantLen)
		})
	}
}

// testUpsTimeout is the upstream timeout used by the wrapper tests.
const testUpsTimeout = 10 * time.Second

func TestServer_WrapUpstreamConfig(t *testing.T) {
	s := &Server{}

	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, s.WrapUpstreamConfig(nil, testUpsTimeout))
	})

	t.Run("all_fields", func(t *testing.T) {
		ups := aghtest.NewUpstreamMock(func(req *dns.Msg) (resp *dns.Msg, err error) {
			panic(testutil.UnexpectedCall(req))
		})
		exclusions := container.NewMapSet("excluded.example.")

		uc := &proxy.UpstreamConfig{
			DomainReservedUpstreams: map[string][]upstream.Upstream{
				"reserved.example.": {ups},
			},
			SpecifiedDomainUpstreams: map[string][]upstream.Upstream{
				"specified.example.": {ups},
			},
			SubdomainExclusions: exclusions,
			Upstreams:           []upstream.Upstream{ups},
		}

		wrapped := s.WrapUpstreamConfig(uc, testUpsTimeout)
		require.NotNil(t, wrapped)

		assert.Same(t, exclusions, wrapped.SubdomainExclusions)

		// The original configuration must stay intact.
		assert.Same(t, ups, uc.Upstreams[0])

		for _, u := range [][]upstream.Upstream{
			wrapped.Upstreams,
			wrapped.DomainReservedUpstreams["reserved.example."],
			wrapped.SpecifiedDomainUpstreams["specified.example."],
		} {
			require.Len(t, u, 1)

			su, ok := u[0].(*statsUpstream)
			require.True(t, ok)

			assert.Same(t, ups, su.upstream)
			assert.Same(t, s, su.srv)
		}
	})

	t.Run("nil_fields", func(t *testing.T) {
		wrapped := s.WrapUpstreamConfig(&proxy.UpstreamConfig{}, testUpsTimeout)
		require.NotNil(t, wrapped)

		assert.Nil(t, wrapped.Upstreams)
		assert.Nil(t, wrapped.DomainReservedUpstreams)
		assert.Nil(t, wrapped.SpecifiedDomainUpstreams)
	})
}

// TestServer_updateStats_optimisticCache is a regression test for the biased
// average upstream response time, see
// https://github.com/AdguardTeam/AdGuardHome/issues/8435.
//
// The response served from the optimistic cache is refreshed in the background,
// and that refresh must be counted in the upstream statistics as well.
func TestServer_updateStats_optimisticCache(t *testing.T) {
	const domain = "example.com."

	// cacheTTL is the TTL that the upstream sets on its answers, and also the
	// maximum TTL of the cache, so that the entry expires quickly.
	const cacheTTL = 1 * time.Second

	upsAddr := aghtest.StartLocalhostUpstream(t, dns.HandlerFunc(func(
		w dns.ResponseWriter,
		r *dns.Msg,
	) {
		resp := (&dns.Msg{}).SetReply(r)
		resp.Answer = []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(cacheTTL.Seconds()),
			},
			A: net.IP{1, 2, 3, 4},
		}}

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})).String()

	st := &testStats{}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		TLSConf:        &TLSConfig{},
		Config: Config{
			UpstreamDNS:              []string{upsAddr},
			UpstreamMode:             UpstreamModeLoadBalance,
			CacheEnabled:             true,
			CacheSize:                4096,
			CacheMaxTTL:              uint32(cacheTTL.Seconds()),
			CacheOptimistic:          true,
			CacheOptimisticAnswerTTL: timeutil.Duration(time.Minute),
			CacheOptimisticMaxAge:    timeutil.Duration(time.Hour),
			EDNSClientSubnet:         &EDNSClientSubnet{Enabled: false},
			ClientsContainer:         EmptyClientsContainer{},
		},
		ServePlainDNS: true,
	}

	s := createTestServer(t, &filtering.Config{
		Logger:            testLogger,
		ProtectionEnabled: true,
		BlockingMode:      filtering.BlockingModeDefault,
	}, forwardConf, testTLSConfigProvider)
	s.stats, s.upstreamStats = st, st

	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()

	req := (&dns.Msg{}).SetQuestion(domain, dns.TypeA)

	// The first query is a cache miss, so it's resolved by the upstream.
	resp, err := dns.Exchange(req, addr)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Answer)

	assert.Len(t, st.upstreamEntries(), 1)

	// Let the cached entry expire.
	time.Sleep(cacheTTL + 100*time.Millisecond)

	// The second query is an optimistic cache hit, so it's answered right away
	// while the entry is refreshed in the background.
	resp, err = dns.Exchange(req, addr)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Answer)

	// Use a timeout that is generous enough for the background refresh to
	// finish on a loaded machine.
	const refreshTimeout = 10 * time.Second

	assert.Eventually(t, func() (ok bool) {
		return len(st.upstreamEntries()) == 2
	}, refreshTimeout, refreshTimeout/100)

	// NOTE:  Don't assert on QueryDuration, since an exchange with a localhost
	// upstream may well finish within a single tick of a coarse system clock.
	for _, e := range st.upstreamEntries() {
		assert.Equal(t, "example.com", e.Domain)
		assert.Equal(t, upsAddr, e.Address)
	}
}

// type check
var _ stats.Interface = (*testStats)(nil)

// TestServer_updateStats_ignoredClient is a regression test for the statistics
// ignore lists being bypassed by the upstream wrappers.
//
// An upstream exchange carries no client identity, so a wrapper that recorded
// every exchange would count the queries of a client whose statistics are
// ignored, which both contradicts the documented behaviour and biases the very
// averages this collection exists to report.
func TestServer_updateStats_ignoredClient(t *testing.T) {
	const domain = "example.com."

	upsAddr := aghtest.StartLocalhostUpstream(t, dns.HandlerFunc(func(
		w dns.ResponseWriter,
		r *dns.Msg,
	) {
		resp := (&dns.Msg{}).SetReply(r)
		resp.Answer = []dns.RR{&dns.A{
			Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   net.IP{1, 2, 3, 4},
		}}

		require.NoError(testutil.PanicT{}, w.WriteMsg(resp))
	})).String()

	var countClient atomic.Bool
	st := &testStats{
		onShouldCount: func(_ string, _, _ uint16, _ []string) (ok bool) {
			return countClient.Load()
		},
	}

	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		TLSConf:        &TLSConfig{},
		Config: Config{
			UpstreamDNS:      []string{upsAddr},
			UpstreamMode:     UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
			ClientsContainer: EmptyClientsContainer{},
		},
		ServePlainDNS: true,
	}

	s := createTestServer(t, &filtering.Config{
		Logger:            testLogger,
		ProtectionEnabled: true,
		BlockingMode:      filtering.BlockingModeDefault,
	}, forwardConf, testTLSConfigProvider)
	s.stats, s.upstreamStats = st, st

	startDeferStop(t, s)
	addr := s.dnsProxy.Addr(proxy.ProtoUDP).String()

	t.Run("ignored", func(t *testing.T) {
		countClient.Store(false)

		resp, err := dns.Exchange((&dns.Msg{}).SetQuestion(domain, dns.TypeA), addr)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Answer)

		assert.Empty(t, st.upstreamEntries())
	})

	t.Run("counted", func(t *testing.T) {
		countClient.Store(true)

		// Use a different name, so that the answer isn't served from the cache.
		resp, err := dns.Exchange((&dns.Msg{}).SetQuestion("other."+domain, dns.TypeA), addr)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Len(t, st.upstreamEntries(), 1)
	})
}

// TestServer_WrapUpstreamConfig_race makes sure that wrapping does not read the
// mutable [Server.conf], which is written under serverLock while the client
// upstream manager may wrap a configuration without holding it.
//
// Run with -race.
func TestServer_WrapUpstreamConfig_race(t *testing.T) {
	t.Parallel()

	s := &Server{}
	uc := &proxy.UpstreamConfig{
		Upstreams: []upstream.Upstream{
			aghtest.NewUpstreamMock(func(req *dns.Msg) (resp *dns.Msg, err error) {
				panic(testutil.UnexpectedCall(req))
			}),
		},
	}

	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(2)

	// A /control/dns_config update writing the timeout under the lock.
	go func() {
		defer wg.Done()

		for i := range iterations {
			s.serverLock.Lock()
			s.conf.UpstreamTimeout = time.Duration(i+1) * time.Second
			s.serverLock.Unlock()
		}
	}()

	// The client upstream manager wrapping a configuration without the lock.
	go func() {
		defer wg.Done()

		for range iterations {
			assert.NotNil(t, s.WrapUpstreamConfig(uc, testUpsTimeout))
		}
	}()

	wg.Wait()
}

// TestServer_prepareInternalDNS_privateTimeout is a regression test for the
// private rDNS wrappers carrying the wrong retry threshold.
//
// Those upstreams are constructed with [defaultLocalTimeout], one second, while
// the configured upstream timeout is normally ten.  A wrapper holding the
// configured timeout would compare a one-second retried exchange against ten
// seconds and record it as an ordinary response, reintroducing exactly the
// inflated sample this collection is meant to exclude.
func TestServer_prepareInternalDNS_privateTimeout(t *testing.T) {
	const upsTimeout = 10 * time.Second

	require.NotEqual(t, upsTimeout, defaultLocalTimeout)

	upsAddr := aghtest.StartLocalhostUpstream(t, dns.HandlerFunc(func(
		w dns.ResponseWriter,
		r *dns.Msg,
	) {
		require.NoError(testutil.PanicT{}, w.WriteMsg((&dns.Msg{}).SetReply(r)))
	})).String()

	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{{}},
		TCPListenAddrs: []*net.TCPAddr{{}},
		TLSConf:        &TLSConfig{},
		Config: Config{
			UpstreamDNS:      []string{upsAddr},
			UpstreamMode:     UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
			ClientsContainer: EmptyClientsContainer{},
		},
		LocalPTRResolvers: []string{upsAddr},
		UsePrivateRDNS:    true,
		UpstreamTimeout:   upsTimeout,
		ServePlainDNS:     true,
	}

	s := createTestServer(t, &filtering.Config{
		Logger:            testLogger,
		ProtectionEnabled: true,
		BlockingMode:      filtering.BlockingModeDefault,
	}, forwardConf, testTLSConfigProvider)

	// wrapperTimeout returns the retry threshold stored in the first wrapper of
	// uc.
	wrapperTimeout := func(uc *proxy.UpstreamConfig) (d time.Duration) {
		require.NotNil(t, uc)
		require.NotEmpty(t, uc.Upstreams)

		su, ok := uc.Upstreams[0].(*statsUpstream)
		require.True(t, ok)

		return su.timeout
	}

	assert.Equal(t, upsTimeout, wrapperTimeout(s.conf.UpstreamConfig))
	assert.Equal(t, defaultLocalTimeout, wrapperTimeout(s.conf.PrivateRDNSUpstreamConfig))
}
