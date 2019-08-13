package home

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/golibs/file"
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

type clientObject struct {
	Name                string `yaml:"name"`
	IP                  string `yaml:"ip"`
	MAC                 string `yaml:"mac"`
	UseGlobalSettings   bool   `yaml:"use_global_settings"`
	FilteringEnabled    bool   `yaml:"filtering_enabled"`
	ParentalEnabled     bool   `yaml:"parental_enabled"`
	SafeSearchEnabled   bool   `yaml:"safebrowsing_enabled"`
	SafeBrowsingEnabled bool   `yaml:"safesearch_enabled"`

	UseGlobalBlockedServices bool     `yaml:"use_global_blocked_services"`
	BlockedServices          []string `yaml:"blocked_services"`
}

type HTTPSServer struct {
	server     *http.Server
	cond       *sync.Cond // reacts to config.TLS.Enabled, PortHTTPS, CertificateChain and PrivateKey
	sync.Mutex            // protects config.TLS
	shutdown   bool       // if TRUE, don't restart the server
}

// configuration is loaded from YAML
// field ordering is important -- yaml fields will mirror ordering from here
type configuration struct {
	// Raw file data to avoid re-reading of configuration file
	// It's reset after config is parsed
	fileData []byte

	ourConfigFilename string // Config filename (can be overridden via the command line arguments)
	ourWorkingDir     string // Location of our directory, used to protect against CWD being somewhere else
	firstRun          bool   // if set to true, don't run any services except HTTP web inteface, and serve only first-run html
	pidFileName       string // PID file name.  Empty if no PID file was created.
	// runningAsService flag is set to true when options are passed from the service runner
	runningAsService bool
	disableUpdate    bool // If set, don't check for updates
	appSignalChannel chan os.Signal
	clients          clientsContainer
	controlLock      sync.Mutex
	transport        *http.Transport
	client           *http.Client

	// cached version.json to avoid hammering github.io for each page reload
	versionCheckJSON     []byte
	versionCheckLastTime time.Time

	dnsctx      dnsContext
	dnsServer   *dnsforward.Server
	dhcpServer  dhcpd.Server
	httpServer  *http.Server
	httpsServer HTTPSServer

	BindHost     string `yaml:"bind_host"`     // BindHost is the IP address of the HTTP server to bind to
	BindPort     int    `yaml:"bind_port"`     // BindPort is the port the HTTP server
	AuthName     string `yaml:"auth_name"`     // AuthName is the basic auth username
	AuthPass     string `yaml:"auth_pass"`     // AuthPass is the basic auth password
	Language     string `yaml:"language"`      // two-letter ISO 639-1 language code
	RlimitNoFile uint   `yaml:"rlimit_nofile"` // Maximum number of opened fd's per process (0: default)

	DNS       dnsConfig          `yaml:"dns"`
	TLS       tlsConfig          `yaml:"tls"`
	Filters   []filter           `yaml:"filters"`
	UserRules []string           `yaml:"user_rules"`
	DHCP      dhcpd.ServerConfig `yaml:"dhcp"`

	// Note: this array is filled only before file read/write and then it's cleared
	Clients []clientObject `yaml:"clients"`

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

var defaultDNS = []string{
	"https://1.1.1.1/dns-query",
	"https://1.0.0.1/dns-query",
}
var defaultBootstrap = []string{"1.1.1.1", "1.0.0.1"}

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
			ProtectionEnabled:  true,       // whether or not use any of dnsfilter features
			FilteringEnabled:   true,       // whether or not use filter lists
			BlockingMode:       "nxdomain", // mode how to answer filtered requests
			BlockedResponseTTL: 10,         // in seconds
			QueryLogEnabled:    true,
			Ratelimit:          20,
			RefuseAny:          true,
			BootstrapDNS:       defaultBootstrap,
			AllServers:         false,
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
		{Filter: dnsfilter.Filter{ID: 4}, Enabled: false, URL: "https://www.malwaredomainlist.com/hostslist/hosts.txt", Name: "MalwareDomainList.com Hosts List"},
	},
	DHCP: dhcpd.ServerConfig{
		LeaseDuration: 86400,
		ICMPTimeout:   1000,
	},
	SchemaVersion: currentSchemaVersion,
}

