package home

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/dnsproxy/fastip"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/renameio/maybe"
	yaml "gopkg.in/yaml.v2"
)

const (
	dataDir   = "data"    // data storage
	filterDir = "filters" // cache location for downloaded filters, it's under DataDir
)

// logSettings
type logSettings struct {
	LogCompress   bool   `yaml:"log_compress"`    // Compress determines if the rotated log files should be compressed using gzip (default: false)
	LogLocalTime  bool   `yaml:"log_localtime"`   // If the time used for formatting the timestamps in is the computer's local time (default: false [UTC])
	LogMaxBackups int    `yaml:"log_max_backups"` // Maximum number of old log files to retain (MaxAge may still cause them to get deleted)
	LogMaxSize    int    `yaml:"log_max_size"`    // Maximum size in megabytes of the log file before it gets rotated (default 100 MB)
	LogMaxAge     int    `yaml:"log_max_age"`     // MaxAge is the maximum number of days to retain old log files
	LogFile       string `yaml:"log_file"`        // Path to the log file. If empty, write to stdout. If "syslog", writes to syslog
	Verbose       bool   `yaml:"verbose"`         // If true, verbose logging is enabled
}

// osConfig contains OS-related configuration.
type osConfig struct {
	// Group is the name of the group which AdGuard Home must switch to on
	// startup.  Empty string means no switching.
	Group string `yaml:"group"`
	// User is the name of the user which AdGuard Home must switch to on
	// startup.  Empty string means no switching.
	User string `yaml:"user"`
	// RlimitNoFile is the maximum number of opened fd's per process.  Zero
	// means use the default value.
	RlimitNoFile uint64 `yaml:"rlimit_nofile"`
}

// configuration is loaded from YAML
// field ordering is important -- yaml fields will mirror ordering from here
type configuration struct {
	// Raw file data to avoid re-reading of configuration file
	// It's reset after config is parsed
	fileData []byte

	BindHost     net.IP `yaml:"bind_host"`      // BindHost is the IP address of the HTTP server to bind to
	BindPort     int    `yaml:"bind_port"`      // BindPort is the port the HTTP server
	BetaBindPort int    `yaml:"beta_bind_port"` // BetaBindPort is the port for new client
	Users        []User `yaml:"users"`          // Users that can access HTTP server
	// AuthAttempts is the maximum number of failed login attempts a user
	// can do before being blocked.
	AuthAttempts uint `yaml:"auth_attempts"`
	// AuthBlockMin is the duration, in minutes, of the block of new login
	// attempts after AuthAttempts unsuccessful login attempts.
	AuthBlockMin uint   `yaml:"block_auth_min"`
	ProxyURL     string `yaml:"http_proxy"`  // Proxy address for our HTTP client
	Language     string `yaml:"language"`    // two-letter ISO 639-1 language code
	DebugPProf   bool   `yaml:"debug_pprof"` // Enable pprof HTTP server on port 6060

	// TTL for a web session (in hours)
	// An active session is automatically refreshed once a day.
	WebSessionTTLHours uint32 `yaml:"web_session_ttl"`

	DNS dnsConfig         `yaml:"dns"`
	TLS tlsConfigSettings `yaml:"tls"`

	Filters          []filter `yaml:"filters"`
	WhitelistFilters []filter `yaml:"whitelist_filters"`
	UserRules        []string `yaml:"user_rules"`

	DHCP dhcpd.ServerConfig `yaml:"dhcp"`

	// Note: this array is filled only before file read/write and then it's cleared
	Clients []clientObject `yaml:"clients"`

	logSettings `yaml:",inline"`

	OSConfig *osConfig `yaml:"os"`

	sync.RWMutex `yaml:"-"`

	SchemaVersion int `yaml:"schema_version"` // keeping last so that users will be less tempted to change it -- used when upgrading between versions
}

