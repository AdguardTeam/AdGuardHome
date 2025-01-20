package home

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/ameshkov/dnscrypt/v2"
	yaml "gopkg.in/yaml.v3"
)

// Default listening ports.
const (
	defaultPortDNS   uint16 = 53
	defaultPortHTTP  uint16 = 80
	defaultPortHTTPS uint16 = 443
	defaultPortQUIC  uint16 = 853
	defaultPortTLS   uint16 = 853
)

// Called by other modules when configuration is changed
func onConfigModified() {
	err := config.write()
	if err != nil {
		log.Error("writing config: %s", err)
	}
}

// initDNS updates all the fields of the [Context] needed to initialize the DNS
// server and initializes it at last.  It also must not be called unless
// [config] and [Context] are initialized.  baseLogger must not be nil.
func initDNS(baseLogger *slog.Logger, statsDir, querylogDir string) (err error) {
	anonymizer := config.anonymizer()

	statsConf := stats.Config{
		Logger:            baseLogger.With(slogutil.KeyPrefix, "stats"),
		Filename:          filepath.Join(statsDir, "stats.db"),
		Limit:             time.Duration(config.Stats.Interval),
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
		Enabled:           config.Stats.Enabled,
		ShouldCountClient: Context.clients.shouldCountClient,
	}

	engine, err := aghnet.NewIgnoreEngine(config.Stats.Ignored)
	if err != nil {
		return fmt.Errorf("statistics: ignored list: %w", err)
	}

	statsConf.Ignored = engine
	Context.stats, err = stats.New(statsConf)
	if err != nil {
		return fmt.Errorf("init stats: %w", err)
	}

	conf := querylog.Config{
		Logger:            baseLogger.With(slogutil.KeyPrefix, "querylog"),
		Anonymizer:        anonymizer,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
		FindClient:        Context.clients.findMultiple,
		BaseDir:           querylogDir,
		AnonymizeClientIP: config.DNS.AnonymizeClientIP,
		RotationIvl:       time.Duration(config.QueryLog.Interval),
		MemSize:           config.QueryLog.MemSize,
		Enabled:           config.QueryLog.Enabled,
		FileEnabled:       config.QueryLog.FileEnabled,
	}

	engine, err = aghnet.NewIgnoreEngine(config.QueryLog.Ignored)
	if err != nil {
		return fmt.Errorf("querylog: ignored list: %w", err)
	}

	conf.Ignored = engine
	Context.queryLog, err = querylog.New(conf)
	if err != nil {
		return fmt.Errorf("init querylog: %w", err)
	}

	Context.filters, err = filtering.New(config.Filtering, nil)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	tlsConf := &tlsConfigSettings{}
	Context.tls.WriteDiskConfig(tlsConf)

	return initDNSServer(
		Context.filters,
		Context.stats,
		Context.queryLog,
		Context.dhcpServer,
		anonymizer,
		httpRegister,
		tlsConf,
		baseLogger,
	)
}

// initDNSServer initializes the [context.dnsServer].  To only use the internal
// proxy, none of the arguments are required, but tlsConf and l still must not
// be nil, in other cases all the arguments also must not be nil.  It also must
// not be called unless [config] and [Context] are initialized.
//
// TODO(e.burkov): Use [dnsforward.DNSCreateParams] as a parameter.
func initDNSServer(
	filters *filtering.DNSFilter,
	sts stats.Interface,
	qlog querylog.QueryLog,
	dhcpSrv dnsforward.DHCP,
	anonymizer *aghnet.IPMut,
	httpReg aghhttp.RegisterFunc,
	tlsConf *tlsConfigSettings,
	l *slog.Logger,
) (err error) {
	Context.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger:      l,
		DNSFilter:   filters,
		Stats:       sts,
		QueryLog:    qlog,
		PrivateNets: parseSubnetSet(config.DNS.PrivateNets),
		Anonymizer:  anonymizer,
		DHCPServer:  dhcpSrv,
		EtcHosts:    Context.etcHosts,
		LocalDomain: config.DHCP.LocalDomainName,
	})
	defer func() {
		if err != nil {
			closeDNSServer()
		}
	}()
	if err != nil {
		return fmt.Errorf("dnsforward.NewServer: %w", err)
	}

	Context.clients.clientChecker = Context.dnsServer

	dnsConf, err := newServerConfig(&config.DNS, config.Clients.Sources, tlsConf, httpReg)
	if err != nil {
		return fmt.Errorf("newServerConfig: %w", err)
	}

	// Try to prepare the server with disabled private RDNS resolution if it
	// failed to prepare as is.  See TODO on [dnsforward.PrivateRDNSError].
	err = Context.dnsServer.Prepare(dnsConf)
	if privRDNSErr := (&dnsforward.PrivateRDNSError{}); errors.As(err, &privRDNSErr) {
		log.Info("WARNING: %s; trying to disable private RDNS resolution", err)

		dnsConf.UsePrivateRDNS = false
		err = Context.dnsServer.Prepare(dnsConf)
	}

	if err != nil {
		return fmt.Errorf("dnsServer.Prepare: %w", err)
	}

	return nil
}

