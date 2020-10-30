package home

import (
	"fmt"
	"testing"
)

func TestUpgrade1to2(t *testing.T) {
	// let's create test config for 1 schema version
	diskConfig := createTestDiskConfig(1)

	// update config
	err := upgradeSchema1to2(&diskConfig)
	if err != nil {
		t.Fatalf("Can't upgrade schema version from 1 to 2")
	}

	// ensure that schema version was bumped
	compareSchemaVersion(t, diskConfig["schema_version"], 2)

	// old coredns entry should be removed
	_, ok := diskConfig["coredns"]
	if ok {
		t.Fatalf("Core DNS config was not removed after upgrade schema version from 1 to 2")
	}

	// pull out new dns config
	dnsMap, ok := diskConfig["dns"]
	if !ok {
		t.Fatalf("No DNS config after upgrade schema version from 1 to 2")
	}

	// cast dns configurations to maps and compare them
	oldDNSConfig := castInterfaceToMap(t, createTestDNSConfig(1))
	newDNSConfig := castInterfaceToMap(t, dnsMap)
	compareConfigs(t, &oldDNSConfig, &newDNSConfig)

	// exclude dns config and schema version from disk config comparison
	oldExcludedEntries := []string{"coredns", "schema_version"}
	newExcludedEntries := []string{"dns", "schema_version"}
	oldDiskConfig := createTestDiskConfig(1)
	compareConfigsWithoutEntries(t, &oldDiskConfig, &diskConfig, oldExcludedEntries, newExcludedEntries)
}

func TestUpgrade2to3(t *testing.T) {
	// let's create test config
	diskConfig := createTestDiskConfig(2)

	// upgrade schema from 2 to 3
	err := upgradeSchema2to3(&diskConfig)
	if err != nil {
		t.Fatalf("Can't update schema version from 2 to 3: %s", err)
	}

	// check new schema version
	compareSchemaVersion(t, diskConfig["schema_version"], 3)

	// pull out new dns configuration
	dnsMap, ok := diskConfig["dns"]
	if !ok {
		t.Fatalf("No dns config in new configuration")
	}

	// cast dns configuration to map
	newDNSConfig := castInterfaceToMap(t, dnsMap)

	// check if bootstrap DNS becomes an array
	bootstrapDNS := newDNSConfig["bootstrap_dns"]
	switch v := bootstrapDNS.(type) {
	case []string:
		if len(v) != 1 {
			t.Fatalf("Wrong count of bootsrap DNS servers: %d", len(v))
		}

		if v[0] != "8.8.8.8:53" {
			t.Fatalf("Bootsrap DNS server is not 8.8.8.8:53 : %s", v[0])
		}
	default:
		t.Fatalf("Wrong type for bootsrap DNS: %T", v)
	}

	// exclude bootstrap DNS from DNS configs comparison
	excludedEntries := []string{"bootstrap_dns"}
	oldDNSConfig := castInterfaceToMap(t, createTestDNSConfig(2))
	compareConfigsWithoutEntries(t, &oldDNSConfig, &newDNSConfig, excludedEntries, excludedEntries)

	// excluded dns config and schema version from disk config comparison
	excludedEntries = []string{"dns", "schema_version"}
	oldDiskConfig := createTestDiskConfig(2)
	compareConfigsWithoutEntries(t, &oldDiskConfig, &diskConfig, excludedEntries, excludedEntries)
}

func castInterfaceToMap(t *testing.T, oldConfig interface{}) (newConfig map[string]interface{}) {
	newConfig = make(map[string]interface{})
	switch v := oldConfig.(type) {
	case map[interface{}]interface{}:
		for key, value := range v {
			newConfig[fmt.Sprint(key)] = value
		}
	case map[string]interface{}:
		for key, value := range v {
			newConfig[key] = value
		}
	default:
		t.Fatalf("DNS configuration is not a map")
	}
	return
}

// compareConfigsWithoutEntry removes entries from configs and returns result of compareConfigs
func compareConfigsWithoutEntries(t *testing.T, oldConfig, newConfig *map[string]interface{}, oldKey, newKey []string) {
	for _, k := range oldKey {
		delete(*oldConfig, k)
	}
	for _, k := range newKey {
		delete(*newConfig, k)
	}
	compareConfigs(t, oldConfig, newConfig)
}

