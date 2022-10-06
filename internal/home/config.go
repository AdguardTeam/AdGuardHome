package home

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/fastip"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/renameio/maybe"
	yaml "gopkg.in/yaml.v3"
)

// dataDir is the name of a directory under the working one to store some
// persistent data.
const dataDir = "data"

// logSettings are the logging settings part of the configuration file.
//
// TODO(a.garipov): Put them into a separate object.
type logSettings struct {
	// File is the path to the log file.  If empty, logs are written to stdout.
	// If "syslog", logs are written to syslog.
	File string `yaml:"log_file"`

	// MaxBackups is the maximum number of old log files to retain.
	//
	// NOTE: MaxAge may still cause them to get deleted.
	MaxBackups int `yaml:"log_max_backups"`

	// MaxSize is the maximum size of the log file before it gets rotated, in
	// megabytes.  The default value is 100 MB.
	MaxSize int `yaml:"log_max_size"`

	// MaxAge is the maximum duration for retaining old log files, in days.
	MaxAge int `yaml:"log_max_age"`

	// Compress determines, if the rotated log files should be compressed using
	// gzip.
	Compress bool `yaml:"log_compress"`

	// LocalTime determines, if the time used for formatting the timestamps in
	// is the computer's local time.
	LocalTime bool `yaml:"log_localtime"`

	// Verbose determines, if verbose (aka debug) logging is enabled.
	Verbose bool `yaml:"verbose"`
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

type clientsConfig struct {
	// Sources defines the set of sources to fetch the runtime clients from.
	Sources *clientSourcesConf `yaml:"runtime_sources"`
	// Persistent are the configured clients.
	Persistent []*clientObject `yaml:"persistent"`
}

// configuration is loaded from YAML
// field ordering is important -- yaml fields will mirror ordering from here
type configuration struct {
	// Raw file data to avoid re-reading of configuration file
	// It's reset after config is parsed
	fileData []byte

	BindHost     net.IP    `yaml:"bind_host"`      // BindHost is the IP address of the HTTP server to bind to
	BindPort     int       `yaml:"bind_port"`      // BindPort is the port the HTTP server
	BetaBindPort int       `yaml:"beta_bind_port"` // BetaBindPort is the port for new client
	Users        []webUser `yaml:"users"`          // Users that can access HTTP server
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

	// Filters reflects the filters from [filtering.Config].  It's cloned to the
	// config used in the filtering module at the startup.  Afterwards it's
	// cloned from the filtering module back here.
	//
	// TODO(e.burkov):  Move all the filtering configuration fields into the
	// only configuration subsection covering the changes with a single
	// migration.
	Filters          []filtering.FilterYAML `yaml:"filters"`
	WhitelistFilters []filtering.FilterYAML `yaml:"whitelist_filters"`
	UserRules        []string               `yaml:"user_rules"`

	DHCP *dhcpd.ServerConfig `yaml:"dhcp"`

	// Clients contains the YAML representations of the persistent clients.
	// This field is only used for reading and writing persistent client data.
	// Keep this field sorted to ensure consistent ordering.
	Clients *clientsConfig `yaml:"clients"`

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

	DnsfilterConf *filtering.Config `yaml:",inline"`

	// UpstreamTimeout is the timeout for querying upstream servers.
	UpstreamTimeout timeutil.Duration `yaml:"upstream_timeout"`

	// PrivateNets is the set of IP networks for which the private reverse DNS
	// resolver should be used.
	PrivateNets []string `yaml:"private_networks"`

	// UsePrivateRDNS defines if the PTR requests for unknown addresses from
	// locally-served networks should be resolved via private PTR resolvers.
	UsePrivateRDNS bool `yaml:"use_private_ptr_resolvers"`

	// LocalPTRResolvers is the slice of addresses to be used as upstreams
	// for PTR queries for locally-served networks.
	LocalPTRResolvers []string `yaml:"local_ptr_upstreams"`

	// ServeHTTP3 defines if HTTP/3 is be allowed for incoming requests.
	//
	// TODO(a.garipov): Add to the UI when HTTP/3 support is no longer
	// experimental.
	ServeHTTP3 bool `yaml:"serve_http3"`

	// UseHTTP3Upstreams defines if HTTP/3 is be allowed for DNS-over-HTTPS
	// upstreams.
	//
	// TODO(a.garipov): Add to the UI when HTTP/3 support is no longer
	// experimental.
	UseHTTP3Upstreams bool `yaml:"use_http3_upstreams"`
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
// TODO(a.garipov, e.burkov): This global is awful and must be removed.
var config = &configuration{
	BindPort:           3000,
	BetaBindPort:       0,
	BindHost:           net.IP{0, 0, 0, 0},
	AuthAttempts:       5,
	AuthBlockMin:       15,
	WebSessionTTLHours: 30 * 24,
	DNS: dnsConfig{
		BindHosts:           []net.IP{{0, 0, 0, 0}},
		Port:                defaultPortDNS,
		StatsInterval:       1,
		QueryLogEnabled:     true,
		QueryLogFileEnabled: true,
		QueryLogInterval:    timeutil.Duration{Duration: 90 * timeutil.Day},
		QueryLogMemSize:     1000,
		FilteringConfig: dnsforward.FilteringConfig{
			ProtectionEnabled:  true, // whether or not use any of filtering features
			BlockingMode:       dnsforward.BlockingModeDefault,
			BlockedResponseTTL: 10, // in seconds
			Ratelimit:          20,
			RefuseAny:          true,
			AllServers:         false,
			HandleDDR:          true,
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
		DnsfilterConf: &filtering.Config{
			SafeBrowsingCacheSize:      1 * 1024 * 1024,
			SafeSearchCacheSize:        1 * 1024 * 1024,
			ParentalCacheSize:          1 * 1024 * 1024,
			CacheTime:                  30,
			FilteringEnabled:           true,
			FiltersUpdateIntervalHours: 24,
		},
		UpstreamTimeout: timeutil.Duration{Duration: dnsforward.DefaultTimeout},
		UsePrivateRDNS:  true,
	},
	TLS: tlsConfigSettings{
		PortHTTPS:       defaultPortHTTPS,
		PortDNSOverTLS:  defaultPortTLS, // needs to be passed through to dnsproxy
		PortDNSOverQUIC: defaultPortQUIC,
	},
	Filters: []filtering.FilterYAML{{
		Filter:  filtering.Filter{ID: 1},
		Enabled: true,
		URL:     "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt",
		Name:    "AdGuard DNS filter",
	}, {
		Filter:  filtering.Filter{ID: 2},
		Enabled: false,
		URL:     "https://adaway.org/hosts.txt",
		Name:    "AdAway Default Blocklist",
	}},
	DHCP: &dhcpd.ServerConfig{
		LocalDomainName: "lan",
		Conf4: dhcpd.V4ServerConf{
			LeaseDuration: dhcpd.DefaultDHCPLeaseTTL,
			ICMPTimeout:   dhcpd.DefaultDHCPTimeoutICMP,
		},
		Conf6: dhcpd.V6ServerConf{
			LeaseDuration: dhcpd.DefaultDHCPLeaseTTL,
		},
	},
	Clients: &clientsConfig{
		Sources: &clientSourcesConf{
			WHOIS:     true,
			ARP:       true,
			RDNS:      true,
			DHCP:      true,
			HostsFile: true,
		},
	},
	logSettings: logSettings{
		Compress:   false,
		LocalTime:  false,
		MaxBackups: 0,
		MaxSize:    100,
		MaxAge:     3,
	},
	OSConfig:      &osConfig{},
	SchemaVersion: currentSchemaVersion,
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
func parseConfig() (err error) {
	var fileData []byte
	fileData, err = readConfigFile()
	if err != nil {
		return err
	}

	config.fileData = nil
	err = yaml.Unmarshal(fileData, &config)
	if err != nil {
		return err
	}

	tcpPorts := aghalg.UniqChecker[tcpPort]{}
	addPorts(tcpPorts, tcpPort(config.BindPort), tcpPort(config.BetaBindPort))

	udpPorts := aghalg.UniqChecker[udpPort]{}
	addPorts(udpPorts, udpPort(config.DNS.Port))

	if config.TLS.Enabled {
		addPorts(
			tcpPorts,
			tcpPort(config.TLS.PortHTTPS),
			tcpPort(config.TLS.PortDNSOverTLS),
			tcpPort(config.TLS.PortDNSCrypt),
		)

		// TODO(e.burkov):  Consider adding a udpPort with the same value when
		// we add support for HTTP/3 for web admin interface.
		addPorts(udpPorts, udpPort(config.TLS.PortDNSOverQUIC))
	}
	if err = tcpPorts.Validate(); err != nil {
		return fmt.Errorf("validating tcp ports: %w", err)
	} else if err = udpPorts.Validate(); err != nil {
		return fmt.Errorf("validating udp ports: %w", err)
	}

	if !filtering.ValidateUpdateIvl(config.DNS.DnsfilterConf.FiltersUpdateIntervalHours) {
		config.DNS.DnsfilterConf.FiltersUpdateIntervalHours = 24
	}

	if config.DNS.UpstreamTimeout.Duration == 0 {
		config.DNS.UpstreamTimeout = timeutil.Duration{Duration: dnsforward.DefaultTimeout}
	}

	return nil
}

// udpPort is the port number for UDP protocol.
type udpPort int

// tcpPort is the port number for TCP protocol.
type tcpPort int

// addPorts is a helper for ports validation that skips zero ports.
func addPorts[T tcpPort | udpPort](uc aghalg.UniqChecker[T], ports ...T) {
	for _, p := range ports {
		if p != 0 {
			uc.Add(p)
		}
	}
}

// readConfigFile reads configuration file contents.
func readConfigFile() (fileData []byte, err error) {
	if len(config.fileData) > 0 {
		return config.fileData, nil
	}

	name := config.getConfigFilename()
	log.Debug("reading config file: %s", name)

	// Do not wrap the error because it's informative enough as is.
	return os.ReadFile(name)
}

// Saves configuration to the YAML file and also saves the user filter contents to a file
func (c *configuration) write() (err error) {
	c.Lock()
	defer c.Unlock()

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

	if Context.filters != nil {
		Context.filters.WriteDiskConfig(config.DNS.DnsfilterConf)
		config.Filters = config.DNS.DnsfilterConf.Filters
		config.WhitelistFilters = config.DNS.DnsfilterConf.WhitelistFilters
		config.UserRules = config.DNS.DnsfilterConf.UserRules
	}

	if s := Context.dnsServer; s != nil {
		c := dnsforward.FilteringConfig{}
		s.WriteDiskConfig(&c)
		dns := &config.DNS
		dns.FilteringConfig = c
		dns.LocalPTRResolvers, config.Clients.Sources.RDNS, dns.UsePrivateRDNS = s.RDNSSettings()
	}

	if Context.dhcpServer != nil {
		Context.dhcpServer.WriteDiskConfig(config.DHCP)
	}

	config.Clients.Persistent = Context.clients.forConfig()

	configFile := config.getConfigFilename()
	log.Debug("writing config file %q", configFile)

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	err = enc.Encode(config)
	if err != nil {
		return fmt.Errorf("generating config file: %w", err)
	}

	err = maybe.WriteFile(configFile, buf.Bytes(), 0o644)
	if err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
