package dnsforward

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov):  Use the better approach to testdata with a separate
// directory for each test, and a separate file for each subtest.  See the
// [configmigrate] package.

// emptySysResolvers is an empty [SystemResolvers] implementation that always
// returns nil.
type emptySysResolvers struct{}

// Addrs implements the aghnet.SystemResolvers interface for emptySysResolvers.
func (emptySysResolvers) Addrs() (addrs []netip.AddrPort) {
	return nil
}

func loadTestData(t *testing.T, casesFileName string, cases any) {
	t.Helper()

	var f *os.File
	f, err := os.Open(filepath.Join("testdata", casesFileName))
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, f.Close)

	err = json.NewDecoder(f).Decode(cases)
	require.NoError(t, err)
}

const (
	jsonExt = ".json"

	// testBlockedRespTTL is the TTL for blocked responses to use in tests.
	testBlockedRespTTL = 10
)

func TestDNSForwardHTTP_handleGetConfig(t *testing.T) {
	filterConf := &filtering.Config{
		ProtectionEnabled:     true,
		BlockingMode:          filtering.BlockingModeDefault,
		BlockedResponseTTL:    testBlockedRespTTL,
		SafeBrowsingEnabled:   true,
		SafeBrowsingCacheSize: 1000,
		SafeSearchConf:        filtering.SafeSearchConfig{Enabled: true},
		SafeSearchCacheSize:   1000,
		ParentalCacheSize:     1000,
		CacheTime:             30,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{},
		TCPListenAddrs: []*net.TCPAddr{},
		Config: Config{
			UpstreamDNS:            []string{"8.8.8.8:53", "8.8.4.4:53"},
			FallbackDNS:            []string{"9.9.9.10"},
			RatelimitSubnetLenIPv4: 24,
			RatelimitSubnetLenIPv6: 56,
			UpstreamMode:           UpstreamModeLoadBalance,
			EDNSClientSubnet:       &EDNSClientSubnet{Enabled: false},
		},
		ConfigModified: func() {},
		ServePlainDNS:  true,
	}
	s := createTestServer(t, filterConf, forwardConf)
	s.sysResolvers = &emptySysResolvers{}

	require.NoError(t, s.Start())
	testutil.CleanupAndRequireSuccess(t, s.Stop)

	defaultConf := s.conf

	w := httptest.NewRecorder()

	testCases := []struct {
		conf func() ServerConfig
		name string
	}{{
		conf: func() ServerConfig {
			return defaultConf
		},
		name: "all_right",
	}, {
		conf: func() ServerConfig {
			conf := defaultConf
			conf.UpstreamMode = UpstreamModeFastestAddr

			return conf
		},
		name: "fastest_addr",
	}, {
		conf: func() ServerConfig {
			conf := defaultConf
			conf.UpstreamMode = UpstreamModeParallel

			return conf
		},
		name: "parallel",
	}}

	var data map[string]json.RawMessage
	loadTestData(t, t.Name()+jsonExt, &data)

	for _, tc := range testCases {
		caseWant, ok := data[tc.name]
		require.True(t, ok)

		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(w.Body.Reset)

			s.conf = tc.conf()
			s.handleGetConfig(w, nil)

			cType := w.Header().Get(httphdr.ContentType)
			assert.Equal(t, aghhttp.HdrValApplicationJSON, cType)
			assert.JSONEq(t, string(caseWant), w.Body.String())
		})
	}
}