// field ordering is important -- yaml fields will mirror ordering from here
type dnsConfig struct {
	BindHosts []net.IP `yaml:"bind_hosts"`
	Port      int      `yaml:"port"`

	// time interval for statistics (in days)
	StatsInterval uint32 `yaml:"statistics_interval"`

	QueryLogEnabled     bool `yaml:"querylog_enabled"`      // if true, query log is enabled
	QueryLogFileEnabled bool `yaml:"querylog_file_enabled"` // if true, query log will be written to a file
	// QueryLogInterval is the interval for query log's files rotation.
	QueryLogInterval  timeutil.Duration `yaml:"querylog_interval"`
	QueryLogMemSize   uint32            `yaml:"querylog_size_memory"` // number of entries kept in memory before they are flushed to disk
	AnonymizeClientIP bool              `yaml:"anonymize_client_ip"`  // anonymize clients' IP addresses in logs and stats

	dnsforward.FilteringConfig `yaml:",inline"`

	FilteringEnabled           bool             `yaml:"filtering_enabled"`       // whether or not use filter lists
	FiltersUpdateIntervalHours uint32           `yaml:"filters_update_interval"` // time period to update filters (in hours)
	DnsfilterConf              filtering.Config `yaml:",inline"`

	// UpstreamTimeout is the timeout for querying upstream servers.
	UpstreamTimeout timeutil.Duration `yaml:"upstream_timeout"`

	// LocalDomainName is the domain name used for known internal hosts.
	// For example, a machine called "myhost" can be addressed as
	// "myhost.lan" when LocalDomainName is "lan".
	LocalDomainName string `yaml:"local_domain_name"`

	// ResolveClients enables and disables resolving clients with RDNS.
	ResolveClients bool `yaml:"resolve_clients"`

	// UsePrivateRDNS defines if the PTR requests for unknown addresses from
	// locally-served networks should be resolved via private PTR resolvers.
	UsePrivateRDNS bool `yaml:"use_private_ptr_resolvers"`

	// LocalPTRResolvers is the slice of addresses to be used as upstreams
	// for PTR queries for locally-served networks.
	LocalPTRResolvers []string `yaml:"local_ptr_upstreams"`
}

