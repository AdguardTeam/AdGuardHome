package main

import (
	"fmt"
	"testing"
)

func TestUpgrade1to2(t *testing.T) {
	// Let's create test config
	diskConfig := createTestDiskConfig(1)
	oldDNSConfig := createTestDNSConfig(1)

	err := upgradeSchema1to2(&diskConfig)
	if err != nil {
		t.Fatalf("Can't upgrade schema version from 1 to 2")
	}

	_, ok := diskConfig["coredns"]
	if ok {
		t.Fatalf("Core DNS config was not removed after upgrade schema version from 1 to 2")
	}

	dnsMap, ok := diskConfig["dns"]
	if !ok {
		t.Fatalf("No DNS config after upgrade schema version from 1 to 2")
	}

	// Cast dns configuration to map
	newDNSConfig := make(map[string]interface{})
	switch v := dnsMap.(type) {
	case map[interface{}]interface{}:
		if len(oldDNSConfig) != len(v) {
			t.Fatalf("We loose some data")
		}
		for key, value := range v {
			newDNSConfig[fmt.Sprint(key)] = value
		}
	default:
		t.Fatalf("DNS configuration is not a map")
	}

	_, v, err := compareConfigs(oldDNSConfig, newDNSConfig)
	if err != nil {
		t.Fatalf("Wrong data %s, %s", v, err)
	}
}

func TestUpgrade2to3(t *testing.T) {
	// Let's create test config
	diskConfig := createTestDiskConfig(2)
	oldDNSConfig := createTestDNSConfig(2)

	// Upgrade schema from 2 to 3
	err := upgradeSchema2to3(&diskConfig)
	if err != nil {
		t.Fatalf("Can't update schema version from 2 to 3: %s", err)
	}

	// Check new schema version
	newSchemaVersion := diskConfig["schema_version"]
	switch v := newSchemaVersion.(type) {
	case int:
		if v != 3 {
			t.Fatalf("Wrong schema version in new config file")
		}
	default:
		t.Fatalf("Schema version is not an integer after update")
	}

	// Let's get new dns configuration
	dnsMap, ok := diskConfig["dns"]
	if !ok {
		t.Fatalf("No dns config in new configuration")
	}

	// Cast dns configuration to map
	newDNSConfig := make(map[string]interface{})
	switch v := dnsMap.(type) {
	case map[string]interface{}:
		for key, value := range v {
			newDNSConfig[fmt.Sprint(key)] = value
		}
	default:
		t.Fatalf("DNS configuration is not a map")
	}

	// Check if bootstrap DNS becomes an array
	bootstrapDNS := newDNSConfig["bootstrap_dns"]
	switch v := bootstrapDNS.(type) {
	case []string:
		if len(v) != 1 {
			t.Fatalf("Wrong count of bootsrap DNS servers")
		}

		if v[0] != "8.8.8.8:53" {
			t.Fatalf("Wrong bootsrap DNS servers")
		}
	default:
		t.Fatalf("Wrong type for bootsrap DNS")
	}

	// Set old value for bootstrap_dns and compare old and new configurations
	newDNSConfig["bootstrap_dns"] = "8.8.8.8:53"
	_, v, err := compareConfigs(oldDNSConfig, newDNSConfig)
	if err != nil {
		t.Fatalf("%s value is wrong: %s", v, err)
	}
}

// TODO add comparation for all possible types
func compareConfigs(oldConfig map[interface{}]interface{}, newConfig map[string]interface{}) (string, interface{}, error) {
	oldConfigCasted := make(map[string]interface{})
	for k, v := range oldConfig {
		oldConfigCasted[fmt.Sprint(k)] = v
	}

	// Check old data and new data
	for k, v := range newConfig {
		switch value := v.(type) {
		case string:
		case int:
			if value != oldConfigCasted[k] {

			}
		case []string:
			for i, s := range value {
				if oldConfigCasted[k].([]string)[i] != s {
					return k, v, fmt.Errorf("wrong data for %s", k)
				}
			}
		case bool:
			if v != oldConfigCasted[k].(bool) {
				return k, v, fmt.Errorf("wrong data for %s", k)
			}
		default:
			return k, v, fmt.Errorf("unknown type in DNS configuration for %s", k)
		}
	}

	return "", nil, nil
}

func createTestDiskConfig(schemaVersion int) (diskConfig map[string]interface{}) {
	diskConfig = make(map[string]interface{})
	diskConfig["language"] = "en"
	diskConfig["filters"] = []filter{}
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