func TestDNSForwardHTTP_handleSetConfig(t *testing.T) {
	filterConf := &filtering.Config{
		ProtectionEnabled:     true,
		BlockingMode:          filtering.BlockingModeDefault,
		BlockedResponseTTL:    testBlockedRespTTL,
		SafeBrowsingEnabled:   true,
		SafeBrowsingCacheSize: 1000,
		SafeSearchConf:        filtering.SafeSearchConfig{Enabled: true},
		SafeSearchCacheSize:   1000,
		ParentalCacheSize:     1000,
		CacheTime:             30,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{},
		TCPListenAddrs: []*net.TCPAddr{},
		Config: Config{
			UpstreamDNS:            []string{"8.8.8.8:53", "8.8.4.4:53"},
			RatelimitSubnetLenIPv4: 24,
			RatelimitSubnetLenIPv6: 56,
			UpstreamMode:           UpstreamModeLoadBalance,
			EDNSClientSubnet:       &EDNSClientSubnet{Enabled: false},
		},
		ConfigModified: func() {},
		ServePlainDNS:  true,
	}
	s := createTestServer(t, filterConf, forwardConf)
	s.sysResolvers = &emptySysResolvers{}

	defaultConf := s.conf

	err := s.Start()
	assert.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, s.Stop)

	w := httptest.NewRecorder()

	testCases := []struct {
		name    string
		wantSet string
	}{{
		name:    "upstream_dns",
		wantSet: "",
	}, {
		name:    "bootstraps",
		wantSet: "",
	}, {
		name:    "blocking_mode_good",
		wantSet: "",
	}, {
		name: "blocking_mode_bad",
		wantSet: "validating dns config: " +
			"blocking_ipv4 must be valid ipv4 on custom_ip blocking_mode",
	}, {
		name:    "ratelimit",
		wantSet: "",
	}, {
		name:    "ratelimit_subnet_len",
		wantSet: "",
	}, {
		name:    "ratelimit_whitelist_not_ip",
		wantSet: `decoding request: ParseAddr("not.ip"): unexpected character (at "not.ip")`,
	}, {
		name:    "edns_cs_enabled",
		wantSet: "",
	}, {
		name:    "edns_cs_use_custom",
		wantSet: "",
	}, {
		name:    "edns_cs_use_custom_bad_ip",
		wantSet: "decoding request: ParseAddr(\"bad.ip\"): unexpected character (at \"bad.ip\")",
	}, {
		name:    "dnssec_enabled",
		wantSet: "",
	}, {
		name:    "cache_size",
		wantSet: "",
	}, {
		name:    "upstream_mode_parallel",
		wantSet: "",
	}, {
		name:    "upstream_mode_fastest_addr",
		wantSet: "",
	}, {
		name: "upstream_dns_bad",
		wantSet: `validating dns config: upstream servers: parsing error at index 0: ` +
			`cannot prepare the upstream: invalid address !!!: bad domain name "!!!": ` +
			`bad top-level domain name label "!!!": bad top-level domain name label rune '!'`,
	}, {
		name: "bootstraps_bad",
		wantSet: `validating dns config: checking bootstrap a: not a bootstrap: ParseAddr("a"): ` +
			`unable to parse IP`,
	}, {
		name:    "cache_bad_ttl",
		wantSet: `validating dns config: cache_ttl_min must be less than or equal to cache_ttl_max`,
	}, {
		name:    "upstream_mode_bad",
		wantSet: `validating dns config: upstream_mode: incorrect value "somethingelse"`,
	}, {
		name:    "local_ptr_upstreams_good",
		wantSet: "",
	}, {
		name: "local_ptr_upstreams_bad",
		wantSet: `validating dns config: private upstream servers: ` +
			`bad arpa domain name "non.arpa": not a reversed ip network`,
	}, {
		name:    "local_ptr_upstreams_null",
		wantSet: "",
	}, {
		name:    "fallbacks",
		wantSet: "",
	}, {
		name:    "blocked_response_ttl",
		wantSet: "",
	}, {
		name:    "multiple_domain_specific_upstreams",
		wantSet: "",
	}}

	var data map[string]struct {
		Req  json.RawMessage `json:"req"`
		Want json.RawMessage `json:"want"`
	}

	testData := t.Name() + jsonExt
	loadTestData(t, testData, &data)

	for _, tc := range testCases {
		// NOTE:  Do not use require.Contains, because the size of the data
		// prevents it from printing a meaningful error message.
		caseData, ok := data[tc.name]
		require.Truef(t, ok, "%q does not contain test data for test case %s", testData, tc.name)

		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				s.dnsFilter.SetBlockingMode(filtering.BlockingModeDefault, netip.Addr{}, netip.Addr{})
				s.conf = defaultConf
				s.conf.Config.EDNSClientSubnet = &EDNSClientSubnet{}
				s.dnsFilter.SetBlockedResponseTTL(testBlockedRespTTL)
			})

			rBody := io.NopCloser(bytes.NewReader(caseData.Req))
			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "http://example.com", rBody)
			require.NoError(t, err)

			s.handleSetConfig(w, r)
			assert.Equal(t, tc.wantSet, strings.TrimSuffix(w.Body.String(), "\n"))
			w.Body.Reset()

			s.handleGetConfig(w, nil)
			assert.JSONEq(t, string(caseData.Want), w.Body.String())
			w.Body.Reset()
		})
	}
}

func TestIsCommentOrEmpty(t *testing.T) {
	for _, tc := range []struct {
		want assert.BoolAssertionFunc
		str  string
	}{{
		want: assert.True,
		str:  "",
	}, {
		want: assert.True,
		str:  "# comment",
	}, {
		want: assert.False,
		str:  "1.2.3.4",
	}} {
		tc.want(t, IsCommentOrEmpty(tc.str))
	}
}

