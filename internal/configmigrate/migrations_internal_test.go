package configmigrate

import (
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov): Cover all migrations, use a testdata/ dir.

func TestUpgradeSchema1to2(t *testing.T) {
	diskConf := testDiskConf(1)

	m := New(&Config{
		WorkingDir: "",
	})

	err := m.migrateTo2(diskConf)
	require.NoError(t, err)

	require.Equal(t, diskConf["schema_version"], 2)

	_, ok := diskConf["coredns"]
	require.False(t, ok)

	newDNSConf, ok := diskConf["dns"]
	require.True(t, ok)

	oldDNSConf := testDNSConf(1)
	assert.Equal(t, oldDNSConf, newDNSConf)

	oldExcludedEntries := []string{"coredns", "schema_version"}
	newExcludedEntries := []string{"dns", "schema_version"}
	oldDiskConf := testDiskConf(1)
	assertEqualExcept(t, oldDiskConf, diskConf, oldExcludedEntries, newExcludedEntries)
}

func TestUpgradeSchema2to3(t *testing.T) {
	diskConf := testDiskConf(2)

	err := migrateTo3(diskConf)
	require.NoError(t, err)

	require.Equal(t, diskConf["schema_version"], 3)

	dnsMap, ok := diskConf["dns"]
	require.True(t, ok)

	newDNSConf, ok := dnsMap.(yobj)
	require.True(t, ok)

	bootstrapDNS := newDNSConf["bootstrap_dns"]
	switch v := bootstrapDNS.(type) {
	case yarr:
		require.Len(t, v, 1)
		require.Equal(t, "8.8.8.8:53", v[0])
	default:
		t.Fatalf("wrong type for bootstrap dns: %T", v)
	}

	excludedEntries := []string{"bootstrap_dns"}
	oldDNSConf := testDNSConf(2)
	assertEqualExcept(t, oldDNSConf, newDNSConf, excludedEntries, excludedEntries)

	excludedEntries = []string{"dns", "schema_version"}
	oldDiskConf := testDiskConf(2)
	assertEqualExcept(t, oldDiskConf, diskConf, excludedEntries, excludedEntries)
}