type tlsConfigSettings struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`                                 // Enabled is the encryption (DoT/DoH/HTTPS) status
	ServerName      string `yaml:"server_name" json:"server_name,omitempty"`               // ServerName is the hostname of your HTTPS/TLS server
	ForceHTTPS      bool   `yaml:"force_https" json:"force_https"`                         // ForceHTTPS: if true, forces HTTP->HTTPS redirect
	PortHTTPS       int    `yaml:"port_https" json:"port_https,omitempty"`                 // HTTPS port. If 0, HTTPS will be disabled
	PortDNSOverTLS  int    `yaml:"port_dns_over_tls" json:"port_dns_over_tls,omitempty"`   // DNS-over-TLS port. If 0, DoT will be disabled
	PortDNSOverQUIC int    `yaml:"port_dns_over_quic" json:"port_dns_over_quic,omitempty"` // DNS-over-QUIC port. If 0, DoQ will be disabled

	// PortDNSCrypt is the port for DNSCrypt requests.  If it's zero,
	// DNSCrypt is disabled.
	PortDNSCrypt int `yaml:"port_dnscrypt" json:"port_dnscrypt"`
	// DNSCryptConfigFile is the path to the DNSCrypt config file.  Must be
	// set if PortDNSCrypt is not zero.
	//
	// See https://github.com/AdguardTeam/dnsproxy and
	// https://github.com/ameshkov/dnscrypt.
	DNSCryptConfigFile string `yaml:"dnscrypt_config_file" json:"dnscrypt_config_file"`

	// Allow DoH queries via unencrypted HTTP (e.g. for reverse proxying)
	AllowUnencryptedDoH bool `yaml:"allow_unencrypted_doh" json:"allow_unencrypted_doh"`

	dnsforward.TLSConfig `yaml:",inline" json:",inline"`
}

// config is the global configuration structure.
//
// TODO(a.garipov, e.burkov): This global is afwul and must be removed.
var config = &configuration{
	BindPort:     3000,
	BetaBindPort: 0,
	BindHost:     net.IP{0, 0, 0, 0},
	AuthAttempts: 5,
	AuthBlockMin: 15,
	DNS: dnsConfig{
		BindHosts:     []net.IP{{0, 0, 0, 0}},
		Port:          defaultPortDNS,
		StatsInterval: 1,
		FilteringConfig: dnsforward.FilteringConfig{
			ProtectionEnabled:  true,      // whether or not use any of filtering features
			BlockingMode:       "default", // mode how to answer filtered requests
			BlockedResponseTTL: 10,        // in seconds
			Ratelimit:          20,
			RefuseAny:          true,
			AllServers:         false,
			FastestTimeout: timeutil.Duration{
				Duration: fastip.DefaultPingWaitTimeout,
			},

			TrustedProxies: []string{"127.0.0.0/8", "::1/128"},

			// set default maximum concurrent queries to 300
			// we introduced a default limit due to this:
			// https://github.com/AdguardTeam/AdGuardHome/issues/2015#issuecomment-674041912
			// was later increased to 300 due to https://github.com/AdguardTeam/AdGuardHome/issues/2257
			MaxGoroutines: 300,
		},
		FilteringEnabled:           true, // whether or not use filter lists
		FiltersUpdateIntervalHours: 24,
		UpstreamTimeout:            timeutil.Duration{Duration: dnsforward.DefaultTimeout},
		LocalDomainName:            "lan",
		ResolveClients:             true,
		UsePrivateRDNS:             true,
	},
	TLS: tlsConfigSettings{
		PortHTTPS:       defaultPortHTTPS,
		PortDNSOverTLS:  defaultPortTLS, // needs to be passed through to dnsproxy
		PortDNSOverQUIC: defaultPortQUIC,
	},
	logSettings: logSettings{
		LogCompress:   false,
		LogLocalTime:  false,
		LogMaxBackups: 0,
		LogMaxSize:    100,
		LogMaxAge:     3,
	},
	OSConfig:      &osConfig{},
	SchemaVersion: currentSchemaVersion,
}

// initConfig initializes default configuration for the current OS&ARCH
func initConfig() {
	config.WebSessionTTLHours = 30 * 24

	config.DNS.QueryLogEnabled = true
	config.DNS.QueryLogFileEnabled = true
	config.DNS.QueryLogInterval = timeutil.Duration{Duration: 90 * timeutil.Day}
	config.DNS.QueryLogMemSize = 1000

	config.DNS.CacheSize = 4 * 1024 * 1024
	config.DNS.DnsfilterConf.SafeBrowsingCacheSize = 1 * 1024 * 1024
	config.DNS.DnsfilterConf.SafeSearchCacheSize = 1 * 1024 * 1024
	config.DNS.DnsfilterConf.ParentalCacheSize = 1 * 1024 * 1024
	config.DNS.DnsfilterConf.CacheTime = 30
	config.Filters = defaultFilters()

	config.DHCP.Conf4.LeaseDuration = dhcpd.DefaultDHCPLeaseTTL
	config.DHCP.Conf4.ICMPTimeout = dhcpd.DefaultDHCPTimeoutICMP
	config.DHCP.Conf6.LeaseDuration = dhcpd.DefaultDHCPLeaseTTL

	if ch := version.Channel(); ch == version.ChannelEdge || ch == version.ChannelDevelopment {
		config.BetaBindPort = 3001
	}
}

// getConfigFilename returns path to the current config file
func (c *configuration) getConfigFilename() string {
	configFile, err := filepath.EvalSymlinks(Context.configFilename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Error("unexpected error while config file path evaluation: %s", err)
		}
		configFile = Context.configFilename
	}
	if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(Context.workDir, configFile)
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

	if !checkFiltersUpdateIntervalHours(config.DNS.FiltersUpdateIntervalHours) {
		config.DNS.FiltersUpdateIntervalHours = 24
	}

	if config.DNS.UpstreamTimeout.Duration == 0 {
		config.DNS.UpstreamTimeout = timeutil.Duration{Duration: dnsforward.DefaultTimeout}
	}

	return nil
}

// readConfigFile reads config file contents if it exists
func readConfigFile() ([]byte, error) {
	if len(config.fileData) != 0 {
		return config.fileData, nil
	}

	configFile := config.getConfigFilename()
	d, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file %s: %w", configFile, err)
	}
	return d, nil
}

// Saves configuration to the YAML file and also saves the user filter contents to a file
func (c *configuration) write() error {
	c.Lock()
	defer c.Unlock()

	Context.clients.WriteDiskConfig(&config.Clients)

	if Context.auth != nil {
		config.Users = Context.auth.GetUsers()
	}
	if Context.tls != nil {
		tlsConf := tlsConfigSettings{}
		Context.tls.WriteDiskConfig(&tlsConf)
		config.TLS = tlsConf
	}

	if Context.stats != nil {
		sdc := stats.DiskConfig{}
		Context.stats.WriteDiskConfig(&sdc)
		config.DNS.StatsInterval = sdc.Interval
	}

	if Context.queryLog != nil {
		dc := querylog.Config{}
		Context.queryLog.WriteDiskConfig(&dc)
		config.DNS.QueryLogEnabled = dc.Enabled
		config.DNS.QueryLogFileEnabled = dc.FileEnabled
		config.DNS.QueryLogInterval = timeutil.Duration{Duration: dc.RotationIvl}
		config.DNS.QueryLogMemSize = dc.MemSize
		config.DNS.AnonymizeClientIP = dc.AnonymizeClientIP
	}

	if Context.dnsFilter != nil {
		c := filtering.Config{}
		Context.dnsFilter.WriteDiskConfig(&c)
		config.DNS.DnsfilterConf = c
	}

	if s := Context.dnsServer; s != nil {
		c := dnsforward.FilteringConfig{}
		s.WriteDiskConfig(&c)
		dns := &config.DNS
		dns.FilteringConfig = c
		dns.LocalPTRResolvers,
			dns.ResolveClients,
			dns.UsePrivateRDNS = s.RDNSSettings()
	}

	if Context.dhcpServer != nil {
		c := dhcpd.ServerConfig{}
		Context.dhcpServer.WriteDiskConfig(&c)
		config.DHCP = c
	}

	configFile := config.getConfigFilename()
	log.Debug("Writing YAML file: %s", configFile)
	yamlText, err := yaml.Marshal(&config)
	config.Clients = nil
	if err != nil {
		log.Error("Couldn't generate YAML file: %s", err)

		return err
	}

	err = maybe.WriteFile(configFile, yamlText, 0o644)
	if err != nil {
		log.Error("Couldn't save YAML config: %s", err)

		return err
	}

	return nil
}