func newLocalUpstreamListener(t *testing.T, port uint16, handler dns.Handler) (real netip.AddrPort) {
	t.Helper()

	startCh := make(chan struct{})
	upsSrv := &dns.Server{
		Addr:              netip.AddrPortFrom(netutil.IPv4Localhost(), port).String(),
		Net:               "tcp",
		Handler:           handler,
		NotifyStartedFunc: func() { close(startCh) },
	}
	go func() {
		err := upsSrv.ListenAndServe()
		require.NoError(testutil.PanicT{}, err)
	}()

	<-startCh
	testutil.CleanupAndRequireSuccess(t, upsSrv.Shutdown)

	return testutil.RequireTypeAssert[*net.TCPAddr](t, upsSrv.Listener.Addr()).AddrPort()
}

func TestServer_HandleTestUpstreamDNS(t *testing.T) {
	hdlr := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		err := w.WriteMsg(new(dns.Msg).SetReply(m))
		require.NoError(testutil.PanicT{}, err)
	})

	ups := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, hdlr).String(),
	}).String()

	const (
		upsTimeout = 100 * time.Millisecond

		hostsFileName = "hosts"
		upstreamHost  = "custom.localhost"
	)

	hostsListener := newLocalUpstreamListener(t, 0, hdlr)
	hostsUps := (&url.URL{
		Scheme: "tcp",
		Host:   netutil.JoinHostPort(upstreamHost, hostsListener.Port()),
	}).String()

	hc, err := aghnet.NewHostsContainer(
		fstest.MapFS{
			hostsFileName: &fstest.MapFile{
				Data: []byte(hostsListener.Addr().String() + " " + upstreamHost),
			},
		},
		&aghtest.FSWatcher{
			OnStart:  func() (_ error) { panic("not implemented") },
			OnEvents: func() (e <-chan struct{}) { return nil },
			OnAdd:    func(_ string) (err error) { return nil },
			OnClose:  func() (err error) { return nil },
		},
		hostsFileName,
	)
	require.NoError(t, err)

	srv := createTestServer(t, &filtering.Config{
		BlockingMode: filtering.BlockingModeDefault,
		EtcHosts:     hc,
	}, ServerConfig{
		UDPListenAddrs:  []*net.UDPAddr{{}},
		TCPListenAddrs:  []*net.TCPAddr{{}},
		UpstreamTimeout: upsTimeout,
		Config: Config{
			UpstreamMode:     UpstreamModeLoadBalance,
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
		},
		ServePlainDNS: true,
	})
	srv.etcHosts = upstream.NewHostsResolver(hc)
	startDeferStop(t, srv)

	testCases := []struct {
		body     map[string]any
		wantResp map[string]any
		name     string
	}{{
		body: map[string]any{
			"upstream_dns": []string{hostsUps},
		},
		wantResp: map[string]any{
			hostsUps: "OK",
		},
		name: "etc_hosts",
	}, {
		body: map[string]any{
			"upstream_dns": []string{ups, "#this.is.comment"},
		},
		wantResp: map[string]any{
			ups: "OK",
		},
		name: "comment_mix",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var reqBody []byte
			reqBody, err = json.Marshal(tc.body)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "", bytes.NewReader(reqBody))
			require.NoError(t, err)

			srv.handleTestUpstreamDNS(w, r)
			require.Equal(t, http.StatusOK, w.Code)

			resp := map[string]any{}
			err = json.NewDecoder(w.Body).Decode(&resp)
			require.NoError(t, err)

			assert.Equal(t, tc.wantResp, resp)
		})
	}

	t.Run("timeout", func(t *testing.T) {
		slowHandler := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
			time.Sleep(upsTimeout * 2)
			writeErr := w.WriteMsg(new(dns.Msg).SetReply(m))
			require.NoError(testutil.PanicT{}, writeErr)
		})
		sleepyUps := (&url.URL{
			Scheme: "tcp",
			Host:   newLocalUpstreamListener(t, 0, slowHandler).String(),
		}).String()

		req := map[string]any{
			"upstream_dns": []string{sleepyUps},
		}

		var reqBody []byte
		reqBody, err = json.Marshal(req)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		var r *http.Request
		r, err = http.NewRequest(http.MethodPost, "", bytes.NewReader(reqBody))
		require.NoError(t, err)

		srv.handleTestUpstreamDNS(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		resp := map[string]any{}
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)

		require.Contains(t, resp, sleepyUps)
		require.IsType(t, "", resp[sleepyUps])
		sleepyRes, _ := resp[sleepyUps].(string)

		// TODO(e.burkov):  Improve the format of an error in dnsproxy.
		assert.True(t, strings.HasSuffix(sleepyRes, "i/o timeout"))
	})
}