func TestUpgradeSchema5to6(t *testing.T) {
	const newSchemaVer = 6

	testCases := []struct {
		in      yobj
		want    yobj
		wantErr string
		name    string
	}{{
		in: yobj{
			"clients": yarr{},
		},
		want: yobj{
			"clients":        yarr{},
			"schema_version": newSchemaVer,
		},
		wantErr: "",
		name:    "no_clients",
	}, {
		in: yobj{
			"clients": yarr{yobj{"ip": "127.0.0.1"}},
		},
		want: yobj{
			"clients": yarr{yobj{
				"ids": yarr{"127.0.0.1"},
				"ip":  "127.0.0.1",
			}},
			"schema_version": newSchemaVer,
		},
		wantErr: "",
		name:    "client_ip",
	}, {
		in: yobj{
			"clients": yarr{yobj{"mac": "mac"}},
		},
		want: yobj{
			"clients": yarr{yobj{
				"ids": yarr{"mac"},
				"mac": "mac",
			}},
			"schema_version": newSchemaVer,
		},
		wantErr: "",
		name:    "client_mac",
	}, {
		in: yobj{
			"clients": yarr{yobj{"ip": "127.0.0.1", "mac": "mac"}},
		},
		want: yobj{
			"clients": yarr{yobj{
				"ids": yarr{"127.0.0.1", "mac"},
				"ip":  "127.0.0.1",
				"mac": "mac",
			}},
			"schema_version": newSchemaVer,
		},
		wantErr: "",
		name:    "client_ip_mac",
	}, {
		in: yobj{
			"clients": yarr{yobj{"ip": 1, "mac": "mac"}},
		},
		want: yobj{
			"clients":        yarr{yobj{"ip": 1, "mac": "mac"}},
			"schema_version": newSchemaVer,
		},
		wantErr: `client at index 0: unexpected type of "ip": int`,
		name:    "inv_client_ip",
	}, {
		in: yobj{
			"clients": yarr{yobj{"ip": "127.0.0.1", "mac": 1}},
		},
		want: yobj{
			"clients":        yarr{yobj{"ip": "127.0.0.1", "mac": 1}},
			"schema_version": newSchemaVer,
		},
		wantErr: `client at index 0: unexpected type of "mac": int`,
		name:    "inv_client_mac",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo6(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema7to8(t *testing.T) {
	const host = "1.2.3.4"
	oldConf := yobj{
		"dns": yobj{
			"bind_host": host,
		},
		"schema_version": 7,
	}

	err := migrateTo8(oldConf)
	require.NoError(t, err)

	require.Equal(t, oldConf["schema_version"], 8)

	dnsVal, ok := oldConf["dns"]
	require.True(t, ok)

	newDNSConf, ok := dnsVal.(yobj)
	require.True(t, ok)

	newBindHosts, ok := newDNSConf["bind_hosts"].(yarr)
	require.True(t, ok)
	require.Len(t, newBindHosts, 1)
	assert.Equal(t, host, newBindHosts[0])
}

func TestUpgradeSchema8to9(t *testing.T) {
	const tld = "foo"

	t.Run("with_autohost_tld", func(t *testing.T) {
		oldConf := yobj{
			"dns": yobj{
				"autohost_tld": tld,
			},
			"schema_version": 8,
		}

		err := migrateTo9(oldConf)
		require.NoError(t, err)

		require.Equal(t, oldConf["schema_version"], 9)

		dnsVal, ok := oldConf["dns"]
		require.True(t, ok)

		newDNSConf, ok := dnsVal.(yobj)
		require.True(t, ok)

		localDomainName, ok := newDNSConf["local_domain_name"].(string)
		require.True(t, ok)

		assert.Equal(t, tld, localDomainName)
	})

	t.Run("without_autohost_tld", func(t *testing.T) {
		oldConf := yobj{
			"dns":            yobj{},
			"schema_version": 8,
		}

		err := migrateTo9(oldConf)
		require.NoError(t, err)

		require.Equal(t, oldConf["schema_version"], 9)

		dnsVal, ok := oldConf["dns"]
		require.True(t, ok)

		newDNSConf, ok := dnsVal.(yobj)
		require.True(t, ok)

		// Should be nil in order to be set to the default value by the
		// following config rewrite.
		_, ok = newDNSConf["local_domain_name"]
		require.False(t, ok)
	})
}

// assertEqualExcept removes entries from configs and compares them.
func assertEqualExcept(t *testing.T, oldConf, newConf yobj, oldKeys, newKeys []string) {
	t.Helper()

	for _, k := range oldKeys {
		delete(oldConf, k)
	}
	for _, k := range newKeys {
		delete(newConf, k)
	}

	assert.Equal(t, oldConf, newConf)
}

func testDiskConf(schemaVersion int) (diskConf yobj) {
	filters := []filtering.FilterYAML{{
		URL:        "https://filters.adtidy.org/android/filters/111_optimized.txt",
		Name:       "Latvian filter",
		RulesCount: 100,
	}, {
		URL:        "https://easylist.to/easylistgermany/easylistgermany.txt",
		Name:       "Germany filter",
		RulesCount: 200,
	}}
	diskConf = yobj{
		"language":       "en",
		"filters":        filters,
		"user_rules":     []string{},
		"schema_version": schemaVersion,
		"bind_host":      "0.0.0.0",
		"bind_port":      80,
		"auth_name":      "name",
		"auth_pass":      "pass",
	}

	dnsConf := testDNSConf(schemaVersion)
	if schemaVersion > 1 {
		diskConf["dns"] = dnsConf
	} else {
		diskConf["coredns"] = dnsConf
	}

	return diskConf
}

// testDNSConf creates a DNS config for test the way gopkg.in/yaml.v3 would
// unmarshal it.  In YAML, keys aren't guaranteed to always only be strings.
func testDNSConf(schemaVersion int) (dnsConf yobj) {
	dnsConf = yobj{
		"port":                 53,
		"blocked_response_ttl": 10,
		"querylog_enabled":     true,
		"ratelimit":            20,
		"bootstrap_dns":        "8.8.8.8:53",
		"parental_sensitivity": 13,
		"ratelimit_whitelist":  []string{},
		"upstream_dns":         []string{"tls://1.1.1.1", "tls://1.0.0.1", "8.8.8.8"},
		"filtering_enabled":    true,
		"refuse_any":           true,
		"parental_enabled":     true,
		"bind_host":            "0.0.0.0",
		"protection_enabled":   true,
		"safesearch_enabled":   true,
		"safebrowsing_enabled": true,
	}

	if schemaVersion > 2 {
		dnsConf["bootstrap_dns"] = []string{"8.8.8.8:53"}
	}

	return dnsConf
}

func TestAddQUICPort(t *testing.T) {
	testCases := []struct {
		name string
		ups  string
		want string
	}{{
		name: "simple_ip",
		ups:  "8.8.8.8",
		want: "8.8.8.8",
	}, {
		name: "url_ipv4",
		ups:  "quic://8.8.8.8",
		want: "quic://8.8.8.8:784",
	}, {
		name: "url_ipv4_with_port",
		ups:  "quic://8.8.8.8:25565",
		want: "quic://8.8.8.8:25565",
	}, {
		name: "url_ipv6",
		ups:  "quic://[::1]",
		want: "quic://[::1]:784",
	}, {
		name: "url_ipv6_invalid",
		ups:  "quic://::1",
		want: "quic://::1",
	}, {
		name: "url_ipv6_with_port",
		ups:  "quic://[::1]:25565",
		want: "quic://[::1]:25565",
	}, {
		name: "url_hostname",
		ups:  "quic://example.com",
		want: "quic://example.com:784",
	}, {
		name: "url_hostname_with_port",
		ups:  "quic://example.com:25565",
		want: "quic://example.com:25565",
	}, {
		name: "url_hostname_with_endpoint",
		ups:  "quic://example.com/some-endpoint",
		want: "quic://example.com:784/some-endpoint",
	}, {
		name: "url_hostname_with_port_endpoint",
		ups:  "quic://example.com:25565/some-endpoint",
		want: "quic://example.com:25565/some-endpoint",
	}, {
		name: "non-quic_proto",
		ups:  "tls://example.com",
		want: "tls://example.com",
	}, {
		name: "comment",
		ups:  "# comment",
		want: "# comment",
	}, {
		name: "blank",
		ups:  "",
		want: "",
	}, {
		name: "with_domain_ip",
		ups:  "[/example.domain/]8.8.8.8",
		want: "[/example.domain/]8.8.8.8",
	}, {
		name: "with_domain_url",
		ups:  "[/example.domain/]quic://example.com",
		want: "[/example.domain/]quic://example.com:784",
	}, {
		name: "invalid_domain",
		ups:  "[/exmaple.domain]quic://example.com",
		want: "[/exmaple.domain]quic://example.com",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			withPort := addQUICPort(tc.ups, 784)

			assert.Equal(t, tc.want, withPort)
		})
	}
}

func TestUpgradeSchema9to10(t *testing.T) {
	const ultimateAns = 42

	testCases := []struct {
		ups     any
		want    any
		wantErr string
		name    string
	}{{
		ups:     yarr{"quic://8.8.8.8"},
		want:    yarr{"quic://8.8.8.8:784"},
		wantErr: "",
		name:    "success",
	}, {
		ups:     ultimateAns,
		want:    nil,
		wantErr: `unexpected type of "upstream_dns": int`,
		name:    "bad_yarr_type",
	}, {
		ups:     yarr{ultimateAns},
		want:    nil,
		wantErr: `unexpected type of upstream field: int`,
		name:    "bad_upstream_type",
	}}

	for _, tc := range testCases {
		conf := yobj{
			"dns": yobj{
				"upstream_dns": tc.ups,
			},
			"schema_version": 9,
		}
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo10(conf)

			if tc.wantErr != "" {
				testutil.AssertErrorMsg(t, tc.wantErr, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, conf["schema_version"], 10)

			dnsVal, ok := conf["dns"]
			require.True(t, ok)

			newDNSConf, ok := dnsVal.(yobj)
			require.True(t, ok)

			fixedUps, ok := newDNSConf["upstream_dns"].(yarr)
			require.True(t, ok)

			assert.Equal(t, tc.want, fixedUps)
		})
	}

	t.Run("no_dns", func(t *testing.T) {
		err := migrateTo10(yobj{})

		assert.NoError(t, err)
	})

	t.Run("bad_dns", func(t *testing.T) {
		err := migrateTo10(yobj{
			"dns": ultimateAns,
		})

		testutil.AssertErrorMsg(t, `unexpected type of "dns": int`, err)
	})
}

func TestUpgradeSchema10to11(t *testing.T) {
	check := func(t *testing.T, conf yobj) {
		rlimit, _ := conf["rlimit_nofile"].(int)

		err := migrateTo11(conf)
		require.NoError(t, err)

		require.Equal(t, conf["schema_version"], 11)

		_, ok := conf["rlimit_nofile"]
		assert.False(t, ok)

		osVal, ok := conf["os"]
		require.True(t, ok)

		newOSConf, ok := osVal.(yobj)
		require.True(t, ok)

		_, ok = newOSConf["group"]
		assert.True(t, ok)

		_, ok = newOSConf["user"]
		assert.True(t, ok)

		rlimitVal, ok := newOSConf["rlimit_nofile"].(int)
		require.True(t, ok)

		assert.Equal(t, rlimit, rlimitVal)
	}

	const rlimit = 42
	t.Run("with_rlimit", func(t *testing.T) {
		conf := yobj{
			"rlimit_nofile":  rlimit,
			"schema_version": 10,
		}
		check(t, conf)
	})

	t.Run("without_rlimit", func(t *testing.T) {
		conf := yobj{
			"schema_version": 10,
		}
		check(t, conf)
	})
}

func TestUpgradeSchema11to12(t *testing.T) {
	testCases := []struct {
		ivl     any
		want    any
		wantErr string
		name    string
	}{{
		ivl:     1,
		want:    timeutil.Duration{Duration: timeutil.Day},
		wantErr: "",
		name:    "success",
	}, {
		ivl:     0.25,
		want:    0,
		wantErr: `unexpected type of "querylog_interval": float64`,
		name:    "fail",
	}}

	for _, tc := range testCases {
		conf := yobj{
			"dns": yobj{
				"querylog_interval": tc.ivl,
			},
			"schema_version": 11,
		}
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo12(conf)

			if tc.wantErr != "" {
				require.Error(t, err)

				assert.Equal(t, tc.wantErr, err.Error())

				return
			}

			require.NoError(t, err)
			require.Equal(t, conf["schema_version"], 12)

			dnsVal, ok := conf["dns"]
			require.True(t, ok)

			var newDNSConf yobj
			newDNSConf, ok = dnsVal.(yobj)
			require.True(t, ok)

			var newIvl timeutil.Duration
			newIvl, ok = newDNSConf["querylog_interval"].(timeutil.Duration)
			require.True(t, ok)

			assert.Equal(t, tc.want, newIvl)
		})
	}

	t.Run("no_dns", func(t *testing.T) {
		err := migrateTo12(yobj{})

		assert.NoError(t, err)
	})

	t.Run("bad_dns", func(t *testing.T) {
		err := migrateTo12(yobj{
			"dns": 0,
		})

		testutil.AssertErrorMsg(t, `unexpected type of "dns": int`, err)
	})

	t.Run("no_field", func(t *testing.T) {
		conf := yobj{
			"dns": yobj{},
		}

		err := migrateTo12(conf)
		require.NoError(t, err)

		dns, ok := conf["dns"]
		require.True(t, ok)

		var dnsVal yobj
		dnsVal, ok = dns.(yobj)
		require.True(t, ok)

		var ivl any
		ivl, ok = dnsVal["querylog_interval"]
		require.True(t, ok)

		var ivlVal timeutil.Duration
		ivlVal, ok = ivl.(timeutil.Duration)
		require.True(t, ok)

		assert.Equal(t, 90*24*time.Hour, ivlVal.Duration)
	})
}