// compares configs before and after schema upgrade
func compareConfigs(t *testing.T, oldConfig, newConfig *map[string]interface{}) {
	if len(*oldConfig) != len(*newConfig) {
		t.Fatalf("wrong config entries count! Before upgrade: %d; After upgrade: %d", len(*oldConfig), len(*oldConfig))
	}

	// Check old and new entries
	for k, v := range *newConfig {
		switch value := v.(type) {
		case string:
			if value != (*oldConfig)[k] {
				t.Fatalf("wrong value for string %s. Before update: %s; After update: %s", k, (*oldConfig)[k], value)
			}
		case int:
			if value != (*oldConfig)[k] {
				t.Fatalf("wrong value for int %s. Before update: %d; After update: %d", k, (*oldConfig)[k], value)
			}
		case []string:
			for i, line := range value {
				if len((*oldConfig)[k].([]string)) != len(value) {
					t.Fatalf("wrong array length for %s. Before update: %d; After update: %d", k, len((*oldConfig)[k].([]string)), len(value))
				}
				if (*oldConfig)[k].([]string)[i] != line {
					t.Fatalf("wrong data for string array %s. Before update: %s; After update: %s", k, (*oldConfig)[k].([]string)[i], line)
				}
			}
		case bool:
			if v != (*oldConfig)[k].(bool) {
				t.Fatalf("wrong boolean value for %s", k)
			}
		case []filter:
			if len((*oldConfig)[k].([]filter)) != len(value) {
				t.Fatalf("wrong filters count. Before update: %d; After update: %d", len((*oldConfig)[k].([]filter)), len(value))
			}
			for i, newFilter := range value {
				oldFilter := (*oldConfig)[k].([]filter)[i]
				if oldFilter.Enabled != newFilter.Enabled || oldFilter.Name != newFilter.Name || oldFilter.RulesCount != newFilter.RulesCount {
					t.Fatalf("old filter %s not equals new filter %s", oldFilter.Name, newFilter.Name)
				}
			}
		default:
			t.Fatalf("uknown data type for %s: %T", k, value)
		}
	}
}

// compareSchemaVersion check if newSchemaVersion equals schemaVersion
func compareSchemaVersion(t *testing.T, newSchemaVersion interface{}, schemaVersion int) {
	switch v := newSchemaVersion.(type) {
	case int:
		if v != schemaVersion {
			t.Fatalf("Wrong schema version in new config file")
		}
	default:
		t.Fatalf("Schema version is not an integer after update")
	}
}

func createTestDiskConfig(schemaVersion int) (diskConfig map[string]interface{}) {
	diskConfig = make(map[string]interface{})
	diskConfig["language"] = "en"
	diskConfig["filters"] = []filter{
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
	diskConfig["user_rules"] = []string{}
	diskConfig["schema_version"] = schemaVersion
	diskConfig["bind_host"] = "0.0.0.0"
	diskConfig["bind_port"] = 80
	diskConfig["auth_name"] = "name"
	diskConfig["auth_pass"] = "pass"
	dnsConfig := createTestDNSConfig(schemaVersion)
	if schemaVersion > 1 {
		diskConfig["dns"] = dnsConfig
	} else {
		diskConfig["coredns"] = dnsConfig
	}
	return diskConfig
}

func createTestDNSConfig(schemaVersion int) map[interface{}]interface{} {
	dnsConfig := make(map[interface{}]interface{})
	dnsConfig["port"] = 53
	dnsConfig["blocked_response_ttl"] = 10
	dnsConfig["querylog_enabled"] = true
	dnsConfig["ratelimit"] = 20
	dnsConfig["bootstrap_dns"] = "8.8.8.8:53"
	if schemaVersion > 2 {
		dnsConfig["bootstrap_dns"] = []string{"8.8.8.8:53"}
	}
	dnsConfig["parental_sensitivity"] = 13
	dnsConfig["ratelimit_whitelist"] = []string{}
	dnsConfig["upstream_dns"] = []string{"tls://1.1.1.1", "tls://1.0.0.1", "8.8.8.8"}
	dnsConfig["filtering_enabled"] = true
	dnsConfig["refuse_any"] = true
	dnsConfig["parental_enabled"] = true
	dnsConfig["bind_host"] = "0.0.0.0"
	dnsConfig["protection_enabled"] = true
	dnsConfig["safesearch_enabled"] = true
	dnsConfig["safebrowsing_enabled"] = true
	return dnsConfig
}