// initConfig initializes default configuration for the current OS&ARCH
func initConfig() {
	config.transport = &http.Transport{
		DialContext: customDialContext,
	}
	config.client = &http.Client{
		Timeout:   time.Minute * 5,
		Transport: config.transport,
	}

	if runtime.GOARCH == "mips" || runtime.GOARCH == "mipsle" {
		// Use plain DNS on MIPS, encryption is too slow
		defaultDNS = []string{"1.1.1.1", "1.0.0.1"}
		// also change the default config
		config.DNS.UpstreamDNS = defaultDNS
	}
}

// getConfigFilename returns path to the current config file
func (c *configuration) getConfigFilename() string {
	configFile, err := filepath.EvalSymlinks(config.ourConfigFilename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("unexpected error while config file path evaluation: %s", err)
		}
		configFile = config.ourConfigFilename
	}
	if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(config.ourWorkingDir, configFile)
	}
	return configFile
}

// getLogSettings reads logging settings from the config file.
// we do it in a separate method in order to configure logger before the actual configuration is parsed and applied.
func getLogSettings() logSettings {
	l := logSettings{}
	yamlFile, err := readConfigFile()
	if err != nil {
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
		return err
	}
	config.fileData = nil
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Error("Couldn't parse config file: %s", err)
		return err
	}

	for _, cy := range config.Clients {
		cli := Client{
			Name:                cy.Name,
			IP:                  cy.IP,
			MAC:                 cy.MAC,
			UseOwnSettings:      !cy.UseGlobalSettings,
			FilteringEnabled:    cy.FilteringEnabled,
			ParentalEnabled:     cy.ParentalEnabled,
			SafeSearchEnabled:   cy.SafeSearchEnabled,
			SafeBrowsingEnabled: cy.SafeBrowsingEnabled,

			UseOwnBlockedServices: !cy.UseGlobalBlockedServices,
			BlockedServices:       cy.BlockedServices,
		}
		_, err = config.clients.Add(cli)
		if err != nil {
			log.Tracef("clientAdd: %s", err)
		}
	}
	config.Clients = nil

	status := tlsConfigStatus{}
	if !tlsLoadConfig(&config.TLS, &status) {
		log.Error("%s", status.WarningValidation)
		return err
	}

	// Deduplicate filters
	deduplicateFilters()

	updateUniqueFilterID(config.Filters)

	return nil
}

// readConfigFile reads config file contents if it exists
func readConfigFile() ([]byte, error) {
	if len(config.fileData) != 0 {
		return config.fileData, nil
	}

	configFile := config.getConfigFilename()
	d, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Error("Couldn't read config file %s: %s", configFile, err)
		return nil, err
	}
	return d, nil
}

// Saves configuration to the YAML file and also saves the user filter contents to a file
func (c *configuration) write() error {
	c.Lock()
	defer c.Unlock()

	clientsList := config.clients.GetList()
	for _, cli := range clientsList {
		ip := cli.IP
		if len(cli.MAC) != 0 {
			ip = ""
		}
		cy := clientObject{
			Name:                cli.Name,
			IP:                  ip,
			MAC:                 cli.MAC,
			UseGlobalSettings:   !cli.UseOwnSettings,
			FilteringEnabled:    cli.FilteringEnabled,
			ParentalEnabled:     cli.ParentalEnabled,
			SafeSearchEnabled:   cli.SafeSearchEnabled,
			SafeBrowsingEnabled: cli.SafeBrowsingEnabled,

			UseGlobalBlockedServices: !cli.UseOwnBlockedServices,
			BlockedServices:          cli.BlockedServices,
		}
		config.Clients = append(config.Clients, cy)
	}

	configFile := config.getConfigFilename()
	log.Debug("Writing YAML file: %s", configFile)
	yamlText, err := yaml.Marshal(&config)
	config.Clients = nil
	if err != nil {
		log.Error("Couldn't generate YAML file: %s", err)
		return err
	}
	err = file.SafeWrite(configFile, yamlText)
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
