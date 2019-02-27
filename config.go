package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/golibs/log"
	yaml "gopkg.in/yaml.v2"
)

const (
	dataDir   = "data"    // data storage
	filterDir = "filters" // cache location for downloaded filters, it's under DataDir
)

// logSettings
type logSettings struct {
	LogFile string `yaml:"log_file"` // Path to the log file. If empty, write to stdout. If "syslog", writes to syslog
	Verbose bool   `yaml:"verbose"`  // If true, verbose logging is enabled
}

// configuration is loaded from YAML
// field ordering is important -- yaml fields will mirror ordering from here
type configuration struct {
	ourConfigFilename string // Config filename (can be overridden via the command line arguments)
	ourWorkingDir     string // Location of our directory, used to protect against CWD being somewhere else
	firstRun          bool   // if set to true, don't run any services except HTTP web inteface, and serve only first-run html

	BindHost  string             `yaml:"bind_host"` // BindHost is the IP address of the HTTP server to bind to
	BindPort  int                `yaml:"bind_port"` // BindPort is the port the HTTP server
	AuthName  string             `yaml:"auth_name"` // AuthName is the basic auth username
	AuthPass  string             `yaml:"auth_pass"` // AuthPass is the basic auth password
	Language  string             `yaml:"language"`  // two-letter ISO 639-1 language code
	DNS       dnsConfig          `yaml:"dns"`
	TLS       tlsConfig          `yaml:"tls"`
	Filters   []filter           `yaml:"filters"`
	UserRules []string           `yaml:"user_rules"`
	DHCP      dhcpd.ServerConfig `yaml:"dhcp"`

	logSettings `yaml:",inline"`

	sync.RWMutex `yaml:"-"`

	SchemaVersion int `yaml:"schema_version"` // keeping last so that users will be less tempted to change it -- used when upgrading between versions
}

// field ordering is important -- yaml fields will mirror ordering from here
type dnsConfig struct {
	BindHost string `yaml:"bind_host"`
	Port     int    `yaml:"port"`

	dnsforward.FilteringConfig `yaml:",inline"`

	UpstreamDNS []string `yaml:"upstream_dns"`
}

var defaultDNS = []string{"tls://1.1.1.1", "tls://1.0.0.1"}

