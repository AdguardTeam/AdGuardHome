package home

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// any is a convenient alias for interface{}.
type any = interface{}

// object is a convenient alias for map[string]interface{}.
type object = map[string]any

func TestUpgradeSchema1to2(t *testing.T) {
	diskConf := testDiskConf(1)

	err := upgradeSchema1to2(&diskConf)
	require.Nil(t, err)

	require.Equal(t, diskConf["schema_version"], 2)

	_, ok := diskConf["coredns"]
	require.False(t, ok)

	dnsMap, ok := diskConf["dns"]
	require.True(t, ok)

	oldDNSConf := convertToObject(t, testDNSConf(1))
	newDNSConf := convertToObject(t, dnsMap)
	assert.Equal(t, oldDNSConf, newDNSConf)

	oldExcludedEntries := []string{"coredns", "schema_version"}
	newExcludedEntries := []string{"dns", "schema_version"}
	oldDiskConf := testDiskConf(1)
	assertEqualExcept(t, oldDiskConf, diskConf, oldExcludedEntries, newExcludedEntries)
}

func TestUpgradeSchema2to3(t *testing.T) {
	diskConf := testDiskConf(2)

	err := upgradeSchema2to3(&diskConf)
	require.Nil(t, err)

	require.Equal(t, diskConf["schema_version"], 3)

	dnsMap, ok := diskConf["dns"]
	require.True(t, ok)

	newDNSConf := convertToObject(t, dnsMap)
	bootstrapDNS := newDNSConf["bootstrap_dns"]
	switch v := bootstrapDNS.(type) {
	case []string:
		require.Len(t, v, 1)
		require.Equal(t, "8.8.8.8:53", v[0])
	default:
		t.Fatalf("wrong type for bootsrap dns: %T", v)
	}

	excludedEntries := []string{"bootstrap_dns"}
	oldDNSConf := convertToObject(t, testDNSConf(2))
	assertEqualExcept(t, oldDNSConf, newDNSConf, excludedEntries, excludedEntries)

	excludedEntries = []string{"dns", "schema_version"}
	oldDiskConf := testDiskConf(2)
	assertEqualExcept(t, oldDiskConf, diskConf, excludedEntries, excludedEntries)
}

func convertToObject(t *testing.T, oldConf any) (newConf object) {
	t.Helper()

	switch v := oldConf.(type) {
	case map[any]any:
		newConf = make(object, len(v))
		for key, value := range v {
			newConf[fmt.Sprint(key)] = value
		}
	case object:
		newConf = make(object, len(v))
		for key, value := range v {
			newConf[key] = value
		}
	default:
		t.Fatalf("dns configuration is not a map, got %T", oldConf)
	}

	return newConf
}

// assertEqualExcept removes entries from configs and compares them.
func assertEqualExcept(t *testing.T, oldConf, newConf object, oldKeys, newKeys []string) {
	t.Helper()

	for _, k := range oldKeys {
		delete(oldConf, k)
	}
	for _, k := range newKeys {
		delete(newConf, k)
	}

	assert.Equal(t, oldConf, newConf)
}

func testDiskConf(schemaVersion int) (diskConf object) {
	filters := []filter{
		{
			URL:        "https://filters.adtidy.org/android/filters/111_optimized.txt",
			Name:       "Latvian filter",
			RulesCount: 100,
		},
		{
			URL:        "https://easylist.to/easylistgermany/easylistgermany.txt",
			Name:       "Germany filter",
			RulesCount: 200,
		},
	}
	diskConf = object{
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
func testDNSConf(schemaVersion int) (dnsConf map[any]any) {
	dnsConf = map[any]any{
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
