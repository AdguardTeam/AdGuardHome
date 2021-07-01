package home

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/renameio/maybe"
	"golang.org/x/crypto/bcrypt"
	yaml "gopkg.in/yaml.v2"
)

// currentSchemaVersion is the current schema version.
const currentSchemaVersion = 12

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
		upgradeSchema8to9,
		upgradeSchema9to10,
		upgradeSchema10to11,
		upgradeSchema11to12,
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
	err = maybe.WriteFile(confFile, body, 0o644)
	if err != nil {
		return fmt.Errorf("saving new config: %w", err)
	}

	return nil
}

// The first schema upgrade:
// No more "dnsfilter.txt", filters are now kept in data/filters/
func upgradeSchema0to1(diskConf yobj) (err error) {
	log.Printf("%s(): called", funcName())

	dnsFilterPath := filepath.Join(Context.workDir, "dnsfilter.txt")
	log.Printf("deleting %s as we don't need it anymore", dnsFilterPath)
	err = os.Remove(dnsFilterPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Info("warning: %s", err)

		// Go on.
	}

	diskConf["schema_version"] = 1

	return nil
}

// Second schema upgrade:
// coredns is now dns in config
// delete 'Corefile', since we don't use that anymore
func upgradeSchema1to2(diskConf yobj) (err error) {
	log.Printf("%s(): called", funcName())

	coreFilePath := filepath.Join(Context.workDir, "Corefile")
	log.Printf("deleting %s as we don't need it anymore", coreFilePath)
	err = os.Remove(coreFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Info("warning: %s", err)

		// Go on.
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
		return fmt.Errorf("unexpected type of dns.bind_host: %T", bindHostVal)
	}

	delete(dns, "bind_host")
	dns["bind_hosts"] = yarr{bindHost}

	return nil
}

// upgradeSchema8to9 performs the following changes:
//
//   # BEFORE:
//   'dns':
//     'autohost_tld': 'lan'
//
//   # AFTER:
//   'dns':
//     'local_domain_name': 'lan'
//
func upgradeSchema8to9(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 8 to 9")

	diskConf["schema_version"] = 9

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	dns, ok := dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	autohostTLDVal, ok := dns["autohost_tld"]
	if !ok {
		// This happens when upgrading directly from v0.105.2, because
		// dns.autohost_tld was never set to any value.  Go on and leave
		// it that way.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2988.
		return nil
	}

	autohostTLD, ok := autohostTLDVal.(string)
	if !ok {
		return fmt.Errorf("unexpected type of dns.autohost_tld: %T", autohostTLDVal)
	}

	delete(dns, "autohost_tld")
	dns["local_domain_name"] = autohostTLD

	return nil
}

// addQUICPort inserts a port into QUIC upstream's hostname if it is missing.
func addQUICPort(ups string, port int) (withPort string) {
	if ups == "" || ups[0] == '#' {
		return ups
	}

	var doms string
	withPort = ups
	if strings.HasPrefix(ups, "[/") {
		domsAndUps := strings.Split(strings.TrimPrefix(ups, "[/"), "/]")
		if len(domsAndUps) != 2 {
			return ups
		}

		doms, withPort = "[/"+domsAndUps[0]+"/]", domsAndUps[1]
	}

	if !strings.Contains(withPort, "://") {
		return ups
	}

	upsURL, err := url.Parse(withPort)
	if err != nil || upsURL.Scheme != "quic" {
		return ups
	}

	var host string
	host, err = aghnet.SplitHost(upsURL.Host)
	if err != nil || host != upsURL.Host {
		return ups
	}

	upsURL.Host = strings.Join([]string{host, strconv.Itoa(port)}, ":")

	return doms + upsURL.String()
}

// upgradeSchema9to10 performs the following changes:
//
//   # BEFORE:
//   'dns':
//     'upstream_dns':
//      - 'quic://some-upstream.com'
//
//   # AFTER:
//   'dns':
//     'upstream_dns':
//      - 'quic://some-upstream.com:784'
//
func upgradeSchema9to10(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 9 to 10")

	diskConf["schema_version"] = 10

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	var dns yobj
	dns, ok = dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	const quicPort = 784
	for _, upsField := range []string{
		"upstream_dns",
		"local_ptr_upstreams",
	} {
		var upsVal any
		upsVal, ok = dns[upsField]
		if !ok {
			continue
		}
		var ups yarr
		ups, ok = upsVal.(yarr)
		if !ok {
			return fmt.Errorf("unexpected type of dns.%s: %T", upsField, upsVal)
		}

		var u string
		for i, uVal := range ups {
			u, ok = uVal.(string)
			if !ok {
				return fmt.Errorf("unexpected type of upstream field: %T", uVal)
			}

			ups[i] = addQUICPort(u, quicPort)
		}
		dns[upsField] = ups
	}

	return nil
}

// upgradeSchema10to11 performs the following changes:
//
//   # BEFORE:
//   'rlimit_nofile': 42
//
//   # AFTER:
//   'os':
//     'group': ''
//     'rlimit_nofile': 42
//     'user': ''
//
func upgradeSchema10to11(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 10 to 11")

	diskConf["schema_version"] = 11

	rlimit := 0
	rlimitVal, ok := diskConf["rlimit_nofile"]
	if ok {
		rlimit, ok = rlimitVal.(int)
		if !ok {
			return fmt.Errorf("unexpected type of rlimit_nofile: %T", rlimitVal)
		}
	}

	delete(diskConf, "rlimit_nofile")
	diskConf["os"] = yobj{
		"group":         "",
		"rlimit_nofile": rlimit,
		"user":          "",
	}

	return nil
}

// upgradeSchema11to12 performs the following changes:
//
//   # BEFORE:
//   'querylog_interval': 90
//
//   # AFTER:
//   'querylog_interval': '2160h'
//
func upgradeSchema11to12(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 11 to 12")
	diskConf["schema_version"] = 12

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	var dns yobj
	dns, ok = dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	const field = "querylog_interval"

	// Set the initial value from home.initConfig function.
	qlogIvl := 90
	qlogIvlVal, ok := dns[field]
	if ok {
		qlogIvl, ok = qlogIvlVal.(int)
		if !ok {
			return fmt.Errorf("unexpected type of %s: %T", field, qlogIvlVal)
		}
	}

	dns[field] = Duration{Duration: time.Duration(qlogIvl) * 24 * time.Hour}

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
