package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/log"
	yaml "gopkg.in/yaml.v2"
)

const currentSchemaVersion = 3 // used for upgrading from old configs to new config

// Performs necessary upgrade operations if needed
func upgradeConfig() error {
	// read a config file into an interface map, so we can manipulate values without losing any
	configFile := config.getConfigFilename()
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("config file %s does not exist, nothing to upgrade", configFile)
		return nil
	}
	diskConfig := map[string]interface{}{}
	body, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("Couldn't read config file '%s': %s", configFile, err)
		return err
	}

	err = yaml.Unmarshal(body, &diskConfig)
	if err != nil {
		log.Printf("Couldn't parse config file '%s': %s", configFile, err)
		return err
	}

	schemaVersionInterface, ok := diskConfig["schema_version"]
	log.Tracef("got schema version %v", schemaVersionInterface)
	if !ok {
		// no schema version, set it to 0
		schemaVersionInterface = 0
	}

	schemaVersion, ok := schemaVersionInterface.(int)
	if !ok {
		err = fmt.Errorf("configuration file contains non-integer schema_version, abort")
		log.Println(err)
		return err
	}

	if schemaVersion == currentSchemaVersion {
		// do nothing
		return nil
	}

	return upgradeConfigSchema(schemaVersion, &diskConfig)
}

// Upgrade from oldVersion to newVersion
func upgradeConfigSchema(oldVersion int, diskConfig *map[string]interface{}) error {
	switch oldVersion {
	case 0:
		err := upgradeSchema0to3(diskConfig)
		if err != nil {
			return err
		}
	case 1:
		err := upgradeSchema1to3(diskConfig)
		if err != nil {
			return err
		}
	case 2:
		err := upgradeSchema2to3(diskConfig)
		if err != nil {
			return err
		}
	default:
		err := fmt.Errorf("configuration file contains unknown schema_version, abort")
		log.Println(err)
		return err
	}

	configFile := config.getConfigFilename()
	body, err := yaml.Marshal(diskConfig)
	if err != nil {
		log.Printf("Couldn't generate YAML file: %s", err)
		return err
	}

	err = safeWriteFile(configFile, body)
	if err != nil {
		log.Printf("Couldn't save YAML config: %s", err)
		return err
	}

	return nil
}

// The first schema upgrade:
// No more "dnsfilter.txt", filters are now kept in data/filters/
func upgradeSchema0to1(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", _Func())

	dnsFilterPath := filepath.Join(config.ourWorkingDir, "dnsfilter.txt")
	if _, err := os.Stat(dnsFilterPath); !os.IsNotExist(err) {
		log.Printf("Deleting %s as we don't need it anymore", dnsFilterPath)
		err = os.Remove(dnsFilterPath)
		if err != nil {
			log.Printf("Cannot remove %s due to %s", dnsFilterPath, err)
			// not fatal, move on
		}
	}

	(*diskConfig)["schema_version"] = 1

	return nil
}

// Second schema upgrade:
// coredns is now dns in config
// delete 'Corefile', since we don't use that anymore
func upgradeSchema1to2(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", _Func())

	coreFilePath := filepath.Join(config.ourWorkingDir, "Corefile")
	if _, err := os.Stat(coreFilePath); !os.IsNotExist(err) {
		log.Printf("Deleting %s as we don't need it anymore", coreFilePath)
		err = os.Remove(coreFilePath)
		if err != nil {
			log.Printf("Cannot remove %s due to %s", coreFilePath, err)
			// not fatal, move on
		}
	}

	if _, ok := (*diskConfig)["dns"]; !ok {
		(*diskConfig)["dns"] = (*diskConfig)["coredns"]
		delete((*diskConfig), "coredns")
	}
	(*diskConfig)["schema_version"] = 2

	return nil
}

// Third schema upgrade:
// Bootstrap DNS becomes an array
func upgradeSchema2to3(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", _Func())

	// Let's read dns configuration from diskConfig
	dnsConfig, ok := (*diskConfig)["dns"]
	if !ok {
		return fmt.Errorf("no DNS configuration in config file")
	}

	// Convert interface{} to map[string]interface{}
	newDNSConfig := make(map[string]interface{})

	switch v := dnsConfig.(type) {
	case map[interface{}]interface{}:
		for k, v := range v {
			newDNSConfig[fmt.Sprint(k)] = v
		}
	default:
		return fmt.Errorf("DNS configuration is not a map")
	}

	// Replace bootstrap_dns value filed with new array contains old bootstrap_dns inside
	if bootstrapDNS, ok := (newDNSConfig)["bootstrap_dns"]; ok {
		newBootstrapConfig := []string{fmt.Sprint(bootstrapDNS)}
		(newDNSConfig)["bootstrap_dns"] = newBootstrapConfig
		(*diskConfig)["dns"] = newDNSConfig
	} else {
		return fmt.Errorf("no bootstrap DNS in DNS config")
	}

	// Bump schema version
	(*diskConfig)["schema_version"] = 3

	return nil
}

// jump three schemas at once -- this time we just do it sequentially
func upgradeSchema0to3(diskConfig *map[string]interface{}) error {
	err := upgradeSchema0to1(diskConfig)
	if err != nil {
		return err
	}

	return upgradeSchema1to3(diskConfig)
}

// jump two schemas at once -- this time we just do it sequentially
func upgradeSchema1to3(diskConfig *map[string]interface{}) error {
	err := upgradeSchema1to2(diskConfig)
	if err != nil {
		return err
	}

	return upgradeSchema2to3(diskConfig)
}