// parseSubnetSet parses a slice of subnets.  If the slice is empty, it returns
// a subnet set that matches all locally served networks, see
// [netutil.IsLocallyServed].
func parseSubnetSet(nets []netutil.Prefix) (s netutil.SubnetSet) {
	switch len(nets) {
	case 0:
		// Use an optimized function-based matcher.
		return netutil.SubnetSetFunc(netutil.IsLocallyServed)
	case 1:
		return nets[0].Prefix
	default:
		return netutil.SliceSubnetSet(netutil.UnembedPrefixes(nets))
	}
}

func isRunning() bool {
	return Context.dnsServer != nil && Context.dnsServer.IsRunning()
}

func ipsToTCPAddrs(ips []netip.Addr, port uint16) (tcpAddrs []*net.TCPAddr) {
	if ips == nil {
		return nil
	}

	tcpAddrs = make([]*net.TCPAddr, 0, len(ips))
	for _, ip := range ips {
		tcpAddrs = append(tcpAddrs, net.TCPAddrFromAddrPort(netip.AddrPortFrom(ip, port)))
	}

	return tcpAddrs
}

func ipsToUDPAddrs(ips []netip.Addr, port uint16) (udpAddrs []*net.UDPAddr) {
	if ips == nil {
		return nil
	}

	udpAddrs = make([]*net.UDPAddr, 0, len(ips))
	for _, ip := range ips {
		udpAddrs = append(udpAddrs, net.UDPAddrFromAddrPort(netip.AddrPortFrom(ip, port)))
	}

	return udpAddrs
}

// newServerConfig converts values from the configuration file into the internal
// DNS server configuration.  All arguments must not be nil.
func newServerConfig(
	dnsConf *dnsConfig,
	clientSrcConf *clientSourcesConfig,
	tlsConf *tlsConfigSettings,
	httpReg aghhttp.RegisterFunc,
) (newConf *dnsforward.ServerConfig, err error) {
	hosts := aghalg.CoalesceSlice(dnsConf.BindHosts, []netip.Addr{netutil.IPv4Localhost()})

	fwdConf := dnsConf.Config
	fwdConf.FilterHandler = applyAdditionalFiltering
	fwdConf.ClientsContainer = &Context.clients

	newConf = &dnsforward.ServerConfig{
		UDPListenAddrs:         ipsToUDPAddrs(hosts, dnsConf.Port),
		TCPListenAddrs:         ipsToTCPAddrs(hosts, dnsConf.Port),
		Config:                 fwdConf,
		TLSConfig:              newDNSTLSConfig(tlsConf, hosts),
		TLSAllowUnencryptedDoH: tlsConf.AllowUnencryptedDoH,
		UpstreamTimeout:        time.Duration(dnsConf.UpstreamTimeout),
		TLSv12Roots:            Context.tlsRoots,
		ConfigModified:         onConfigModified,
		HTTPRegister:           httpReg,
		LocalPTRResolvers:      dnsConf.PrivateRDNSResolvers,
		UseDNS64:               dnsConf.UseDNS64,
		DNS64Prefixes:          dnsConf.DNS64Prefixes,
		UsePrivateRDNS:         dnsConf.UsePrivateRDNS,
		ServeHTTP3:             dnsConf.ServeHTTP3,
		UseHTTP3Upstreams:      dnsConf.UseHTTP3Upstreams,
		ServePlainDNS:          dnsConf.ServePlainDNS,
	}

	var initialAddresses []netip.Addr
	// Context.stats may be nil here if initDNSServer is called from
	// [cmdlineUpdate].
	if sts := Context.stats; sts != nil {
		const initialClientsNum = 100
		initialAddresses = Context.stats.TopClientsIP(initialClientsNum)
	}

	// Do not set DialContext, PrivateSubnets, and UsePrivateRDNS, because they
	// are set by [dnsforward.Server.Prepare].
	newConf.AddrProcConf = &client.DefaultAddrProcConfig{
		Exchanger:        Context.dnsServer,
		AddressUpdater:   &Context.clients,
		InitialAddresses: initialAddresses,
		CatchPanics:      true,
		UseRDNS:          clientSrcConf.RDNS,
		UseWHOIS:         clientSrcConf.WHOIS,
	}

	newConf.DNSCryptConfig, err = newDNSCryptConfig(tlsConf, hosts)
	if err != nil {
		// Don't wrap the error, because it's already wrapped by
		// newDNSCryptConfig.
		return nil, err
	}

	return newConf, nil
}

