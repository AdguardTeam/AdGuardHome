package home

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(a.garipov): Cover all migrations, use a testdata/ dir.

func TestUpgradeSchema1to2(t *testing.T) {
	diskConf := testDiskConf(1)

	err := upgradeSchema1to2(diskConf)
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

	err := upgradeSchema2to3(diskConf)
	require.NoError(t, err)

	require.Equal(t, diskConf["schema_version"], 3)

	dnsMap, ok := diskConf["dns"]
	require.True(t, ok)

	newDNSConf, ok := dnsMap.(yobj)
	require.True(t, ok)

	bootstrapDNS := newDNSConf["bootstrap_dns"]
	switch v := bootstrapDNS.(type) {
	case []string:
		require.Len(t, v, 1)
		require.Equal(t, "8.8.8.8:53", v[0])
	default:
		t.Fatalf("wrong type for bootsrap dns: %T", v)
	}

	excludedEntries := []string{"bootstrap_dns"}
	oldDNSConf := testDNSConf(2)
	assertEqualExcept(t, oldDNSConf, newDNSConf, excludedEntries, excludedEntries)

	excludedEntries = []string{"dns", "schema_version"}
	oldDiskConf := testDiskConf(2)
	assertEqualExcept(t, oldDiskConf, diskConf, excludedEntries, excludedEntries)
}

func TestUpgradeSchema7to8(t *testing.T) {
	const host = "1.2.3.4"
	oldConf := yobj{
		"dns": yobj{
			"bind_host": host,
		},
		"schema_version": 7,
	}

	err := upgradeSchema7to8(oldConf)
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

		err := upgradeSchema8to9(oldConf)
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

		err := upgradeSchema8to9(oldConf)
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
	filters := []filter{{
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

// testDNSConf creates a DNS config for test the way gopkg.in/yaml.v2 would
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
		wantErr: "unexpected type of dns.upstream_dns: int",
		name:    "bad_yarr_type",
	}, {
		ups:     yarr{ultimateAns},
		want:    nil,
		wantErr: "unexpected type of upstream field: int",
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
			err := upgradeSchema9to10(conf)

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.wantErr, err.Error())

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
		err := upgradeSchema9to10(yobj{})

		assert.NoError(t, err)
	})

	t.Run("bad_dns", func(t *testing.T) {
		err := upgradeSchema9to10(yobj{
			"dns": ultimateAns,
		})

		require.Error(t, err)
		assert.Equal(t, "unexpected type of dns: int", err.Error())
	})
}

func TestUpgradeSchema10to11(t *testing.T) {
	check := func(t *testing.T, conf yobj) {
		rlimit, _ := conf["rlimit_nofile"].(int)

		err := upgradeSchema10to11(conf)
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
		want:    Duration{Duration: 24 * time.Hour},
		wantErr: "",
		name:    "success",
	}, {
		ivl:     0.25,
		want:    0,
		wantErr: "unexpected type of querylog_interval: float64",
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
			err := upgradeSchema11to12(conf)

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

			var newIvl Duration
			newIvl, ok = newDNSConf["querylog_interval"].(Duration)
			require.True(t, ok)

			assert.Equal(t, tc.want, newIvl)
		})
	}

	t.Run("no_dns", func(t *testing.T) {
		err := upgradeSchema11to12(yobj{})

		assert.NoError(t, err)
	})

	t.Run("bad_dns", func(t *testing.T) {
		err := upgradeSchema11to12(yobj{
			"dns": 0,
		})

		require.Error(t, err)
		assert.Equal(t, "unexpected type of dns: int", err.Error())
	})

	t.Run("no_field", func(t *testing.T) {
		conf := yobj{
			"dns": yobj{},
		}

		err := upgradeSchema11to12(conf)
		require.NoError(t, err)

		dns, ok := conf["dns"]
		require.True(t, ok)

		var dnsVal yobj
		dnsVal, ok = dns.(yobj)
		require.True(t, ok)

		var ivl interface{}
		ivl, ok = dnsVal["querylog_interval"]
		require.True(t, ok)

		var ivlVal Duration
		ivlVal, ok = ivl.(Duration)
		require.True(t, ok)

		assert.Equal(t, 90*24*time.Hour, ivlVal.Duration)
	})
}
