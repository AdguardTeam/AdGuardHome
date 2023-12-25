package home

import (
	"bytes"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/configmigrate"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/fastip"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/renameio/v2/maybe"
	yaml "gopkg.in/yaml.v3"
)

// dataDir is the name of a directory under the working one to store some
// persistent data.
const dataDir = "data"

// logSettings are the logging settings part of the configuration file.
type logSettings struct {
	// File is the path to the log file.  If empty, logs are written to stdout.
	// If "syslog", logs are written to syslog.
	File string `yaml:"file"`

	// MaxBackups is the maximum number of old log files to retain.
	//
	// NOTE: MaxAge may still cause them to get deleted.
	MaxBackups int `yaml:"max_backups"`

	// MaxSize is the maximum size of the log file before it gets rotated, in
	// megabytes.  The default value is 100 MB.
	MaxSize int `yaml:"max_size"`

	// MaxAge is the maximum duration for retaining old log files, in days.
	MaxAge int `yaml:"max_age"`

	// Compress determines, if the rotated log files should be compressed using
	// gzip.
	Compress bool `yaml:"compress"`

	// LocalTime determines, if the time used for formatting the timestamps in
	// is the computer's local time.
	LocalTime bool `yaml:"local_time"`

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
	Sources *clientSourcesConfig `yaml:"runtime_sources"`
	// Persistent are the configured clients.
	Persistent []*clientObject `yaml:"persistent"`
}

// clientSourceConfig is used to configure where the runtime clients will be
// obtained from.
type clientSourcesConfig struct {
	WHOIS     bool `yaml:"whois"`
	ARP       bool `yaml:"arp"`
	RDNS      bool `yaml:"rdns"`
	DHCP      bool `yaml:"dhcp"`
	HostsFile bool `yaml:"hosts"`
}

// configuration is loaded from YAML.
//
// Field ordering is important, YAML fields better not to be reordered, if it's
// not absolutely necessary.
type configuration struct {
	// Raw file data to avoid re-reading of configuration file
	// It's reset after config is parsed
	fileData []byte

	// HTTPConfig is the block with http conf.
	HTTPConfig httpConfig `yaml:"http"`
	// Users are the clients capable for accessing the web interface.
	Users []webUser `yaml:"users"`
	// AuthAttempts is the maximum number of failed login attempts a user
	// can do before being blocked.
	AuthAttempts uint `yaml:"auth_attempts"`
	// AuthBlockMin is the duration, in minutes, of the block of new login
	// attempts after AuthAttempts unsuccessful login attempts.
	AuthBlockMin uint `yaml:"block_auth_min"`
	// ProxyURL is the address of proxy server for the internal HTTP client.
	ProxyURL string `yaml:"http_proxy"`
	// Language is a two-letter ISO 639-1 language code.
	Language string `yaml:"language"`
	// Theme is a UI theme for current user.
	Theme Theme `yaml:"theme"`

	// TODO(a.garipov): Make DNS and the fields below pointers and validate
	// and/or reset on explicit nulling.
	DNS      dnsConfig         `yaml:"dns"`
	TLS      tlsConfigSettings `yaml:"tls"`
	QueryLog queryLogConfig    `yaml:"querylog"`
	Stats    statsConfig       `yaml:"statistics"`

	// Filters reflects the filters from [filtering.Config].  It's cloned to the
	// config used in the filtering module at the startup.  Afterwards it's
	// cloned from the filtering module back here.
	//
	// TODO(e.burkov):  Move all the filtering configuration fields into the
	// only configuration subsection covering the changes with a single
	// migration.  Also keep the blocked services in mind.
	Filters          []filtering.FilterYAML `yaml:"filters"`
	WhitelistFilters []filtering.FilterYAML `yaml:"whitelist_filters"`
	UserRules        []string               `yaml:"user_rules"`

	DHCP      *dhcpd.ServerConfig `yaml:"dhcp"`
	Filtering *filtering.Config   `yaml:"filtering"`

	// Clients contains the YAML representations of the persistent clients.
	// This field is only used for reading and writing persistent client data.
	// Keep this field sorted to ensure consistent ordering.
	Clients *clientsConfig `yaml:"clients"`

	// Log is a block with log configuration settings.
	Log logSettings `yaml:"log"`

	OSConfig *osConfig `yaml:"os"`

	sync.RWMutex `yaml:"-"`

	// SchemaVersion is the version of the configuration schema.  See
	// [configmigrate.LastSchemaVersion].
	SchemaVersion uint `yaml:"schema_version"`
}