type tlsConfigSettings struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`                               // Enabled is the encryption (DOT/DOH/HTTPS) status
	ServerName     string `yaml:"server_name" json:"server_name,omitempty"`             // ServerName is the hostname of your HTTPS/TLS server
	ForceHTTPS     bool   `yaml:"force_https" json:"force_https,omitempty"`             // ForceHTTPS: if true, forces HTTP->HTTPS redirect
	PortHTTPS      int    `yaml:"port_https" json:"port_https,omitempty"`               // HTTPS port. If 0, HTTPS will be disabled
	PortDNSOverTLS int    `yaml:"port_dns_over_tls" json:"port_dns_over_tls,omitempty"` // DNS-over-TLS port. If 0, DOT will be disabled

	dnsforward.TLSConfig `yaml:",inline" json:",inline"`
}

// field ordering is not important -- these are for API and are recalculated on each run
type tlsConfigStatus struct {
	ValidCert  bool      `yaml:"-" json:"valid_cert"`           // ValidCert is true if the specified certificates chain is a valid chain of X509 certificates
	ValidChain bool      `yaml:"-" json:"valid_chain"`          // ValidChain is true if the specified certificates chain is verified and issued by a known CA
	Subject    string    `yaml:"-" json:"subject,omitempty"`    // Subject is the subject of the first certificate in the chain
	Issuer     string    `yaml:"-" json:"issuer,omitempty"`     // Issuer is the issuer of the first certificate in the chain
	NotBefore  time.Time `yaml:"-" json:"not_before,omitempty"` // NotBefore is the NotBefore field of the first certificate in the chain
	NotAfter   time.Time `yaml:"-" json:"not_after,omitempty"`  // NotAfter is the NotAfter field of the first certificate in the chain
	DNSNames   []string  `yaml:"-" json:"dns_names"`            // DNSNames is the value of SubjectAltNames field of the first certificate in the chain

	// key status
	ValidKey bool   `yaml:"-" json:"valid_key"`          // ValidKey is true if the key is a valid private key
	KeyType  string `yaml:"-" json:"key_type,omitempty"` // KeyType is one of RSA or ECDSA

	// is usable? set by validator
	ValidPair bool `yaml:"-" json:"valid_pair"` // ValidPair is true if both certificate and private key are correct

	// warnings
	WarningValidation string `yaml:"-" json:"warning_validation,omitempty"` // WarningValidation is a validation warning message with the issue description
}

// field ordering is important -- yaml fields will mirror ordering from here
type tlsConfig struct {
	tlsConfigSettings `yaml:",inline" json:",inline"`
	tlsConfigStatus   `yaml:"-" json:",inline"`
}

// initialize to default values, will be changed later when reading config or parsing command line
var config = configuration{
	ourConfigFilename: "AdGuardHome.yaml",
	BindPort:          3000,
	BindHost:          "0.0.0.0",
	DNS: dnsConfig{
		BindHost: "0.0.0.0",
		Port:     53,
		FilteringConfig: dnsforward.FilteringConfig{
			ProtectionEnabled:  true, // whether or not use any of dnsfilter features
			FilteringEnabled:   true, // whether or not use filter lists
			BlockedResponseTTL: 10,   // in seconds
			QueryLogEnabled:    true,
			Ratelimit:          20,
			RefuseAny:          true,
			BootstrapDNS:       "8.8.8.8:53",
		},
		UpstreamDNS: defaultDNS,
	},
	TLS: tlsConfig{
		tlsConfigSettings: tlsConfigSettings{
			PortHTTPS:      443,
			PortDNSOverTLS: 853, // needs to be passed through to dnsproxy
		},
	},
	Filters: []filter{
		{Filter: dnsfilter.Filter{ID: 1}, Enabled: true, URL: "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt", Name: "AdGuard Simplified Domain Names filter"},
		{Filter: dnsfilter.Filter{ID: 2}, Enabled: false, URL: "https://adaway.org/hosts.txt", Name: "AdAway"},
		{Filter: dnsfilter.Filter{ID: 3}, Enabled: false, URL: "https://hosts-file.net/ad_servers.txt", Name: "hpHosts - Ad and Tracking servers only"},
		{Filter: dnsfilter.Filter{ID: 4}, Enabled: false, URL: "http://www.malwaredomainlist.com/hostslist/hosts.txt", Name: "MalwareDomainList.com Hosts List"},
	},
	SchemaVersion: currentSchemaVersion,
}

// getConfigFilename returns path to the current config file
func (c *configuration) getConfigFilename() string {
	configFile := config.ourConfigFilename
	if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(config.ourWorkingDir, config.ourConfigFilename)
	}
	return configFile
}

// getLogSettings reads logging settings from the config file.
// we do it in a separate method in order to configure logger before the actual configuration is parsed and applied.
func getLogSettings() logSettings {
	l := logSettings{}
	yamlFile, err := readConfigFile()
	if err != nil || yamlFile == nil {
		return l
	}
	err = yaml.Unmarshal(yamlFile, &l)
	if err != nil {
		log.Error("Couldn't get logging settings from the configuration: %s", err)
	}
	return l
}

// parseConfig loads configuration from the YAML file
func parseConfig() error {
	configFile := config.getConfigFilename()
	log.Debug("Reading config file: %s", configFile)
	yamlFile, err := readConfigFile()
	if err != nil {
		log.Error("Couldn't read config file: %s", err)
		return err
	}
	if yamlFile == nil {
		log.Error("YAML file doesn't exist, skipping it")
		return nil
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Error("Couldn't parse config file: %s", err)
		return err
	}

	// Deduplicate filters
	deduplicateFilters()

	updateUniqueFilterID(config.Filters)

	return nil
}

// readConfigFile reads config file contents if it exists
func readConfigFile() ([]byte, error) {
	configFile := config.getConfigFilename()
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return nil, nil
	}
	return ioutil.ReadFile(configFile)
}

// Saves configuration to the YAML file and also saves the user filter contents to a file
func (c *configuration) write() error {
	c.Lock()
	defer c.Unlock()
	if config.firstRun {
		log.Debug("Silently refusing to write config because first run and not configured yet")
		return nil
	}
	configFile := config.getConfigFilename()
	log.Debug("Writing YAML file: %s", configFile)
	yamlText, err := yaml.Marshal(&config)
	if err != nil {
		log.Error("Couldn't generate YAML file: %s", err)
		return err
	}
	err = safeWriteFile(configFile, yamlText)
	if err != nil {
		log.Error("Couldn't save YAML config: %s", err)
		return err
	}

	return nil
}

func writeAllConfigs() error {
	err := config.write()
	if err != nil {
		log.Error("Couldn't write config: %s", err)
		return err
	}

	userFilter := userFilter()
	err = userFilter.save()
	if err != nil {
		log.Error("Couldn't save the user filter: %s", err)
		return err
	}

	return nil
}