func TestUpgradeSchema12to13(t *testing.T) {
	const newSchemaVer = 13

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in:   yobj{},
		want: yobj{"schema_version": newSchemaVer},
		name: "no_dns",
	}, {
		in: yobj{"dns": yobj{}},
		want: yobj{
			"dns":            yobj{},
			"schema_version": newSchemaVer,
		},
		name: "no_dhcp",
	}, {
		in: yobj{
			"dns": yobj{
				"local_domain_name": "lan",
			},
			"dhcp":           yobj{},
			"schema_version": newSchemaVer - 1,
		},
		want: yobj{
			"dns": yobj{},
			"dhcp": yobj{
				"local_domain_name": "lan",
			},
			"schema_version": newSchemaVer,
		},
		name: "good",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo13(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema13to14(t *testing.T) {
	const newSchemaVer = 14

	testClient := yobj{
		"name":                "agh-client",
		"ids":                 []string{"id1"},
		"use_global_settings": true,
	}

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in: yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
			// The clients field will be added anyway.
			"clients": yobj{
				"persistent": yarr{},
				"runtime_sources": yobj{
					"whois": true,
					"arp":   true,
					"rdns":  false,
					"dhcp":  true,
					"hosts": true,
				},
			},
		},
		name: "no_clients",
	}, {
		in: yobj{
			"clients": yarr{testClient},
		},
		want: yobj{
			"schema_version": newSchemaVer,
			"clients": yobj{
				"persistent": yarr{testClient},
				"runtime_sources": yobj{
					"whois": true,
					"arp":   true,
					"rdns":  false,
					"dhcp":  true,
					"hosts": true,
				},
			},
		},
		name: "no_dns",
	}, {
		in: yobj{
			"clients": yarr{testClient},
			"dns": yobj{
				"resolve_clients": true,
			},
		},
		want: yobj{
			"schema_version": newSchemaVer,
			"clients": yobj{
				"persistent": yarr{testClient},
				"runtime_sources": yobj{
					"whois": true,
					"arp":   true,
					"rdns":  true,
					"dhcp":  true,
					"hosts": true,
				},
			},
			"dns": yobj{},
		},
		name: "good",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo14(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema14to15(t *testing.T) {
	const newSchemaVer = 15

	defaultWantObj := yobj{
		"querylog": map[string]any{
			"enabled":      true,
			"file_enabled": true,
			"interval":     "2160h",
			"size_memory":  1000,
			"ignored":      []any{},
		},
		"dns":            map[string]any{},
		"schema_version": newSchemaVer,
	}

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in: yobj{
			"dns": map[string]any{
				"querylog_enabled":      true,
				"querylog_file_enabled": true,
				"querylog_interval":     "2160h",
				"querylog_size_memory":  1000,
			},
		},
		want: defaultWantObj,
		name: "basic",
	}, {
		in: yobj{
			"dns": map[string]any{},
		},
		want: defaultWantObj,
		name: "default_values",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo15(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema15to16(t *testing.T) {
	const newSchemaVer = 16

	defaultWantObj := yobj{
		"statistics": map[string]any{
			"enabled":  true,
			"interval": 1,
			"ignored":  []any{},
		},
		"dns":            map[string]any{},
		"schema_version": newSchemaVer,
	}

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in: yobj{
			"dns": map[string]any{
				"statistics_interval": 1,
			},
		},
		want: defaultWantObj,
		name: "basic",
	}, {
		in: yobj{
			"dns": map[string]any{},
		},
		want: defaultWantObj,
		name: "default_values",
	}, {
		in: yobj{
			"dns": map[string]any{
				"statistics_interval": 0,
			},
		},
		want: yobj{
			"statistics": map[string]any{
				"enabled":  false,
				"interval": 1,
				"ignored":  []any{},
			},
			"dns":            map[string]any{},
			"schema_version": newSchemaVer,
		},
		name: "stats_disabled",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo16(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema16to17(t *testing.T) {
	const newSchemaVer = 17

	defaultWantObj := yobj{
		"dns": map[string]any{
			"edns_client_subnet": map[string]any{
				"enabled":    false,
				"use_custom": false,
				"custom_ip":  "",
			},
		},
		"schema_version": newSchemaVer,
	}

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in: yobj{
			"dns": map[string]any{
				"edns_client_subnet": false,
			},
		},
		want: defaultWantObj,
		name: "basic",
	}, {
		in: yobj{
			"dns": map[string]any{},
		},
		want: defaultWantObj,
		name: "default_values",
	}, {
		in: yobj{
			"dns": map[string]any{
				"edns_client_subnet": true,
			},
		},
		want: yobj{
			"dns": map[string]any{
				"edns_client_subnet": map[string]any{
					"enabled":    true,
					"use_custom": false,
					"custom_ip":  "",
				},
			},
			"schema_version": newSchemaVer,
		},
		name: "is_true",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo17(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema17to18(t *testing.T) {
	const newSchemaVer = 18

	defaultWantObj := yobj{
		"dns": yobj{
			"safe_search": yobj{
				"enabled":    true,
				"bing":       true,
				"duckduckgo": true,
				"google":     true,
				"pixabay":    true,
				"yandex":     true,
				"youtube":    true,
			},
		},
		"schema_version": newSchemaVer,
	}

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in:   yobj{"dns": yobj{}},
		want: defaultWantObj,
		name: "default_values",
	}, {
		in:   yobj{"dns": yobj{"safesearch_enabled": true}},
		want: defaultWantObj,
		name: "enabled",
	}, {
		in: yobj{"dns": yobj{"safesearch_enabled": false}},
		want: yobj{
			"dns": yobj{
				"safe_search": map[string]any{
					"enabled":    false,
					"bing":       true,
					"duckduckgo": true,
					"google":     true,
					"pixabay":    true,
					"yandex":     true,
					"youtube":    true,
				},
			},
			"schema_version": newSchemaVer,
		},
		name: "disabled",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo18(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema18to19(t *testing.T) {
	const newSchemaVer = 19

	defaultWantObj := yobj{
		"clients": yobj{
			"persistent": yarr{yobj{
				"name": "localhost",
				"safe_search": yobj{
					"enabled":    true,
					"bing":       true,
					"duckduckgo": true,
					"google":     true,
					"pixabay":    true,
					"yandex":     true,
					"youtube":    true,
				},
			}},
		},
		"schema_version": newSchemaVer,
	}

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in: yobj{
			"clients": yobj{},
		},
		want: yobj{
			"clients":        yobj{},
			"schema_version": newSchemaVer,
		},
		name: "no_clients",
	}, {
		in: yobj{
			"clients": yobj{
				"persistent": yarr{yobj{"name": "localhost"}},
			},
		},
		want: defaultWantObj,
		name: "default_values",
	}, {
		in: yobj{
			"clients": yobj{
				"persistent": yarr{yobj{"name": "localhost", "safesearch_enabled": true}},
			},
		},
		want: defaultWantObj,
		name: "enabled",
	}, {
		in: yobj{
			"clients": yobj{
				"persistent": yarr{yobj{"name": "localhost", "safesearch_enabled": false}},
			},
		},
		want: yobj{
			"clients": yobj{"persistent": yarr{yobj{
				"name": "localhost",
				"safe_search": yobj{
					"enabled":    false,
					"bing":       true,
					"duckduckgo": true,
					"google":     true,
					"pixabay":    true,
					"yandex":     true,
					"youtube":    true,
				},
			}}},
			"schema_version": newSchemaVer,
		},
		name: "disabled",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo19(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema19to20(t *testing.T) {
	testCases := []struct {
		ivl     any
		want    any
		wantErr string
		name    string
	}{{
		ivl:     1,
		want:    timeutil.Duration{Duration: timeutil.Day},
		wantErr: "",
		name:    "success",
	}, {
		ivl:     0,
		want:    timeutil.Duration{Duration: timeutil.Day},
		wantErr: "",
		name:    "success",
	}, {
		ivl:     0.25,
		want:    0,
		wantErr: `unexpected type of "interval": float64`,
		name:    "fail",
	}}

	for _, tc := range testCases {
		conf := yobj{
			"statistics": yobj{
				"interval": tc.ivl,
			},
			"schema_version": 19,
		}
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo20(conf)

			if tc.wantErr != "" {
				require.Error(t, err)

				assert.Equal(t, tc.wantErr, err.Error())

				return
			}

			require.NoError(t, err)
			require.Equal(t, conf["schema_version"], 20)

			statsVal, ok := conf["statistics"]
			require.True(t, ok)

			var stats yobj
			stats, ok = statsVal.(yobj)
			require.True(t, ok)

			var newIvl timeutil.Duration
			newIvl, ok = stats["interval"].(timeutil.Duration)
			require.True(t, ok)

			assert.Equal(t, tc.want, newIvl)
		})
	}

	t.Run("no_stats", func(t *testing.T) {
		err := migrateTo20(yobj{})

		assert.NoError(t, err)
	})

	t.Run("bad_stats", func(t *testing.T) {
		err := migrateTo20(yobj{
			"statistics": 0,
		})

		testutil.AssertErrorMsg(t, `unexpected type of "statistics": int`, err)
	})

	t.Run("no_field", func(t *testing.T) {
		conf := yobj{
			"statistics": yobj{},
		}

		err := migrateTo20(conf)
		require.NoError(t, err)

		statsVal, ok := conf["statistics"]
		require.True(t, ok)

		var stats yobj
		stats, ok = statsVal.(yobj)
		require.True(t, ok)

		var ivl any
		ivl, ok = stats["interval"]
		require.True(t, ok)

		var ivlVal timeutil.Duration
		ivlVal, ok = ivl.(timeutil.Duration)
		require.True(t, ok)

		assert.Equal(t, 24*time.Hour, ivlVal.Duration)
	})
}

func TestUpgradeSchema20to21(t *testing.T) {
	const newSchemaVer = 21

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		name: "nothing",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
	}, {
		name: "no_clients",
		in: yobj{
			"dns": yobj{
				"blocked_services": yarr{"ok"},
			},
		},
		want: yobj{
			"dns": yobj{
				"blocked_services": yobj{
					"ids": yarr{"ok"},
					"schedule": yobj{
						"time_zone": "Local",
					},
				},
			},
			"schema_version": newSchemaVer,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo21(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema21to22(t *testing.T) {
	const newSchemaVer = 22

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		in: yobj{
			"clients": yobj{},
		},
		want: yobj{
			"clients":        yobj{},
			"schema_version": newSchemaVer,
		},
		name: "nothing",
	}, {
		in: yobj{
			"clients": yobj{
				"persistent": []any{yobj{"name": "localhost", "blocked_services": yarr{}}},
			},
		},
		want: yobj{
			"clients": yobj{
				"persistent": []any{yobj{
					"name": "localhost",
					"blocked_services": yobj{
						"ids": yarr{},
						"schedule": yobj{
							"time_zone": "Local",
						},
					},
				}},
			},
			"schema_version": newSchemaVer,
		},
		name: "no_services",
	}, {
		in: yobj{
			"clients": yobj{
				"persistent": []any{yobj{"name": "localhost", "blocked_services": yarr{"ok"}}},
			},
		},
		want: yobj{
			"clients": yobj{
				"persistent": []any{yobj{
					"name": "localhost",
					"blocked_services": yobj{
						"ids": yarr{"ok"},
						"schedule": yobj{
							"time_zone": "Local",
						},
					},
				}},
			},
			"schema_version": newSchemaVer,
		},
		name: "services",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo22(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema22to23(t *testing.T) {
	const newSchemaVer = 23

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		name: "empty",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
	}, {
		name: "ok",
		in: yobj{
			"bind_host":       "1.2.3.4",
			"bind_port":       8081,
			"web_session_ttl": 720,
		},
		want: yobj{
			"http": yobj{
				"address":     "1.2.3.4:8081",
				"session_ttl": "720h",
			},
			"schema_version": newSchemaVer,
		},
	}, {
		name: "v6_address",
		in: yobj{
			"bind_host":       "2001:db8::1",
			"bind_port":       8081,
			"web_session_ttl": 720,
		},
		want: yobj{
			"http": yobj{
				"address":     "[2001:db8::1]:8081",
				"session_ttl": "720h",
			},
			"schema_version": newSchemaVer,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo23(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema23to24(t *testing.T) {
	const newSchemaVer = 24

	testCases := []struct {
		in         yobj
		want       yobj
		name       string
		wantErrMsg string
	}{{
		name: "empty",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
		wantErrMsg: "",
	}, {
		name: "ok",
		in: yobj{
			"log_file":        "/test/path.log",
			"log_max_backups": 1,
			"log_max_size":    2,
			"log_max_age":     3,
			"log_compress":    true,
			"log_localtime":   true,
			"verbose":         true,
		},
		want: yobj{
			"log": yobj{
				"file":        "/test/path.log",
				"max_backups": 1,
				"max_size":    2,
				"max_age":     3,
				"compress":    true,
				"local_time":  true,
				"verbose":     true,
			},
			"schema_version": newSchemaVer,
		},
		wantErrMsg: "",
	}, {
		name: "invalid",
		in: yobj{
			"log_file":        "/test/path.log",
			"log_max_backups": 1,
			"log_max_size":    2,
			"log_max_age":     3,
			"log_compress":    "",
			"log_localtime":   true,
			"verbose":         true,
		},
		want: yobj{
			"log_compress":   "",
			"schema_version": newSchemaVer,
		},
		wantErrMsg: `unexpected type of "log_compress": string`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo24(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema24to25(t *testing.T) {
	const newSchemaVer = 25

	testCases := []struct {
		in         yobj
		want       yobj
		name       string
		wantErrMsg string
	}{{
		name: "empty",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
		wantErrMsg: "",
	}, {
		name: "ok",
		in: yobj{
			"http": yobj{
				"address":     "0.0.0.0:3000",
				"session_ttl": "720h",
			},
			"debug_pprof": true,
		},
		want: yobj{
			"http": yobj{
				"address":     "0.0.0.0:3000",
				"session_ttl": "720h",
				"pprof": yobj{
					"enabled": true,
					"port":    6060,
				},
			},
			"schema_version": newSchemaVer,
		},
		wantErrMsg: "",
	}, {
		name: "ok_disabled",
		in: yobj{
			"http": yobj{
				"address":     "0.0.0.0:3000",
				"session_ttl": "720h",
			},
			"debug_pprof": false,
		},
		want: yobj{
			"http": yobj{
				"address":     "0.0.0.0:3000",
				"session_ttl": "720h",
				"pprof": yobj{
					"enabled": false,
					"port":    6060,
				},
			},
			"schema_version": newSchemaVer,
		},
		wantErrMsg: "",
	}, {
		name: "invalid",
		in: yobj{
			"http": yobj{
				"address":     "0.0.0.0:3000",
				"session_ttl": "720h",
			},
			"debug_pprof": 1,
		},
		want: yobj{
			"http": yobj{
				"address":     "0.0.0.0:3000",
				"session_ttl": "720h",
			},
			"debug_pprof":    1,
			"schema_version": newSchemaVer,
		},
		wantErrMsg: `unexpected type of "debug_pprof": int`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo25(tc.in)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema25to26(t *testing.T) {
	const newSchemaVer = 26

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		name: "empty",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
	}, {
		name: "ok",
		in: yobj{
			"dns": yobj{
				"filtering_enabled":       true,
				"filters_update_interval": 24,
				"parental_enabled":        false,
				"safebrowsing_enabled":    false,
				"safebrowsing_cache_size": 1048576,
				"safesearch_cache_size":   1048576,
				"parental_cache_size":     1048576,
				"safe_search": yobj{
					"enabled":    false,
					"bing":       true,
					"duckduckgo": true,
					"google":     true,
					"pixabay":    true,
					"yandex":     true,
					"youtube":    true,
				},
				"rewrites": yarr{},
				"blocked_services": yobj{
					"schedule": yobj{
						"time_zone": "Local",
					},
					"ids": yarr{},
				},
				"protection_enabled":        true,
				"blocking_mode":             "custom_ip",
				"blocking_ipv4":             "1.2.3.4",
				"blocking_ipv6":             "1:2:3::4",
				"blocked_response_ttl":      10,
				"protection_disabled_until": nil,
				"parental_block_host":       "p.dns.adguard.com",
				"safebrowsing_block_host":   "s.dns.adguard.com",
			},
		},
		want: yobj{
			"dns": yobj{},
			"filtering": yobj{
				"filtering_enabled":       true,
				"filters_update_interval": 24,
				"parental_enabled":        false,
				"safebrowsing_enabled":    false,
				"safebrowsing_cache_size": 1048576,
				"safesearch_cache_size":   1048576,
				"parental_cache_size":     1048576,
				"safe_search": yobj{
					"enabled":    false,
					"bing":       true,
					"duckduckgo": true,
					"google":     true,
					"pixabay":    true,
					"yandex":     true,
					"youtube":    true,
				},
				"rewrites": yarr{},
				"blocked_services": yobj{
					"schedule": yobj{
						"time_zone": "Local",
					},
					"ids": yarr{},
				},
				"protection_enabled":        true,
				"blocking_mode":             "custom_ip",
				"blocking_ipv4":             "1.2.3.4",
				"blocking_ipv6":             "1:2:3::4",
				"blocked_response_ttl":      10,
				"protection_disabled_until": nil,
				"parental_block_host":       "p.dns.adguard.com",
				"safebrowsing_block_host":   "s.dns.adguard.com",
			},
			"schema_version": newSchemaVer,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo26(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema26to27(t *testing.T) {
	const newSchemaVer = 27

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		name: "empty",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
	}, {
		name: "single_dot",
		in: yobj{
			"querylog": yobj{
				"ignored": yarr{
					".",
				},
			},
			"statistics": yobj{
				"ignored": yarr{
					".",
				},
			},
		},
		want: yobj{
			"querylog": yobj{
				"ignored": yarr{
					"|.^",
				},
			},
			"statistics": yobj{
				"ignored": yarr{
					"|.^",
				},
			},
			"schema_version": newSchemaVer,
		},
	}, {
		name: "mixed",
		in: yobj{
			"querylog": yobj{
				"ignored": yarr{
					".",
					"example.com",
				},
			},
			"statistics": yobj{
				"ignored": yarr{
					".",
					"example.org",
				},
			},
		},
		want: yobj{
			"querylog": yobj{
				"ignored": yarr{
					"|.^",
					"example.com",
				},
			},
			"statistics": yobj{
				"ignored": yarr{
					"|.^",
					"example.org",
				},
			},
			"schema_version": newSchemaVer,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo27(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}

func TestUpgradeSchema27to28(t *testing.T) {
	const newSchemaVer = 28

	testCases := []struct {
		in   yobj
		want yobj
		name string
	}{{
		name: "empty",
		in:   yobj{},
		want: yobj{
			"schema_version": newSchemaVer,
		},
	}, {
		name: "load_balance",
		in: yobj{
			"dns": yobj{
				"all_servers":  false,
				"fastest_addr": false,
			},
		},
		want: yobj{
			"dns": yobj{
				"upstream_mode": dnsforward.UpstreamModeLoadBalance,
			},
			"schema_version": newSchemaVer,
		},
	}, {
		name: "parallel",
		in: yobj{
			"dns": yobj{
				"all_servers":  true,
				"fastest_addr": false,
			},
		},
		want: yobj{
			"dns": yobj{
				"upstream_mode": dnsforward.UpstreamModeParallel,
			},
			"schema_version": newSchemaVer,
		},
	}, {
		name: "parallel_fastest",
		in: yobj{
			"dns": yobj{
				"all_servers":  true,
				"fastest_addr": true,
			},
		},
		want: yobj{
			"dns": yobj{
				"upstream_mode": dnsforward.UpstreamModeParallel,
			},
			"schema_version": newSchemaVer,
		},
	}, {
		name: "load_balance",
		in: yobj{
			"dns": yobj{
				"all_servers":  false,
				"fastest_addr": true,
			},
		},
		want: yobj{
			"dns": yobj{
				"upstream_mode": dnsforward.UpstreamModeFastestAddr,
			},
			"schema_version": newSchemaVer,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrateTo28(tc.in)
			require.NoError(t, err)

			assert.Equal(t, tc.want, tc.in)
		})
	}
}