// newDNSTLSConfig converts values from the configuration file into the internal
// TLS settings for the DNS server.  tlsConf must not be nil.
func newDNSTLSConfig(conf *tlsConfigSettings, addrs []netip.Addr) (dnsConf dnsforward.TLSConfig) {
	if !conf.Enabled {
		return dnsforward.TLSConfig{}
	}

	dnsConf = conf.TLSConfig
	dnsConf.ServerName = conf.ServerName

	if conf.PortHTTPS != 0 {
		dnsConf.HTTPSListenAddrs = ipsToTCPAddrs(addrs, conf.PortHTTPS)
	}

	if conf.PortDNSOverTLS != 0 {
		dnsConf.TLSListenAddrs = ipsToTCPAddrs(addrs, conf.PortDNSOverTLS)
	}

	if conf.PortDNSOverQUIC != 0 {
		dnsConf.QUICListenAddrs = ipsToUDPAddrs(addrs, conf.PortDNSOverQUIC)
	}

	return dnsConf
}

// newDNSCryptConfig converts values from the configuration file into the
// internal DNSCrypt settings for the DNS server.  conf must not be nil.
func newDNSCryptConfig(
	conf *tlsConfigSettings,
	addrs []netip.Addr,
) (dnsCryptConf dnsforward.DNSCryptConfig, err error) {
	if !conf.Enabled || conf.PortDNSCrypt == 0 {
		return dnsforward.DNSCryptConfig{}, nil
	}

	if conf.DNSCryptConfigFile == "" {
		return dnsforward.DNSCryptConfig{}, errors.Error("no dnscrypt_config_file")
	}

	f, err := os.Open(conf.DNSCryptConfigFile)
	if err != nil {
		return dnsforward.DNSCryptConfig{}, fmt.Errorf("opening dnscrypt config: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	rc := &dnscrypt.ResolverConfig{}
	err = yaml.NewDecoder(f).Decode(rc)
	if err != nil {
		return dnsforward.DNSCryptConfig{}, fmt.Errorf("decoding dnscrypt config: %w", err)
	}

	cert, err := rc.CreateCert()
	if err != nil {
		return dnsforward.DNSCryptConfig{}, fmt.Errorf("creating dnscrypt cert: %w", err)
	}

	return dnsforward.DNSCryptConfig{
		ResolverCert:   cert,
		ProviderName:   rc.ProviderName,
		UDPListenAddrs: ipsToUDPAddrs(addrs, conf.PortDNSCrypt),
		TCPListenAddrs: ipsToTCPAddrs(addrs, conf.PortDNSCrypt),
		Enabled:        true,
	}, nil
}

type dnsEncryption struct {
	https string
	tls   string
	quic  string
}

func getDNSEncryption() (de dnsEncryption) {
	tlsConf := tlsConfigSettings{}

	Context.tls.WriteDiskConfig(&tlsConf)

	if !tlsConf.Enabled || len(tlsConf.ServerName) == 0 {
		return dnsEncryption{}
	}

	hostname := tlsConf.ServerName
	if tlsConf.PortHTTPS != 0 {
		addr := hostname
		if p := tlsConf.PortHTTPS; p != defaultPortHTTPS {
			addr = netutil.JoinHostPort(addr, p)
		}

		de.https = (&url.URL{
			Scheme: urlutil.SchemeHTTPS,
			Host:   addr,
			Path:   "/dns-query",
		}).String()
	}

	if p := tlsConf.PortDNSOverTLS; p != 0 {
		de.tls = (&url.URL{
			Scheme: "tls",
			Host:   netutil.JoinHostPort(hostname, p),
		}).String()
	}

	if p := tlsConf.PortDNSOverQUIC; p != 0 {
		de.quic = (&url.URL{
			Scheme: "quic",
			Host:   netutil.JoinHostPort(hostname, p),
		}).String()
	}

	return de
}

// applyAdditionalFiltering adds additional client information and settings if
// the client has them.
func applyAdditionalFiltering(clientIP netip.Addr, clientID string, setts *filtering.Settings) {
	// pref is a prefix for logging messages around the scope.
	const pref = "applying filters"

	Context.filters.ApplyBlockedServices(setts)

	log.Debug("%s: looking for client with ip %s and clientid %q", pref, clientIP, clientID)

	if !clientIP.IsValid() {
		return
	}

	setts.ClientIP = clientIP

	c, ok := Context.clients.storage.Find(clientID)
	if !ok {
		c, ok = Context.clients.storage.Find(clientIP.String())
		if !ok {
			log.Debug("%s: no clients with ip %s and clientid %q", pref, clientIP, clientID)

			return
		}
	}

	log.Debug("%s: using settings for client %q (%s; %q)", pref, c.Name, clientIP, clientID)

	if c.UseOwnBlockedServices {
		// TODO(e.burkov):  Get rid of this crutch.
		setts.ServicesRules = nil
		svcs := c.BlockedServices.IDs
		if !c.BlockedServices.Schedule.Contains(time.Now()) {
			Context.filters.ApplyBlockedServicesList(setts, svcs)
			log.Debug("%s: services for client %q set: %s", pref, c.Name, svcs)
		}
	}

	setts.ClientName = c.Name
	setts.ClientTags = c.Tags
	if !c.UseOwnSettings {
		return
	}

	setts.FilteringEnabled = c.FilteringEnabled
	setts.SafeSearchEnabled = c.SafeSearchConf.Enabled
	setts.ClientSafeSearch = c.SafeSearch
	setts.SafeBrowsingEnabled = c.SafeBrowsingEnabled
	setts.ParentalEnabled = c.ParentalEnabled
}

func startDNSServer() error {
	config.RLock()
	defer config.RUnlock()

	if isRunning() {
		return fmt.Errorf("unable to start forwarding DNS server: Already running")
	}

	Context.filters.EnableFilters(false)

	// TODO(s.chzhen):  Pass context.
	ctx := context.TODO()
	err := Context.clients.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting clients container: %w", err)
	}

	err = Context.dnsServer.Start()
	if err != nil {
		return fmt.Errorf("starting dns server: %w", err)
	}

	Context.filters.Start()
	Context.stats.Start()

	err = Context.queryLog.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting query log: %w", err)
	}

	return nil
}

