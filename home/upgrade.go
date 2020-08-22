package home

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/crypto/bcrypt"
	yaml "gopkg.in/yaml.v2"
)

const currentSchemaVersion = 7 // used for upgrading from old configs to new config

// Performs necessary upgrade operations if needed
func upgradeConfig() error {
	// read a config file into an interface map, so we can manipulate values without losing any
	diskConfig := map[string]interface{}{}
	body, err := readConfigFile()
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(body, &diskConfig)
	if err != nil {
		log.Printf("Couldn't parse config file: %s", err)
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
		err := upgradeSchema0to1(diskConfig)
		if err != nil {
			return err
		}
		fallthrough
	case 1:
		err := upgradeSchema1to2(diskConfig)
		if err != nil {
			return err
		}
		fallthrough
	case 2:
		err := upgradeSchema2to3(diskConfig)
		if err != nil {
			return err
		}
		fallthrough
	case 3:
		err := upgradeSchema3to4(diskConfig)
		if err != nil {
			return err
		}
		fallthrough
	case 4:
		err := upgradeSchema4to5(diskConfig)
		if err != nil {
			return err
		}
		fallthrough
	case 5:
		err := upgradeSchema5to6(diskConfig)
		if err != nil {
			return err
		}
		fallthrough
	case 6:
		err := upgradeSchema6to7(diskConfig)
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

	config.fileData = body
	err = file.SafeWrite(configFile, body)
	if err != nil {
		log.Printf("Couldn't save YAML config: %s", err)
		return err
	}

	return nil
}

// The first schema upgrade:
// No more "dnsfilter.txt", filters are now kept in data/filters/
func upgradeSchema0to1(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", util.FuncName())

	dnsFilterPath := filepath.Join(Context.workDir, "dnsfilter.txt")
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
	log.Printf("%s(): called", util.FuncName())

	coreFilePath := filepath.Join(Context.workDir, "Corefile")
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
	log.Printf("%s(): called", util.FuncName())

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

// Add use_global_blocked_services=true setting for existing "clients" array
func upgradeSchema3to4(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", util.FuncName())

	(*diskConfig)["schema_version"] = 4

	clients, ok := (*diskConfig)["clients"]
	if !ok {
		return nil
	}

	switch arr := clients.(type) {
	case []interface{}:

		for i := range arr {

			switch c := arr[i].(type) {

			case map[interface{}]interface{}:
				c["use_global_blocked_services"] = true

			default:
				continue
			}
		}

	default:
		return nil
	}

	return nil
}

// Replace "auth_name", "auth_pass" string settings with an array:
// users:
// - name: "..."
//   password: "..."
// ...
func upgradeSchema4to5(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", util.FuncName())

	(*diskConfig)["schema_version"] = 5

	name, ok := (*diskConfig)["auth_name"]
	if !ok {
		return nil
	}
	nameStr, ok := name.(string)
	if !ok {
		log.Fatal("Please use double quotes in your user name in \"auth_name\" and restart AdGuardHome")
		return nil
	}

	pass, ok := (*diskConfig)["auth_pass"]
	if !ok {
		return nil
	}
	passStr, ok := pass.(string)
	if !ok {
		log.Fatal("Please use double quotes in your password in \"auth_pass\" and restart AdGuardHome")
		return nil
	}

	if len(nameStr) == 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(passStr), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Can't use password \"%s\": bcrypt.GenerateFromPassword: %s", passStr, err)
		return nil
	}
	u := User{
		Name:         nameStr,
		PasswordHash: string(hash),
	}
	users := []User{u}
	(*diskConfig)["users"] = users
	return nil
}

// clients:
// ...
//   ip: 127.0.0.1
//   mac: ...
//
// ->
//
// clients:
// ...
//   ids:
//   - 127.0.0.1
//   - ...
func upgradeSchema5to6(diskConfig *map[string]interface{}) error {
	log.Printf("%s(): called", util.FuncName())

	(*diskConfig)["schema_version"] = 6

	clients, ok := (*diskConfig)["clients"]
	if !ok {
		return nil
	}

	switch arr := clients.(type) {
	case []interface{}:

		for i := range arr {

			switch c := arr[i].(type) {

			case map[interface{}]interface{}:
				_ip, ok := c["ip"]
				ids := []string{}
				if ok {
					ip, ok := _ip.(string)
					if !ok {
						log.Fatalf("client.ip is not a string: %v", _ip)
						return nil
					}
					if len(ip) != 0 {
						ids = append(ids, ip)
					}
				}

				_mac, ok := c["mac"]
				if ok {
					mac, ok := _mac.(string)
					if !ok {
						log.Fatalf("client.mac is not a string: %v", _mac)
						return nil
					}
					if len(mac) != 0 {
						ids = append(ids, mac)
					}
				}

				c["ids"] = ids

			default:
				continue
			}
		}

	default:
		return nil
	}

	return nil
}

// dhcp:
//   enabled: false
//   interface_name: vboxnet0
//   gateway_ip: 192.168.56.1
//   ...
//
// ->
//
// dhcp:
//   enabled: false
//   interface_name: vboxnet0
//   dhcpv4:
//     gateway_ip: 192.168.56.1
//     ...
func upgradeSchema6to7(diskConfig *map[string]interface{}) error {
	log.Printf("Upgrade yaml: 6 to 7")

	(*diskConfig)["schema_version"] = 7

	_dhcp, ok := (*diskConfig)["dhcp"]
	if !ok {
		return nil
	}

	switch dhcp := _dhcp.(type) {
	case map[interface{}]interface{}:
		dhcpv4 := map[string]interface{}{}
		val, ok := dhcp["gateway_ip"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "gateway_ip")
			return nil
		}
		dhcpv4["gateway_ip"] = val
		delete(dhcp, "gateway_ip")

		val, ok = dhcp["subnet_mask"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "subnet_mask")
			return nil
		}
		dhcpv4["subnet_mask"] = val
		delete(dhcp, "subnet_mask")

		val, ok = dhcp["range_start"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "range_start")
			return nil
		}
		dhcpv4["range_start"] = val
		delete(dhcp, "range_start")

		val, ok = dhcp["range_end"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "range_end")
			return nil
		}
		dhcpv4["range_end"] = val
		delete(dhcp, "range_end")

		intVal, ok := dhcp["lease_duration"].(int)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be an integer", "lease_duration")
			return nil
		}
		dhcpv4["lease_duration"] = intVal
		delete(dhcp, "lease_duration")

		intVal, ok = dhcp["icmp_timeout_msec"].(int)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be an integer", "icmp_timeout_msec")
			return nil
		}
		dhcpv4["icmp_timeout_msec"] = intVal
		delete(dhcp, "icmp_timeout_msec")

		dhcp["dhcpv4"] = dhcpv4

	default:
		return nil
	}

	return nil
}
