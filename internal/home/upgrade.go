package home

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/renameio/maybe"
	"golang.org/x/crypto/bcrypt"
	yaml "gopkg.in/yaml.v3"
)

// currentSchemaVersion is the current schema version.
const currentSchemaVersion = 20

// These aliases are provided for convenience.
type (
	yarr = []any
	yobj = map[string]any
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
		log.Printf("parsing config file for upgrade: %s", err)

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
		upgradeSchema12to13,
		upgradeSchema13to14,
		upgradeSchema14to15,
		upgradeSchema15to16,
		upgradeSchema16to17,
		upgradeSchema17to18,
		upgradeSchema18to19,
		upgradeSchema19to20,
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

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	err = enc.Encode(diskConf)
	if err != nil {
		return fmt.Errorf("generating new config: %w", err)
	}

	config.fileData = buf.Bytes()
	confFile := config.getConfigFilename()
	err = maybe.WriteFile(confFile, config.fileData, 0o644)
	if err != nil {
		return fmt.Errorf("writing new config: %w", err)
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

	// Convert any to yobj
	newDNSConfig := make(yobj)

	switch v := dnsConfig.(type) {
	case yobj:
		for k, v := range v {
			newDNSConfig[fmt.Sprint(k)] = v
		}
	default:
		return fmt.Errorf("unexpected type of dns: %T", dnsConfig)
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
	case []any:

		for i := range arr {
			switch c := arr[i].(type) {

			case map[any]any:
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
//   - name: "..."
//     password: "..."
//
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
	u := webUser{
		Name:         nameStr,
		PasswordHash: string(hash),
	}
	users := []webUser{u}
	diskConf["users"] = users
	return nil
}

// clients:
// ...
//
//	ip: 127.0.0.1
//	mac: ...
//
// ->
//
// clients:
// ...
//
//	ids:
//	- 127.0.0.1
//	- ...
func upgradeSchema5to6(diskConf yobj) error {
	log.Printf("%s(): called", funcName())

	diskConf["schema_version"] = 6

	clients, ok := diskConf["clients"]
	if !ok {
		return nil
	}

	switch arr := clients.(type) {
	case []any:
		for i := range arr {
			switch c := arr[i].(type) {
			case map[any]any:
				var ipVal any
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

				var macVal any
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
//
//	enabled: false
//	interface_name: vboxnet0
//	gateway_ip: 192.168.56.1
//	...
//
// ->
//
// dhcp:
//
//	enabled: false
//	interface_name: vboxnet0
//	dhcpv4:
//	  gateway_ip: 192.168.56.1
//	  ...
func upgradeSchema6to7(diskConf yobj) error {
	log.Printf("Upgrade yaml: 6 to 7")

	diskConf["schema_version"] = 7

	dhcpVal, ok := diskConf["dhcp"]
	if !ok {
		return nil
	}

	switch dhcp := dhcpVal.(type) {
	case map[any]any:
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
//	# BEFORE:
//	'dns':
//	  'bind_host': '127.0.0.1'
//
//	# AFTER:
//	'dns':
//	  'bind_hosts':
//	  - '127.0.0.1'
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
//	# BEFORE:
//	'dns':
//	  'autohost_tld': 'lan'
//
//	# AFTER:
//	'dns':
//	  'local_domain_name': 'lan'
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
	host, err = netutil.SplitHost(upsURL.Host)
	if err != nil || host != upsURL.Host {
		return ups
	}

	upsURL.Host = strings.Join([]string{host, strconv.Itoa(port)}, ":")

	return doms + upsURL.String()
}

// upgradeSchema9to10 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'upstream_dns':
//	   - 'quic://some-upstream.com'
//
//	# AFTER:
//	'dns':
//	  'upstream_dns':
//	   - 'quic://some-upstream.com:784'
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
//	# BEFORE:
//	'rlimit_nofile': 42
//
//	# AFTER:
//	'os':
//	  'group': ''
//	  'rlimit_nofile': 42
//	  'user': ''
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
//	# BEFORE:
//	'querylog_interval': 90
//
//	# AFTER:
//	'querylog_interval': '2160h'
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

	dns[field] = timeutil.Duration{Duration: time.Duration(qlogIvl) * timeutil.Day}

	return nil
}

// upgradeSchema12to13 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  # …
//	  'local_domain_name': 'lan'
//
//	# AFTER:
//	'dhcp':
//	  # …
//	  'local_domain_name': 'lan'
func upgradeSchema12to13(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 12 to 13")
	diskConf["schema_version"] = 13

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	var dns yobj
	dns, ok = dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T; want: %T", dnsVal, dns)
	}

	dhcpVal, ok := diskConf["dhcp"]
	if !ok {
		return nil
	}

	var dhcp yobj
	dhcp, ok = dhcpVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dhcp: %T", dhcpVal)
	}

	const field = "local_domain_name"

	dhcp[field] = dns[field]
	delete(dns, field)

	return nil
}

// upgradeSchema13to14 performs the following changes:
//
//	# BEFORE:
//	'clients':
//	- 'name': 'client-name'
//	  # …
//
//	# AFTER:
//	'clients':
//	  'persistent':
//	  - 'name': 'client-name'
//	    # …
//	  'runtime_sources':
//	    'whois': true
//	    'arp': true
//	    'rdns': true
//	    'dhcp': true
//	    'hosts': true
func upgradeSchema13to14(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 13 to 14")
	diskConf["schema_version"] = 14

	clientsVal, ok := diskConf["clients"]
	if !ok {
		clientsVal = yarr{}
	}

	var rdnsSrc bool
	if dnsVal, dok := diskConf["dns"]; dok {
		var dnsSettings yobj
		dnsSettings, ok = dnsVal.(yobj)
		if !ok {
			return fmt.Errorf("unexpected type of dns: %T", dnsVal)
		}

		var rdnsSrcVal any
		rdnsSrcVal, ok = dnsSettings["resolve_clients"]
		if ok {
			rdnsSrc, ok = rdnsSrcVal.(bool)
			if !ok {
				return fmt.Errorf("unexpected type of resolve_clients: %T", rdnsSrcVal)
			}

			delete(dnsSettings, "resolve_clients")
		}
	}

	diskConf["clients"] = yobj{
		"persistent": clientsVal,
		"runtime_sources": &clientSourcesConfig{
			WHOIS:     true,
			ARP:       true,
			RDNS:      rdnsSrc,
			DHCP:      true,
			HostsFile: true,
		},
	}

	return nil
}

// upgradeSchema14to15 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'querylog_enabled': true
//	  'querylog_file_enabled': true
//	  'querylog_interval': '2160h'
//	  'querylog_size_memory': 1000
//
//	# AFTER:
//	'querylog':
//	  'enabled': true
//	  'file_enabled': true
//	  'interval': '2160h'
//	  'size_memory': 1000
//	  'ignored': []
func upgradeSchema14to15(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 14 to 15")
	diskConf["schema_version"] = 15

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	dns, ok := dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	type temp struct {
		val  any
		from string
		to   string
	}
	replaces := []temp{
		{from: "querylog_enabled", to: "enabled", val: true},
		{from: "querylog_file_enabled", to: "file_enabled", val: true},
		{from: "querylog_interval", to: "interval", val: "2160h"},
		{from: "querylog_size_memory", to: "size_memory", val: 1000},
	}
	qlog := map[string]any{
		"ignored": []any{},
	}
	for _, r := range replaces {
		v, has := dns[r.from]
		if !has {
			v = r.val
		}
		delete(dns, r.from)
		qlog[r.to] = v
	}
	diskConf["querylog"] = qlog

	return nil
}

// upgradeSchema15to16 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'statistics_interval': 1
//
//	# AFTER:
//	'statistics':
//	  'enabled': true
//	  'interval': 1
//	  'ignored': []
//
// If statistics were disabled:
//
//	# BEFORE:
//	'dns':
//	  'statistics_interval': 0
//
//	# AFTER:
//	'statistics':
//	  'enabled': false
//	  'interval': 1
//	  'ignored': []
func upgradeSchema15to16(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 15 to 16")
	diskConf["schema_version"] = 16

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	dns, ok := dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	stats := map[string]any{
		"enabled":  true,
		"interval": 1,
		"ignored":  []any{},
	}

	const field = "statistics_interval"
	statsIvlVal, has := dns[field]
	if has {
		var statsIvl int
		statsIvl, ok = statsIvlVal.(int)
		if !ok {
			return fmt.Errorf("unexpected type of dns.statistics_interval: %T", statsIvlVal)
		}

		if statsIvl == 0 {
			// Set the interval to the default value of one day to make sure
			// that it passes the validations.
			stats["interval"] = 1
			stats["enabled"] = false
		} else {
			stats["interval"] = statsIvl
			stats["enabled"] = true
		}
	}
	delete(dns, field)

	diskConf["statistics"] = stats

	return nil
}

// upgradeSchema16to17 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'edns_client_subnet': false
//
//	# AFTER:
//	'dns':
//	  'edns_client_subnet':
//	    'enabled': false
//	    'use_custom': false
//	    'custom_ip': ""
func upgradeSchema16to17(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 16 to 17")
	diskConf["schema_version"] = 17

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	dns, ok := dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	const field = "edns_client_subnet"

	dns[field] = map[string]any{
		"enabled":    dns[field] == true,
		"use_custom": false,
		"custom_ip":  "",
	}

	return nil
}

// upgradeSchema17to18 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'safesearch_enabled': true
//
//	# AFTER:
//	'dns':
//	  'safe_search':
//	    'enabled': true
//	    'bing': true
//	    'duckduckgo': true
//	    'google': true
//	    'pixabay': true
//	    'yandex': true
//	    'youtube': true
func upgradeSchema17to18(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 17 to 18")
	diskConf["schema_version"] = 18

	dnsVal, ok := diskConf["dns"]
	if !ok {
		return nil
	}

	dns, ok := dnsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of dns: %T", dnsVal)
	}

	safeSearch := yobj{
		"enabled":    true,
		"bing":       true,
		"duckduckgo": true,
		"google":     true,
		"pixabay":    true,
		"yandex":     true,
		"youtube":    true,
	}

	const safeSearchKey = "safesearch_enabled"

	v, has := dns[safeSearchKey]
	if has {
		safeSearch["enabled"] = v
	}
	delete(dns, safeSearchKey)

	dns["safe_search"] = safeSearch

	return nil
}

// upgradeSchema18to19 performs the following changes:
//
//	# BEFORE:
//	'clients':
//	  'persistent':
//	  - 'name': 'client-name'
//	    'safesearch_enabled': true
//
//	# AFTER:
//	'clients':
//	  'persistent':
//	  - 'name': 'client-name'
//	    'safe_search':
//	      'enabled': true
//		  'bing': true
//		  'duckduckgo': true
//		  'google': true
//		  'pixabay': true
//		  'yandex': true
//		  'youtube': true
func upgradeSchema18to19(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 18 to 19")
	diskConf["schema_version"] = 19

	clientsVal, ok := diskConf["clients"]
	if !ok {
		return nil
	}

	clients, ok := clientsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of clients: %T", clientsVal)
	}

	persistent, ok := clients["persistent"].([]yobj)
	if !ok {
		return nil
	}

	const safeSearchKey = "safesearch_enabled"

	for i := range persistent {
		c := persistent[i]

		safeSearch := yobj{
			"enabled":    true,
			"bing":       true,
			"duckduckgo": true,
			"google":     true,
			"pixabay":    true,
			"yandex":     true,
			"youtube":    true,
		}

		v, has := c[safeSearchKey]
		if has {
			safeSearch["enabled"] = v
		}
		delete(c, safeSearchKey)

		c["safe_search"] = safeSearch
	}

	return nil
}

// upgradeSchema19to20 performs the following changes:
//
//	# BEFORE:
//	'statistics':
//	  'interval': 1
//
//	# AFTER:
//	'statistics':
//	  'interval': 24h
func upgradeSchema19to20(diskConf yobj) (err error) {
	log.Printf("Upgrade yaml: 19 to 20")
	diskConf["schema_version"] = 20

	statsVal, ok := diskConf["statistics"]
	if !ok {
		return nil
	}

	var stats yobj
	stats, ok = statsVal.(yobj)
	if !ok {
		return fmt.Errorf("unexpected type of stats: %T", statsVal)
	}

	const field = "interval"

	// Set the initial value from the global configuration structure.
	statsIvl := 1
	statsIvlVal, ok := stats[field]
	if ok {
		statsIvl, ok = statsIvlVal.(int)
		if !ok {
			return fmt.Errorf("unexpected type of %s: %T", field, statsIvlVal)
		}

		// The initial version of upgradeSchema16to17 did not set the zero
		// interval to a non-zero one.  So, reset it now.
		if statsIvl == 0 {
			statsIvl = 1
		}
	}

	stats[field] = timeutil.Duration{Duration: time.Duration(statsIvl) * timeutil.Day}

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
