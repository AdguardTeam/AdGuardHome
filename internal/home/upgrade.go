package home

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/crypto/bcrypt"
	yaml "gopkg.in/yaml.v2"
)

// currentSchemaVersion is the current schema version.
const currentSchemaVersion = 8

// These aliases are provided for convenience.
type (
	any  = interface{}
	yarr = []any
	yobj = map[any]any
)

// Performs necessary upgrade operations if needed
func upgradeConfig() error {
	// read a config file into an interface map, so we can manipulate values without losing any
	diskConf := yobj{}
	body, err := readConfigFile()
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(body, &diskConf)
	if err != nil {
		log.Printf("Couldn't parse config file: %s", err)
		return err
	}

	schemaVersionInterface, ok := diskConf["schema_version"]
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

	return upgradeConfigSchema(schemaVersion, diskConf)
}

// upgradeFunc is a function that upgrades a config and returns an error.
type upgradeFunc = func(diskConf yobj) (err error)

// Upgrade from oldVersion to newVersion
func upgradeConfigSchema(oldVersion int, diskConf yobj) (err error) {
	upgrades := []upgradeFunc{
		upgradeSchema0to1,
		upgradeSchema1to2,
		upgradeSchema2to3,
		upgradeSchema3to4,
		upgradeSchema4to5,
		upgradeSchema5to6,
		upgradeSchema6to7,
		upgradeSchema7to8,
	}

	n := 0
	for i, u := range upgrades {
		if i >= oldVersion {
			err = u(diskConf)
			if err != nil {
				return err
			}

			n++
		}
	}

	if n == 0 {
		return fmt.Errorf("unknown configuration schema version %d", oldVersion)
	}

	body, err := yaml.Marshal(diskConf)
	if err != nil {
		return fmt.Errorf("generating new config: %w", err)
	}

	config.fileData = body
	confFile := config.getConfigFilename()
	err = file.SafeWrite(confFile, body)
	if err != nil {
		return fmt.Errorf("saving new config: %w", err)
	}

	return nil
}

// The first schema upgrade:
// No more "dnsfilter.txt", filters are now kept in data/filters/
func upgradeSchema0to1(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	dnsFilterPath := filepath.Join(Context.workDir, "dnsfilter.txt")
	if _, err := os.Stat(dnsFilterPath); !os.IsNotExist(err) {
		log.Printf("Deleting %s as we don't need it anymore", dnsFilterPath)
		err = os.Remove(dnsFilterPath)
		if err != nil {
			log.Printf("Cannot remove %s due to %s", dnsFilterPath, err)
			// not fatal, move on
		}
	}

	diskConf["schema_version"] = 1

	return nil
}

// Second schema upgrade:
// coredns is now dns in config
// delete 'Corefile', since we don't use that anymore
func upgradeSchema1to2(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	coreFilePath := filepath.Join(Context.workDir, "Corefile")
	if _, err := os.Stat(coreFilePath); !os.IsNotExist(err) {
		log.Printf("Deleting %s as we don't need it anymore", coreFilePath)
		err = os.Remove(coreFilePath)
		if err != nil {
			log.Printf("Cannot remove %s due to %s", coreFilePath, err)
			// not fatal, move on
		}
	}

	if _, ok := diskConf["dns"]; !ok {
		diskConf["dns"] = diskConf["coredns"]
		delete(diskConf, "coredns")
	}
	diskConf["schema_version"] = 2

	return nil
}

// Third schema upgrade:
// Bootstrap DNS becomes an array
func upgradeSchema2to3(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	// Let's read dns configuration from diskConf
	dnsConfig, ok := diskConf["dns"]
	if !ok {
		return fmt.Errorf("no DNS configuration in config file")
	}

	// Convert interface{} to yobj
	newDNSConfig := make(yobj)

	switch v := dnsConfig.(type) {
	case map[interface{}]interface{}:
		for k, v := range v {
			newDNSConfig[fmt.Sprint(k)] = v
		}
	default:
		return fmt.Errorf("dns configuration is not a map")
	}

	// Replace bootstrap_dns value filed with new array contains old bootstrap_dns inside
	bootstrapDNS, ok := newDNSConfig["bootstrap_dns"]
	if !ok {
		return fmt.Errorf("no bootstrap DNS in DNS config")
	}

	newBootstrapConfig := []string{fmt.Sprint(bootstrapDNS)}
	newDNSConfig["bootstrap_dns"] = newBootstrapConfig
	diskConf["dns"] = newDNSConfig

	// Bump schema version
	diskConf["schema_version"] = 3

	return nil
}

