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
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSystemResolvers is a mock aghnet.SystemResolvers implementation for
// tests.
type fakeSystemResolvers struct {
	// SystemResolvers is embedded here simply to make *fakeSystemResolvers
	// an aghnet.SystemResolvers without actually implementing all methods.
	aghnet.SystemResolvers
}

// Get implements the aghnet.SystemResolvers interface for *fakeSystemResolvers.
// It always returns nil.
func (fsr *fakeSystemResolvers) Get() (rs []string) {
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

const jsonExt = ".json"

func TestDNSForwardHTTP_handleGetConfig(t *testing.T) {
	filterConf := &filtering.Config{
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
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeDefault,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
			EDNSClientSubnet:  &EDNSClientSubnet{Enabled: false},
		},
		ConfigModified: func() {},
	}
	s := createTestServer(t, filterConf, forwardConf, nil)
	s.sysResolvers = &fakeSystemResolvers{}

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
			conf.FastestAddr = true

			return conf
		},
		name: "fastest_addr",
	}, {
		conf: func() ServerConfig {
			conf := defaultConf
			conf.AllServers = true

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
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			BlockingMode:      BlockingModeDefault,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
			EDNSClientSubnet:  &EDNSClientSubnet{Enabled: false},
		},
		ConfigModified: func() {},
	}
	s := createTestServer(t, filterConf, forwardConf, nil)
	s.sysResolvers = &fakeSystemResolvers{}

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
		name:    "blocking_mode_bad",
		wantSet: "blocking_ipv4 must be set when blocking_mode is custom_ip",
	}, {
		name:    "ratelimit",
		wantSet: "",
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
		name:    "upstream_dns_bad",
		wantSet: `validating upstream servers: validating upstream "!!!": not an ip:port`,
	}, {
		name: "bootstraps_bad",
		wantSet: `checking bootstrap a: invalid address: bootstrap a:53: ` +
			`ParseAddr("a"): unable to parse IP`,
	}, {
		name:    "cache_bad_ttl",
		wantSet: `cache_ttl_min must be less or equal than cache_ttl_max`,
	}, {
		name:    "upstream_mode_bad",
		wantSet: `upstream_mode: incorrect value`,
	}, {
		name:    "local_ptr_upstreams_good",
		wantSet: "",
	}, {
		name: "local_ptr_upstreams_bad",
		wantSet: `validating private upstream servers: checking domain-specific upstreams: ` +
			`bad arpa domain name "non.arpa.": not a reversed ip network`,
	}, {
		name:    "local_ptr_upstreams_null",
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
				s.conf = defaultConf
				s.conf.FilteringConfig.EDNSClientSubnet = &EDNSClientSubnet{}
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

func TestValidateUpstreams(t *testing.T) {
	const sdnsStamp = `sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_J` +
		`S3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczE` +
		`uYWRndWFyZC5jb20`

	testCases := []struct {
		name    string
		wantErr string
		set     []string
	}{{
		name:    "empty",
		wantErr: ``,
		set:     nil,
	}, {
		name:    "comment",
		wantErr: ``,
		set:     []string{"# comment"},
	}, {
		name:    "no_default",
		wantErr: `no default upstreams specified`,
		set: []string{
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]" + sdnsStamp,
		},
	}, {
		name:    "with_default",
		wantErr: ``,
		set: []string{
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]" + sdnsStamp,
			"8.8.8.8",
		},
	}, {
		name:    "invalid",
		wantErr: `validating upstream "dhcp://fake.dns": bad protocol "dhcp"`,
		set:     []string{"dhcp://fake.dns"},
	}, {
		name:    "invalid",
		wantErr: `validating upstream "1.2.3.4.5": not an ip:port`,
		set:     []string{"1.2.3.4.5"},
	}, {
		name:    "invalid",
		wantErr: `validating upstream "123.3.7m": not an ip:port`,
		set:     []string{"123.3.7m"},
	}, {
		name: "invalid",
		wantErr: `bad upstream for domain "[/host.com]tls://dns.adguard.com": ` +
			`missing separator`,
		set: []string{"[/host.com]tls://dns.adguard.com"},
	}, {
		name:    "invalid",
		wantErr: `validating upstream "[host.ru]#": not an ip:port`,
		set:     []string{"[host.ru]#"},
	}, {
		name:    "valid_default",
		wantErr: ``,
		set: []string{
			"1.1.1.1",
			"tls://1.1.1.1",
			"https://dns.adguard.com/dns-query",
			sdnsStamp,
			"udp://dns.google",
			"udp://8.8.8.8",
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]" + sdnsStamp,
			"[/пример.рф/]8.8.8.8",
		},
	}, {
		name: "bad_domain",
		wantErr: `bad upstream for domain "[/!/]8.8.8.8": domain at index 0: ` +
			`bad domain name "!": bad top-level domain name label "!": ` +
			`bad top-level domain name label rune '!'`,
		set: []string{"[/!/]8.8.8.8"},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUpstreams(tc.set)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
		})
	}
}

