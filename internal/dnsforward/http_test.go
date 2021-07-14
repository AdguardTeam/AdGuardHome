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
	t.Cleanup(func() {
		require.NoError(t, f.Close())
	})

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

	require.Nil(t, s.Start())
	t.Cleanup(func() {
		require.Nil(t, s.Stop())
	})

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
	assert.Nil(t, err)
	t.Cleanup(func() {
		assert.Nil(t, s.Stop())
	})

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
		wantSet: `wrong upstreams specification: address !!!: ` +
			`missing port in address`,
	}, {
		name: "bootstraps_bad",
		wantSet: `a can not be used as bootstrap dns cause: ` +
			`invalid bootstrap server address: ` +
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
			t.Cleanup(func() {
				s.conf = defaultConf
			})

			rBody := io.NopCloser(bytes.NewReader(caseData.Req))
			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "http://example.com", rBody)
			require.Nil(t, err)

			s.handleSetConfig(w, r)
			assert.Equal(t, tc.wantSet, strings.TrimSuffix(w.Body.String(), "\n"))
			w.Body.Reset()

			s.handleGetConfig(w, nil)
			assert.JSONEq(t, string(caseData.Want), w.Body.String())
			w.Body.Reset()
		})
	}
}

// TODO(a.garipov): Rewrite to check the actual error messages.
func TestValidateUpstream(t *testing.T) {
	testCases := []struct {
		name     string
		upstream string
		valid    bool
		wantDef  bool
	}{{
		name:     "invalid",
		upstream: "1.2.3.4.5",
		valid:    false,
		wantDef:  false,
	}, {
		name:     "invalid",
		upstream: "123.3.7m",
		valid:    false,
		wantDef:  false,
	}, {
		name:     "invalid",
		upstream: "htttps://google.com/dns-query",
		valid:    false,
		wantDef:  false,
	}, {
		name:     "invalid",
		upstream: "[/host.com]tls://dns.adguard.com",
		valid:    false,
		wantDef:  false,
	}, {
		name:     "invalid",
		upstream: "[host.ru]#",
		valid:    false,
		wantDef:  false,
	}, {
		name:     "valid_default",
		upstream: "1.1.1.1",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid_default",
		upstream: "tls://1.1.1.1",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid_default",
		upstream: "https://dns.adguard.com/dns-query",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid_default",
		upstream: "sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		valid:    true,
		wantDef:  true,
	}, {
		name:     "valid",
		upstream: "[/host.com/]1.1.1.1",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[//]tls://1.1.1.1",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[/www.host.com/]#",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[/host.com/google.com/]8.8.8.8",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "valid",
		upstream: "[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "idna",
		upstream: "[/пример.рф/]8.8.8.8",
		valid:    true,
		wantDef:  false,
	}, {
		name:     "bad_domain",
		upstream: "[/!/]8.8.8.8",
		valid:    false,
		wantDef:  false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defaultUpstream, err := validateUpstream(tc.upstream)
			require.Equal(t, tc.valid, err == nil)
			if tc.valid {
				assert.Equal(t, tc.wantDef, defaultUpstream)
			}
		})
	}
}

func TestValidateUpstreamsSet(t *testing.T) {
	testCases := []struct {
		name    string
		msg     string
		set     []string
		wantNil bool
	}{{
		name:    "empty",
		msg:     "empty upstreams array should be valid",
		set:     nil,
		wantNil: true,
	}, {
		name:    "comment",
		msg:     "comments should not be validated",
		set:     []string{"# comment"},
		wantNil: true,
	}, {
		name: "valid_no_default",
		msg:  "there is no default upstream",
		set: []string{
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
		},
		wantNil: false,
	}, {
		name: "valid_with_default",
		msg:  "upstreams set is valid, but doesn't pass through validation cause: %s",
		set: []string{
			"[/host.com/]1.1.1.1",
			"[//]tls://1.1.1.1",
			"[/www.host.com/]#",
			"[/host.com/google.com/]8.8.8.8",
			"[/host/]sdns://AQMAAAAAAAAAFDE3Ni4xMDMuMTMwLjEzMDo1NDQzINErR_JS3PLCu_iZEIbq95zkSV2LFsigxDIuUso_OQhzIjIuZG5zY3J5cHQuZGVmYXVsdC5uczEuYWRndWFyZC5jb20",
			"8.8.8.8",
		},
		wantNil: true,
	}, {
		name:    "invalid",
		msg:     "there is an invalid upstream in set, but it pass through validation",
		set:     []string{"dhcp://fake.dns"},
		wantNil: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateUpstreams(tc.set)

			assert.Equalf(t, tc.wantNil, err == nil, tc.msg, err)
		})
	}
}