// Add use_global_blocked_services=true setting for existing "clients" array
func upgradeSchema3to4(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	diskConf["schema_version"] = 4

	clients, ok := diskConf["clients"]
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
func upgradeSchema4to5(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	diskConf["schema_version"] = 5

	name, ok := diskConf["auth_name"]
	if !ok {
		return nil
	}
	nameStr, ok := name.(string)
	if !ok {
		log.Fatal("Please use double quotes in your user name in \"auth_name\" and restart AdGuardHome")
		return nil
	}

	pass, ok := diskConf["auth_pass"]
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
	diskConf["users"] = users
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
func upgradeSchema5to6(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	diskConf["schema_version"] = 6

	clients, ok := diskConf["clients"]
	if !ok {
		return nil
	}

	switch arr := clients.(type) {
	case []interface{}:
		for i := range arr {
			switch c := arr[i].(type) {
			case map[interface{}]interface{}:
				var ipVal interface{}
				ipVal, ok = c["ip"]
				ids := []string{}
				if ok {
					var ip string
					ip, ok = ipVal.(string)
					if !ok {
						log.Fatalf("client.ip is not a string: %v", ipVal)
						return nil
					}
					if len(ip) != 0 {
						ids = append(ids, ip)
					}
				}

				var macVal interface{}
				macVal, ok = c["mac"]
				if ok {
					var mac string
					mac, ok = macVal.(string)
					if !ok {
						log.Fatalf("client.mac is not a string: %v", macVal)
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
func upgradeSchema6to7(diskConf yobj) error {
	log.Printf("Upgrade yaml: 6 to 7")

	diskConf["schema_version"] = 7

	dhcpVal, ok := diskConf["dhcp"]
	if !ok {
		return nil
	}

	switch dhcp := dhcpVal.(type) {
	case map[interface{}]interface{}:
		var str string
		str, ok = dhcp["gateway_ip"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "gateway_ip")
			return nil
		}

		dhcpv4 := yobj{
			"gateway_ip": str,
		}
		delete(dhcp, "gateway_ip")

		str, ok = dhcp["subnet_mask"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "subnet_mask")
			return nil
		}
		dhcpv4["subnet_mask"] = str
		delete(dhcp, "subnet_mask")

		str, ok = dhcp["range_start"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "range_start")
			return nil
		}
		dhcpv4["range_start"] = str
		delete(dhcp, "range_start")

		str, ok = dhcp["range_end"].(string)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be a string", "range_end")
			return nil
		}
		dhcpv4["range_end"] = str
		delete(dhcp, "range_end")

		var n int
		n, ok = dhcp["lease_duration"].(int)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be an integer", "lease_duration")
			return nil
		}
		dhcpv4["lease_duration"] = n
		delete(dhcp, "lease_duration")

		n, ok = dhcp["icmp_timeout_msec"].(int)
		if !ok {
			log.Fatalf("expecting dhcp.%s to be an integer", "icmp_timeout_msec")
			return nil
		}
		dhcpv4["icmp_timeout_msec"] = n
		delete(dhcp, "icmp_timeout_msec")

		dhcp["dhcpv4"] = dhcpv4
	default:
		return nil
	}

	return nil
}

// upgradeSchema7to8 performs the following changes:
//
//   # BEFORE:
//   'dns':
//     'bind_host': '127.0.0.1'
//
//   # AFTER:
//   'dns':
//     'bind_hosts':
//     - '127.0.0.1'
//
func upgradeSchema7to8(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 7 to 8")

	diskConf["schema_version"] = 8

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	dns, ok := dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	bindHostVal := dns["bind_host"]
	bindHost, ok := bindHostVal.(string)
	if !ok {
		return fmt.Errorf("undexpected type of dns.bind_host: %T", bindHostVal)
	}

	delete(dns, "bind_host")
	dns["bind_hosts"] = yarr{bindHost}

	return nil
}

// TODO(a.garipov): Replace with log.Output when we port it to our logging
// package.
func funcName() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}