func TestValidateUpstreamsPrivate(t *testing.T) {
	ss := netutil.SubnetSetFunc(netutil.IsLocallyServed)

	testCases := []struct {
		name    string
		wantErr string
		u       string
	}{{
		name:    "success_address",
		wantErr: ``,
		u:       "[/1.0.0.127.in-addr.arpa/]#",
	}, {
		name:    "success_subnet",
		wantErr: ``,
		u:       "[/127.in-addr.arpa/]#",
	}, {
		name: "not_arpa_subnet",
		wantErr: `checking domain-specific upstreams: ` +
			`bad arpa domain name "hello.world.": not a reversed ip network`,
		u: "[/hello.world/]#",
	}, {
		name: "non-private_arpa_address",
		wantErr: `checking domain-specific upstreams: ` +
			`arpa domain "1.2.3.4.in-addr.arpa." should point to a locally-served network`,
		u: "[/1.2.3.4.in-addr.arpa/]#",
	}, {
		name: "non-private_arpa_subnet",
		wantErr: `checking domain-specific upstreams: ` +
			`arpa domain "128.in-addr.arpa." should point to a locally-served network`,
		u: "[/128.in-addr.arpa/]#",
	}, {
		name: "several_bad",
		wantErr: `checking domain-specific upstreams: 2 errors: ` +
			`"arpa domain \"1.2.3.4.in-addr.arpa.\" should point to a locally-served network", ` +
			`"bad arpa domain name \"non.arpa.\": not a reversed ip network"`,
		u: "[/non.arpa/1.2.3.4.in-addr.arpa/127.in-addr.arpa/]#",
	}, {
		name:    "partial_good",
		wantErr: "",
		u:       "[/a.1.2.3.10.in-addr.arpa/a.10.in-addr.arpa/]#",
	}}

	for _, tc := range testCases {
		set := []string{"192.168.0.1", tc.u}

		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUpstreamsPrivate(set, ss)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
		})
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
	goodHandler := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		err := w.WriteMsg(new(dns.Msg).SetReply(m))
		require.NoError(testutil.PanicT{}, err)
	})
	badHandler := dns.HandlerFunc(func(w dns.ResponseWriter, _ *dns.Msg) {
		err := w.WriteMsg(new(dns.Msg))
		require.NoError(testutil.PanicT{}, err)
	})

	goodUps := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, goodHandler).String(),
	}).String()
	badUps := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, badHandler).String(),
	}).String()

	const (
		upsTimeout = 100 * time.Millisecond

		hostsFileName = "hosts"
		upstreamHost  = "custom.localhost"
	)

	hostsListener := newLocalUpstreamListener(t, 0, goodHandler)
	hostsUps := (&url.URL{
		Scheme: "tcp",
		Host:   netutil.JoinHostPort(upstreamHost, int(hostsListener.Port())),
	}).String()

	hc, err := aghnet.NewHostsContainer(
		filtering.SysHostsListID,
		fstest.MapFS{
			hostsFileName: &fstest.MapFile{
				Data: []byte(hostsListener.Addr().String() + " " + upstreamHost),
			},
		},
		&aghtest.FSWatcher{
			OnEvents: func() (e <-chan struct{}) { return nil },
			OnAdd:    func(_ string) (err error) { return nil },
			OnClose:  func() (err error) { return nil },
		},
		hostsFileName,
	)
	require.NoError(t, err)

	srv := createTestServer(t, &filtering.Config{
		EtcHosts: hc,
	}, ServerConfig{
		UDPListenAddrs:  []*net.UDPAddr{{}},
		TCPListenAddrs:  []*net.TCPAddr{{}},
		UpstreamTimeout: upsTimeout,
		FilteringConfig: FilteringConfig{
			EDNSClientSubnet: &EDNSClientSubnet{Enabled: false},
		},
	}, nil)
	startDeferStop(t, srv)

	testCases := []struct {
		body     map[string]any
		wantResp map[string]any
		name     string
	}{{
		body: map[string]any{
			"upstream_dns": []string{goodUps},
		},
		wantResp: map[string]any{
			goodUps: "OK",
		},
		name: "success",
	}, {
		body: map[string]any{
			"upstream_dns": []string{badUps},
		},
		wantResp: map[string]any{
			badUps: `couldn't communicate with upstream: exchanging with ` +
				badUps + ` over tcp: dns: id mismatch`,
		},
		name: "broken",
	}, {
		body: map[string]any{
			"upstream_dns": []string{goodUps, badUps},
		},
		wantResp: map[string]any{
			goodUps: "OK",
			badUps: `couldn't communicate with upstream: exchanging with ` +
				badUps + ` over tcp: dns: id mismatch`,
		},
		name: "both",
	}, {
		body: map[string]any{
			"upstream_dns": []string{"[/domain.example/]" + badUps},
		},
		wantResp: map[string]any{
			"[/domain.example/]" + badUps: `WARNING: couldn't communicate ` +
				`with upstream: exchanging with ` + badUps + ` over tcp: ` +
				`dns: id mismatch`,
		},
		name: "domain_specific_error",
	}, {
		body: map[string]any{
			"upstream_dns": []string{hostsUps},
		},
		wantResp: map[string]any{
			hostsUps: "OK",
		},
		name: "etc_hosts",
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