// httpConfig is a block with HTTP configuration params.
//
// Field ordering is important, YAML fields better not to be reordered, if it's
// not absolutely necessary.
type httpConfig struct {
	// Pprof defines the profiling HTTP handler.
	Pprof *httpPprofConfig `yaml:"pprof"`

	// Address is the address to serve the web UI on.
	Address netip.AddrPort

	// SessionTTL for a web session.
	// An active session is automatically refreshed once a day.
	SessionTTL timeutil.Duration `yaml:"session_ttl"`
}

// httpPprofConfig is the block with pprof HTTP configuration.
type httpPprofConfig struct {
	// Port for the profiling handler.
	Port uint16 `yaml:"port"`

	// Enabled defines if the profiling handler is enabled.
	Enabled bool `yaml:"enabled"`
}

// dnsConfig is a block with DNS configuration params.
//
// Field ordering is important, YAML fields better not to be reordered, if it's
// not absolutely necessary.
type dnsConfig struct {
	BindHosts []netip.Addr `yaml:"bind_hosts"`
	Port      uint16       `yaml:"port"`

	// AnonymizeClientIP defines if clients' IP addresses should be anonymized
	// in query log and statistics.
	AnonymizeClientIP bool `yaml:"anonymize_client_ip"`

	// Config is the embed configuration with DNS params.
	//
	// TODO(a.garipov): Remove embed.
	dnsforward.Config `yaml:",inline"`

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

	// UseDNS64 defines if DNS64 should be used for incoming requests.
	UseDNS64 bool `yaml:"use_dns64"`

	// DNS64Prefixes is the list of NAT64 prefixes to be used for DNS64.
	DNS64Prefixes []netip.Prefix `yaml:"dns64_prefixes"`

	// ServeHTTP3 defines if HTTP/3 is allowed for incoming requests.
	//
	// TODO(a.garipov): Add to the UI when HTTP/3 support is no longer
	// experimental.
	ServeHTTP3 bool `yaml:"serve_http3"`

	// UseHTTP3Upstreams defines if HTTP/3 is allowed for DNS-over-HTTPS
	// upstreams.
	//
	// TODO(a.garipov): Add to the UI when HTTP/3 support is no longer
	// experimental.
	UseHTTP3Upstreams bool `yaml:"use_http3_upstreams"`

	// ServePlainDNS defines if plain DNS is allowed for incoming requests.
	ServePlainDNS bool `yaml:"serve_plain_dns"`
}