func reconfigureDNSServer() (err error) {
	tlsConf := &tlsConfigSettings{}
	Context.tls.WriteDiskConfig(tlsConf)

	newConf, err := newServerConfig(&config.DNS, config.Clients.Sources, tlsConf, httpRegister)
	if err != nil {
		return fmt.Errorf("generating forwarding dns server config: %w", err)
	}

	err = Context.dnsServer.Reconfigure(newConf)
	if err != nil {
		return fmt.Errorf("starting forwarding dns server: %w", err)
	}

	return nil
}

func stopDNSServer() (err error) {
	if !isRunning() {
		return nil
	}

	err = Context.dnsServer.Stop()
	if err != nil {
		return fmt.Errorf("stopping forwarding dns server: %w", err)
	}

	err = Context.clients.close(context.TODO())
	if err != nil {
		return fmt.Errorf("closing clients container: %w", err)
	}

	closeDNSServer()

	return nil
}

func closeDNSServer() {
	// DNS forward module must be closed BEFORE stats or queryLog because it depends on them
	if Context.dnsServer != nil {
		Context.dnsServer.Close()
		Context.dnsServer = nil
	}

	if Context.filters != nil {
		Context.filters.Close()
	}

	if Context.stats != nil {
		err := Context.stats.Close()
		if err != nil {
			log.Error("closing stats: %s", err)
		}
	}

	if Context.queryLog != nil {
		// TODO(s.chzhen):  Pass context.
		err := Context.queryLog.Shutdown(context.TODO())
		if err != nil {
			log.Error("closing query log: %s", err)
		}
	}

	log.Debug("all dns modules are closed")
}

// checkStatsAndQuerylogDirs checks and returns directory paths to store
// statistics and query log.
func checkStatsAndQuerylogDirs(
	ctx *homeContext,
	conf *configuration,
) (statsDir, querylogDir string, err error) {
	baseDir := ctx.getDataDir()

	statsDir = conf.Stats.DirPath
	if statsDir == "" {
		statsDir = baseDir
	} else {
		err = checkDir(statsDir)
		if err != nil {
			return "", "", fmt.Errorf("statistics: custom directory: %w", err)
		}
	}

	querylogDir = conf.QueryLog.DirPath
	if querylogDir == "" {
		querylogDir = baseDir
	} else {
		err = checkDir(querylogDir)
		if err != nil {
			return "", "", fmt.Errorf("querylog: custom directory: %w", err)
		}
	}

	return statsDir, querylogDir, nil
}

// checkDir checks if the path is a directory.  It's used to check for
// misconfiguration at startup.
func checkDir(path string) (err error) {
	var fi os.FileInfo
	if fi, err = os.Stat(path); err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory", path)
	}

	return nil
}
