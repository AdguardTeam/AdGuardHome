package dnsforward

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
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

func loadTestData(t *testing.T, casesFileName string, cases interface{}) {
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
		SafeSearchEnabled:     true,
		SafeSearchCacheSize:   1000,
		ParentalCacheSize:     1000,
		CacheTime:             30,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{},
		TCPListenAddrs: []*net.TCPAddr{},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
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

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.JSONEq(t, string(caseWant), w.Body.String())
		})
	}
}

func TestDNSForwardHTTP_handleSetConfig(t *testing.T) {
	filterConf := &filtering.Config{
		SafeBrowsingEnabled:   true,
		SafeBrowsingCacheSize: 1000,
		SafeSearchEnabled:     true,
		SafeSearchCacheSize:   1000,
		ParentalCacheSize:     1000,
		CacheTime:             30,
	}
	forwardConf := ServerConfig{
		UDPListenAddrs: []*net.UDPAddr{},
		TCPListenAddrs: []*net.TCPAddr{},
		FilteringConfig: FilteringConfig{
			ProtectionEnabled: true,
			UpstreamDNS:       []string{"8.8.8.8:53", "8.8.4.4:53"},
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
		wantSet: "blocking_mode: incorrect value",
	}, {
		name:    "ratelimit",
		wantSet: "",
	}, {
		name:    "edns_cs_enabled",
		wantSet: "",
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
		wantSet: `validating upstream servers: bad ipport address "!!!": ` +
			`address !!!: missing port in address`,
	}, {
		name: "bootstraps_bad",
		wantSet: `checking bootstrap a: invalid address: ` +
			`Resolver a is not eligible to be a bootstrap DNS server`,
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
			`bad arpa domain name "non.arpa": not a reversed ip network`,
	}, {
		name:    "local_ptr_upstreams_null",
		wantSet: "",
	}}

	var data map[string]struct {
		Req  json.RawMessage `json:"req"`
		Want json.RawMessage `json:"want"`
	}
	loadTestData(t, t.Name()+jsonExt, &data)

	for _, tc := range testCases {
		caseData, ok := data[tc.name]
		require.True(t, ok)

		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { s.conf = defaultConf })

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

func TestValidateUpstream(t *testing.T) {
	testCases := []struct {
		wantDef  assert.BoolAssertionFunc
		name     string
		upstream string
		wantErr  string
	}{{
		wantDef:  assert.True,
		name:     "invalid",
		upstream: "1.2.3.4.5",
		wantErr:  `bad ipport address "1.2.3.4.5": address 1.2.3.4.5: missing port in address`,
	}, {
		wantDef:  assert.True,
		name:     "invalid",
		upstream: "123.3.7m",
		wantErr:  `bad ipport address "123.3.7m": address 123.3.7m: missing port in address`,
	}, {
		wantDef:  assert.True,
		name:     "invalid",
		upstream: "htttps://google.com/dns-query",
		wantErr:  `wrong protocol`,
	}, {
		wantDef:  assert.True,
		name:     "invalid",
		upstream: "[/host.com]tls://dns.adguard.com",
		wantErr:  `bad upstream for domain "[/host.com]tls://dns.adguard.com": missing separator`,
	}, {
		wantDef:  assert.True,
		name:     "invalid",
		upstream: "[host.ru]#",
		wantErr:  `bad ipport address "[host.ru]#": address [host.ru]#: missing port in address`,
	}, {
		wantDef:  assert.True,
		name:     "valid_default",
		upstream: "1.1.1.1",
		wantErr:  ``,
	}, {
		wantDef:  assert.True,
		name:     "valid_default",
		upstream: "tls://1.1.1.1",
		wantErr:  ``,
	}, {
		wantDef:  assert.True,
		name:     "valid_default",
		upstream: "https://dns.adguard.com/dns-query",
		wantErr:  ``,
	}, {
		wantDef:  assert.True,
		name:     "valid_default",
		upstream: "sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		wantErr:  ``,
	}, {
		wantDef:  assert.True,
		name:     "default_udp_host",
		upstream: "udp://dns.google",
	}, {
		wantDef:  assert.True,
		name:     "default_udp_ip",
		upstream: "udp://8.8.8.8",
	}, {
		wantDef:  assert.False,
		name:     "valid",
		upstream: "[/host.com/]1.1.1.1",
		wantErr:  ``,
	}, {
		wantDef:  assert.False,
		name:     "valid",
		upstream: "[//]tls://1.1.1.1",
		wantErr:  ``,
	}, {
		wantDef:  assert.False,
		name:     "valid",
		upstream: "[/www.host.com/]#",
		wantErr:  ``,
	}, {
		wantDef:  assert.False,
		name:     "valid",
		upstream: "[/host.com/google.com/]8.8.8.8",
		wantErr:  ``,
	}, {
		wantDef:  assert.False,
		name:     "valid",
		upstream: "[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		wantErr:  ``,
	}, {
		wantDef:  assert.False,
		name:     "idna",
		upstream: "[/пример.рф/]8.8.8.8",
		wantErr:  ``,
	}, {
		wantDef:  assert.False,
		name:     "bad_domain",
		upstream: "[/!/]8.8.8.8",
		wantErr: `bad upstream for domain "[/!/]8.8.8.8": domain at index 0: ` +
			`bad domain name "!": bad domain name label "!": bad domain name label rune '!'`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpstream, err := validateUpstream(tc.upstream)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
			tc.wantDef(t, defaultUpstream)
		})
	}
}

func TestValidateUpstreams(t *testing.T) {
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
		name:    "valid_no_default",
		wantErr: `no default upstreams specified`,
		set: []string{
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		},
	}, {
		name:    "valid_with_default",
		wantErr: ``,
		set: []string{
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
			"8.8.8.8",
		},
	}, {
		name:    "invalid",
		wantErr: `cannot prepare the upstream dhcp://fake.dns ([]): unsupported url scheme: dhcp`,
		set:     []string{"dhcp://fake.dns"},
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
			`bad arpa domain name "hello.world": not a reversed ip network`,
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
			`"bad arpa domain name \"non.arpa\": not a reversed ip network"`,
		u: "[/non.arpa/1.2.3.4.in-addr.arpa/127.in-addr.arpa/]#",
	}}

	for _, tc := range testCases {
		set := []string{"192.168.0.1", tc.u}

		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUpstreamsPrivate(set, ss)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
		})
	}
}