type tlsConfigSettings struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`                                 // Enabled is the encryption (DoT/DoH/HTTPS) status
	ServerName      string `yaml:"server_name" json:"server_name,omitempty"`               // ServerName is the hostname of your HTTPS/TLS server
	ForceHTTPS      bool   `yaml:"force_https" json:"force_https"`                         // ForceHTTPS: if true, forces HTTP->HTTPS redirect
	PortHTTPS       uint16 `yaml:"port_https" json:"port_https,omitempty"`                 // HTTPS port. If 0, HTTPS will be disabled
	PortDNSOverTLS  uint16 `yaml:"port_dns_over_tls" json:"port_dns_over_tls,omitempty"`   // DNS-over-TLS port. If 0, DoT will be disabled
	PortDNSOverQUIC uint16 `yaml:"port_dns_over_quic" json:"port_dns_over_quic,omitempty"` // DNS-over-QUIC port. If 0, DoQ will be disabled

	// PortDNSCrypt is the port for DNSCrypt requests.  If it's zero,
	// DNSCrypt is disabled.
	PortDNSCrypt uint16 `yaml:"port_dnscrypt" json:"port_dnscrypt"`
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

type queryLogConfig struct {
	// Ignored is the list of host names, which should not be written to log.
	// "." is considered to be the root domain.
	Ignored []string `yaml:"ignored"`

	// Interval is the interval for query log's files rotation.
	Interval timeutil.Duration `yaml:"interval"`

	// MemSize is the number of entries kept in memory before they are flushed
	// to disk.
	MemSize int `yaml:"size_memory"`

	// Enabled defines if the query log is enabled.
	Enabled bool `yaml:"enabled"`

	// FileEnabled defines, if the query log is written to the file.
	FileEnabled bool `yaml:"file_enabled"`
}

type statsConfig struct {
	// Ignored is the list of host names, which should not be counted.
	Ignored []string `yaml:"ignored"`

	// Interval is the retention interval for statistics.
	Interval timeutil.Duration `yaml:"interval"`

	// Enabled defines if the statistics are enabled.
	Enabled bool `yaml:"enabled"`
}

// Default block host constants.
const (
	defaultSafeBrowsingBlockHost = "standard-block.dns.adguard.com"
	defaultParentalBlockHost     = "family-block.dns.adguard.com"
)

// config is the global configuration structure.
//
// TODO(a.garipov, e.burkov): This global is awful and must be removed.
var config = &configuration{
	AuthAttempts: 5,
	AuthBlockMin: 15,
	HTTPConfig: httpConfig{
		Address:    netip.AddrPortFrom(netip.IPv4Unspecified(), 3000),
		SessionTTL: timeutil.Duration{Duration: 30 * timeutil.Day},
		Pprof: &httpPprofConfig{
			Enabled: false,
			Port:    6060,
		},
	},
	DNS: dnsConfig{
		BindHosts: []netip.Addr{netip.IPv4Unspecified()},
		Port:      defaultPortDNS,
		Config: dnsforward.Config{
			Ratelimit:              20,
			RatelimitSubnetLenIPv4: 24,
			RatelimitSubnetLenIPv6: 56,
			RefuseAny:              true,
			UpstreamMode:           dnsforward.UpstreamModeLoadBalance,
			HandleDDR:              true,
			FastestTimeout: timeutil.Duration{
				Duration: fastip.DefaultPingWaitTimeout,
			},

			TrustedProxies: []string{"127.0.0.0/8", "::1/128"},
			CacheSize:      4 * 1024 * 1024,

			EDNSClientSubnet: &dnsforward.EDNSClientSubnet{
				CustomIP:  netip.Addr{},
				Enabled:   false,
				UseCustom: false,
			},

			// set default maximum concurrent queries to 300
			// we introduced a default limit due to this:
			// https://github.com/AdguardTeam/AdGuardHome/issues/2015#issuecomment-674041912
			// was later increased to 300 due to https://github.com/AdguardTeam/AdGuardHome/issues/2257
			MaxGoroutines: 300,
		},
		UpstreamTimeout: timeutil.Duration{Duration: dnsforward.DefaultTimeout},
		UsePrivateRDNS:  true,
		ServePlainDNS:   true,
	},
	TLS: tlsConfigSettings{
		PortHTTPS:       defaultPortHTTPS,
		PortDNSOverTLS:  defaultPortTLS, // needs to be passed through to dnsproxy
		PortDNSOverQUIC: defaultPortQUIC,
	},
	QueryLog: queryLogConfig{
		Enabled:     true,
		FileEnabled: true,
		Interval:    timeutil.Duration{Duration: 90 * timeutil.Day},
		MemSize:     1000,
		Ignored:     []string{},
	},
	Stats: statsConfig{
		Enabled:  true,
		Interval: timeutil.Duration{Duration: 1 * timeutil.Day},
		Ignored:  []string{},
	},
	// NOTE: Keep these parameters in sync with the one put into
	// client/src/helpers/filters/filters.js by scripts/vetted-filters.
	//
	// TODO(a.garipov): Think of a way to make scripts/vetted-filters update
	// these as well if necessary.
	Filters: []filtering.FilterYAML{{
		Filter:  filtering.Filter{ID: 1},
		Enabled: true,
		URL:     "https://adguardteam.github.io/HostlistsRegistry/assets/filter_1.txt",
		Name:    "AdGuard DNS filter",
	}, {
		Filter:  filtering.Filter{ID: 2},
		Enabled: false,
		URL:     "https://adguardteam.github.io/HostlistsRegistry/assets/filter_2.txt",
		Name:    "AdAway Default Blocklist",
	}},
	Filtering: &filtering.Config{
		ProtectionEnabled:  true,
		BlockingMode:       filtering.BlockingModeDefault,
		BlockedResponseTTL: 10, // in seconds

		FilteringEnabled:           true,
		FiltersUpdateIntervalHours: 24,

		ParentalEnabled:     false,
		SafeBrowsingEnabled: false,

		SafeBrowsingCacheSize: 1 * 1024 * 1024,
		SafeSearchCacheSize:   1 * 1024 * 1024,
		ParentalCacheSize:     1 * 1024 * 1024,
		CacheTime:             30,

		SafeSearchConf: filtering.SafeSearchConfig{
			Enabled:    false,
			Bing:       true,
			DuckDuckGo: true,
			Google:     true,
			Pixabay:    true,
			Yandex:     true,
			YouTube:    true,
		},

		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{},
		},

		ParentalBlockHost:     defaultParentalBlockHost,
		SafeBrowsingBlockHost: defaultSafeBrowsingBlockHost,
	},
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
		Sources: &clientSourcesConfig{
			WHOIS:     true,
			ARP:       true,
			RDNS:      true,
			DHCP:      true,
			HostsFile: true,
		},
	},
	Log: logSettings{
		Compress:   false,
		LocalTime:  false,
		MaxBackups: 0,
		MaxSize:    100,
		MaxAge:     3,
	},
	OSConfig:      &osConfig{},
	SchemaVersion: configmigrate.LastSchemaVersion,
	Theme:         ThemeAuto,
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

// validateBindHosts returns error if any of binding hosts from configuration is
// not a valid IP address.
func validateBindHosts(conf *configuration) (err error) {
	if !conf.HTTPConfig.Address.IsValid() {
		return errors.Error("http.address is not a valid ip address")
	}

	for i, addr := range conf.DNS.BindHosts {
		if !addr.IsValid() {
			return fmt.Errorf("dns.bind_hosts at index %d is not a valid ip address", i)
		}
	}

	return nil
}

// parseConfig loads configuration from the YAML file, upgrading it if
// necessary.
func parseConfig() (err error) {
	// Do the upgrade if necessary.
	config.fileData, err = readConfigFile()
	if err != nil {
		return err
	}

	migrator := configmigrate.New(&configmigrate.Config{
		WorkingDir: Context.workDir,
	})

	var upgraded bool
	config.fileData, upgraded, err = migrator.Migrate(
		config.fileData,
		configmigrate.LastSchemaVersion,
	)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	} else if upgraded {
		err = maybe.WriteFile(config.getConfigFilename(), config.fileData, 0o644)
		if err != nil {
			return fmt.Errorf("writing new config: %w", err)
		}
	}

	err = yaml.Unmarshal(config.fileData, &config)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = validateConfig()
	if err != nil {
		return err
	}

	if config.DNS.UpstreamTimeout.Duration == 0 {
		config.DNS.UpstreamTimeout = timeutil.Duration{Duration: dnsforward.DefaultTimeout}
	}

	err = setContextTLSCipherIDs()
	if err != nil {
		return err
	}

	return nil
}

// validateConfig returns error if the configuration is invalid.
func validateConfig() (err error) {
	err = validateBindHosts(config)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	tcpPorts := aghalg.UniqChecker[tcpPort]{}
	addPorts(tcpPorts, tcpPort(config.HTTPConfig.Address.Port()))

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

	if !filtering.ValidateUpdateIvl(config.Filtering.FiltersUpdateIntervalHours) {
		config.Filtering.FiltersUpdateIntervalHours = 24
	}

	return nil
}

// udpPort is the port number for UDP protocol.
type udpPort uint16

// tcpPort is the port number for TCP protocol.
type tcpPort uint16

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
		config.Users = Context.auth.usersList()
	}

	if Context.tls != nil {
		tlsConf := tlsConfigSettings{}
		Context.tls.WriteDiskConfig(&tlsConf)
		config.TLS = tlsConf
	}

	if Context.stats != nil {
		statsConf := stats.Config{}
		Context.stats.WriteDiskConfig(&statsConf)
		config.Stats.Interval = timeutil.Duration{Duration: statsConf.Limit}
		config.Stats.Enabled = statsConf.Enabled
		config.Stats.Ignored = statsConf.Ignored.Values()
	}

	if Context.queryLog != nil {
		dc := querylog.Config{}
		Context.queryLog.WriteDiskConfig(&dc)
		config.DNS.AnonymizeClientIP = dc.AnonymizeClientIP
		config.QueryLog.Enabled = dc.Enabled
		config.QueryLog.FileEnabled = dc.FileEnabled
		config.QueryLog.Interval = timeutil.Duration{Duration: dc.RotationIvl}
		config.QueryLog.MemSize = dc.MemSize
		config.QueryLog.Ignored = dc.Ignored.Values()
	}

	if Context.filters != nil {
		Context.filters.WriteDiskConfig(config.Filtering)
		config.Filters = config.Filtering.Filters
		config.WhitelistFilters = config.Filtering.WhitelistFilters
		config.UserRules = config.Filtering.UserRules
	}

	if s := Context.dnsServer; s != nil {
		c := dnsforward.Config{}
		s.WriteDiskConfig(&c)
		dns := &config.DNS
		dns.Config = c

		dns.LocalPTRResolvers = s.LocalPTRResolvers()

		addrProcConf := s.AddrProcConfig()
		config.Clients.Sources.RDNS = addrProcConf.UseRDNS
		config.Clients.Sources.WHOIS = addrProcConf.UseWHOIS
		dns.UsePrivateRDNS = addrProcConf.UsePrivateRDNS
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

// setContextTLSCipherIDs sets the TLS cipher suite IDs to use.
func setContextTLSCipherIDs() (err error) {
	if len(config.TLS.OverrideTLSCiphers) == 0 {
		log.Info("tls: using default ciphers")

		Context.tlsCipherIDs = aghtls.SaferCipherSuites()

		return nil
	}

	log.Info("tls: overriding ciphers: %s", config.TLS.OverrideTLSCiphers)

	Context.tlsCipherIDs, err = aghtls.ParseCiphers(config.TLS.OverrideTLSCiphers)
	if err != nil {
		return fmt.Errorf("parsing override ciphers: %w", err)
	}

	return nil
}
