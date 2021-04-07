package dnsforward

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSForwardHTTTP_handleGetConfig(t *testing.T) {
	filterConf := &dnsfilter.Config{
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
	s := createTestServer(t, filterConf, forwardConf)
	require.Nil(t, s.Start())
	t.Cleanup(func() {
		require.Nil(t, s.Stop())
	})

	defaultConf := s.conf

	w := httptest.NewRecorder()

	testCases := []struct {
		name string
		conf func() ServerConfig
		want string
	}{{
		name: "all_right",
		conf: func() ServerConfig {
			return defaultConf
		},
		want: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name: "fastest_addr",
		conf: func() ServerConfig {
			conf := defaultConf
			conf.FastestAddr = true

			return conf
		},
		want: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"fastest_addr\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name: "parallel",
		conf: func() ServerConfig {
			conf := defaultConf
			conf.AllServers = true

			return conf
		},
		want: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"parallel\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(w.Body.Reset)

			s.conf = tc.conf()
			s.handleGetConfig(w, nil)

			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, tc.want, w.Body.String())
		})
	}
}

func TestDNSForwardHTTTP_handleSetConfig(t *testing.T) {
	filterConf := &dnsfilter.Config{
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
	s := createTestServer(t, filterConf, forwardConf)

	defaultConf := s.conf

	err := s.Start()
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, s.Stop())
	}()

	w := httptest.NewRecorder()

	const defaultConfJSON = "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n"
	testCases := []struct {
		name    string
		req     string
		wantSet string
		wantGet string
	}{{
		name:    "upstream_dns",
		req:     "{\"upstream_dns\":[\"8.8.8.8:77\",\"8.8.4.4:77\"]}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:77\",\"8.8.4.4:77\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "bootstraps",
		req:     "{\"bootstrap_dns\":[\"9.9.9.10\"]}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "blocking_mode_good",
		req:     "{\"blocking_mode\":\"refused\"}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"refused\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "blocking_mode_bad",
		req:     "{\"blocking_mode\":\"custom_ip\"}",
		wantSet: "blocking_mode: incorrect value\n",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "ratelimit",
		req:     "{\"ratelimit\":6}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":6,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "edns_cs_enabled",
		req:     "{\"edns_cs_enabled\":true}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":true,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "dnssec_enabled",
		req:     "{\"dnssec_enabled\":true}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":true,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "cache_size",
		req:     "{\"cache_size\":1024}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"\",\"cache_size\":1024,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "upstream_mode_parallel",
		req:     "{\"upstream_mode\":\"parallel\"}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"parallel\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "upstream_mode_fastest_addr",
		req:     "{\"upstream_mode\":\"fastest_addr\"}",
		wantSet: "",
		wantGet: "{\"upstream_dns\":[\"8.8.8.8:53\",\"8.8.4.4:53\"],\"upstream_dns_file\":\"\",\"bootstrap_dns\":[\"9.9.9.10\",\"149.112.112.10\",\"2620:fe::10\",\"2620:fe::fe:10\"],\"protection_enabled\":true,\"ratelimit\":0,\"blocking_mode\":\"\",\"blocking_ipv4\":\"\",\"blocking_ipv6\":\"\",\"edns_cs_enabled\":false,\"dnssec_enabled\":false,\"disable_ipv6\":false,\"upstream_mode\":\"fastest_addr\",\"cache_size\":0,\"cache_ttl_min\":0,\"cache_ttl_max\":0}\n",
	}, {
		name:    "upstream_dns_bad",
		req:     "{\"upstream_dns\":[\"\"]}",
		wantSet: "wrong upstreams specification: missing port in address\n",
		wantGet: defaultConfJSON,
	}, {
		name:    "bootstraps_bad",
		req:     "{\"bootstrap_dns\":[\"a\"]}",
		wantSet: "a can not be used as bootstrap dns cause: invalid bootstrap server address: Resolver a is not eligible to be a bootstrap DNS server\n",
		wantGet: defaultConfJSON,
	}, {
		name:    "cache_bad_ttl",
		req:     "{\"cache_ttl_min\":1024,\"cache_ttl_max\":512}",
		wantSet: "cache_ttl_min must be less or equal than cache_ttl_max\n",
		wantGet: defaultConfJSON,
	}, {
		name:    "upstream_mode_bad",
		req:     "{\"upstream_mode\":\"somethingelse\"}",
		wantSet: "upstream_mode: incorrect value\n",
		wantGet: defaultConfJSON,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				s.conf = defaultConf
			})

			rBody := ioutil.NopCloser(strings.NewReader(tc.req))
			var r *http.Request
			r, err = http.NewRequest(http.MethodPost, "http://example.com", rBody)
			require.Nil(t, err)

			s.handleSetConfig(w, r)
			assert.Equal(t, tc.wantSet, w.Body.String())
			w.Body.Reset()

			s.handleGetConfig(w, nil)
			assert.Equal(t, tc.wantGet, w.Body.String())
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
