package home

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/ameshkov/dnscrypt/v2"
	yaml "gopkg.in/yaml.v3"
)

// Default listening ports.
const (
	defaultPortDNS   = 53
	defaultPortHTTP  = 80
	defaultPortHTTPS = 443
	defaultPortQUIC  = 853
	defaultPortTLS   = 853
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
// [config] and [Context] are initialized.
func initDNS() (err error) {
	baseDir := Context.getDataDir()

	anonymizer := config.anonymizer()

	statsConf := stats.Config{
		Filename:          filepath.Join(baseDir, "stats.db"),
		Limit:             config.Stats.Interval.Duration,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
		Enabled:           config.Stats.Enabled,
		ShouldCountClient: Context.clients.shouldCountClient,
	}

	set, err := aghnet.NewDomainNameSet(config.Stats.Ignored)
	if err != nil {
		return fmt.Errorf("statistics: ignored list: %w", err)
	}

	statsConf.Ignored = set
	Context.stats, err = stats.New(statsConf)
	if err != nil {
		return fmt.Errorf("init stats: %w", err)
	}

	conf := querylog.Config{
		Anonymizer:        anonymizer,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
		FindClient:        Context.clients.findMultiple,
		BaseDir:           baseDir,
		AnonymizeClientIP: config.DNS.AnonymizeClientIP,
		RotationIvl:       config.QueryLog.Interval.Duration,
		MemSize:           config.QueryLog.MemSize,
		Enabled:           config.QueryLog.Enabled,
		FileEnabled:       config.QueryLog.FileEnabled,
	}

	set, err = aghnet.NewDomainNameSet(config.QueryLog.Ignored)
	if err != nil {
		return fmt.Errorf("querylog: ignored list: %w", err)
	}

	conf.Ignored = set
	Context.queryLog, err = querylog.New(conf)
	if err != nil {
		return fmt.Errorf("init querylog: %w", err)
	}

	Context.filters, err = filtering.New(config.DNS.DnsfilterConf, nil)
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
	)
}

// initDNSServer initializes the [context.dnsServer].  To only use the internal
// proxy, none of the arguments are required, but tlsConf still must not be nil,
// in other cases all the arguments also must not be nil.  It also must not be
// called unless [config] and [Context] are initialized.
func initDNSServer(
	filters *filtering.DNSFilter,
	sts stats.Interface,
	qlog querylog.QueryLog,
	dhcpSrv dhcpd.Interface,
	anonymizer *aghnet.IPMut,
	httpReg aghhttp.RegisterFunc,
	tlsConf *tlsConfigSettings,
) (err error) {
	privateNets, err := parseSubnetSet(config.DNS.PrivateNets)
	if err != nil {
		return fmt.Errorf("preparing set of private subnets: %w", err)
	}

	p := dnsforward.DNSCreateParams{
		DNSFilter:   filters,
		Stats:       sts,
		QueryLog:    qlog,
		PrivateNets: privateNets,
		Anonymizer:  anonymizer,
		LocalDomain: config.DHCP.LocalDomainName,
		DHCPServer:  dhcpSrv,
	}

	Context.dnsServer, err = dnsforward.NewServer(p)
	if err != nil {
		closeDNSServer()

		return fmt.Errorf("dnsforward.NewServer: %w", err)
	}

	Context.clients.dnsServer = Context.dnsServer

	dnsConf, err := generateServerConfig(tlsConf, httpReg)
	if err != nil {
		closeDNSServer()

		return fmt.Errorf("generateServerConfig: %w", err)
	}

	err = Context.dnsServer.Prepare(&dnsConf)
	if err != nil {
		closeDNSServer()

		return fmt.Errorf("dnsServer.Prepare: %w", err)
	}

	if config.Clients.Sources.RDNS {
		Context.rdns = NewRDNS(Context.dnsServer, &Context.clients, config.DNS.UsePrivateRDNS)
	}

	initWHOIS()

	return nil
}

// initWHOIS initializes the WHOIS.
//
// TODO(s.chzhen):  Consider making configurable.
func initWHOIS() {
	const (
		// defaultQueueSize is the size of queue of IPs for WHOIS processing.
		defaultQueueSize = 255

		// defaultTimeout is the timeout for WHOIS requests.
		defaultTimeout = 5 * time.Second

		// defaultCacheSize is the maximum size of the cache.  If it's zero,
		// cache size is unlimited.
		defaultCacheSize = 10_000

		// defaultMaxConnReadSize is an upper limit in bytes for reading from
		// net.Conn.
		defaultMaxConnReadSize = 64 * 1024

		// defaultMaxRedirects is the maximum redirects count.
		defaultMaxRedirects = 5

		// defaultMaxInfoLen is the maximum length of whois.Info fields.
		defaultMaxInfoLen = 250

		// defaultIPTTL is the Time to Live duration for cached IP addresses.
		defaultIPTTL = 1 * time.Hour
	)

	Context.whoisCh = make(chan netip.Addr, defaultQueueSize)

	var w whois.Interface

	if config.Clients.Sources.WHOIS {
		w = whois.New(&whois.Config{
			DialContext:     customDialContext,
			ServerAddr:      whois.DefaultServer,
			Port:            whois.DefaultPort,
			Timeout:         defaultTimeout,
			CacheSize:       defaultCacheSize,
			MaxConnReadSize: defaultMaxConnReadSize,
			MaxRedirects:    defaultMaxRedirects,
			MaxInfoLen:      defaultMaxInfoLen,
			CacheTTL:        defaultIPTTL,
		})
	} else {
		w = whois.Empty{}
	}

	go func() {
		defer log.OnPanic("whois")

		for ip := range Context.whoisCh {
			info, changed := w.Process(context.Background(), ip)
			if info != nil && changed {
				Context.clients.setWHOISInfo(ip, info)
			}
		}
	}()
}

// parseSubnetSet parses a slice of subnets.  If the slice is empty, it returns
// a subnet set that matches all locally served networks, see
// [netutil.IsLocallyServed].
func parseSubnetSet(nets []string) (s netutil.SubnetSet, err error) {
	switch len(nets) {
	case 0:
		// Use an optimized function-based matcher.
		return netutil.SubnetSetFunc(netutil.IsLocallyServed), nil
	case 1:
		s, err = netutil.ParseSubnet(nets[0])
		if err != nil {
			return nil, err
		}

		return s, nil
	default:
		var nets []*net.IPNet
		nets, err = netutil.ParseSubnets(config.DNS.PrivateNets...)
		if err != nil {
			return nil, err
		}

		return netutil.SliceSubnetSet(nets), nil
	}
}

func isRunning() bool {
	return Context.dnsServer != nil && Context.dnsServer.IsRunning()
}

func onDNSRequest(pctx *proxy.DNSContext) {
	ip := netutil.NetAddrToAddrPort(pctx.Addr).Addr()
	if ip == (netip.Addr{}) {
		// This would be quite weird if we get here.
		return
	}

	srcs := config.Clients.Sources
	if srcs.RDNS && !ip.IsLoopback() {
		Context.rdns.Begin(ip)
	}

	Context.whoisCh <- ip
}

func ipsToTCPAddrs(ips []netip.Addr, port int) (tcpAddrs []*net.TCPAddr) {
	if ips == nil {
		return nil
	}

	tcpAddrs = make([]*net.TCPAddr, 0, len(ips))
	for _, ip := range ips {
		tcpAddrs = append(tcpAddrs, net.TCPAddrFromAddrPort(netip.AddrPortFrom(ip, uint16(port))))
	}

	return tcpAddrs
}

func ipsToUDPAddrs(ips []netip.Addr, port int) (udpAddrs []*net.UDPAddr) {
	if ips == nil {
		return nil
	}

	udpAddrs = make([]*net.UDPAddr, 0, len(ips))
	for _, ip := range ips {
		udpAddrs = append(udpAddrs, net.UDPAddrFromAddrPort(netip.AddrPortFrom(ip, uint16(port))))
	}

	return udpAddrs
}

func generateServerConfig(
	tlsConf *tlsConfigSettings,
	httpReg aghhttp.RegisterFunc,
) (newConf dnsforward.ServerConfig, err error) {
	dnsConf := config.DNS
	hosts := aghalg.CoalesceSlice(dnsConf.BindHosts, []netip.Addr{netutil.IPv4Localhost()})
	newConf = dnsforward.ServerConfig{
		UDPListenAddrs:  ipsToUDPAddrs(hosts, dnsConf.Port),
		TCPListenAddrs:  ipsToTCPAddrs(hosts, dnsConf.Port),
		FilteringConfig: dnsConf.FilteringConfig,
		ConfigModified:  onConfigModified,
		HTTPRegister:    httpReg,
		OnDNSRequest:    onDNSRequest,
		UseDNS64:        config.DNS.UseDNS64,
		DNS64Prefixes:   config.DNS.DNS64Prefixes,
	}

	if tlsConf.Enabled {
		newConf.TLSConfig = tlsConf.TLSConfig
		newConf.TLSConfig.ServerName = tlsConf.ServerName

		if tlsConf.PortHTTPS != 0 {
			newConf.HTTPSListenAddrs = ipsToTCPAddrs(hosts, tlsConf.PortHTTPS)
		}

		if tlsConf.PortDNSOverTLS != 0 {
			newConf.TLSListenAddrs = ipsToTCPAddrs(hosts, tlsConf.PortDNSOverTLS)
		}

		if tlsConf.PortDNSOverQUIC != 0 {
			newConf.QUICListenAddrs = ipsToUDPAddrs(hosts, tlsConf.PortDNSOverQUIC)
		}

		if tlsConf.PortDNSCrypt != 0 {
			newConf.DNSCryptConfig, err = newDNSCrypt(hosts, *tlsConf)
			if err != nil {
				// Don't wrap the error, because it's already
				// wrapped by newDNSCrypt.
				return dnsforward.ServerConfig{}, err
			}
		}
	}

	newConf.TLSv12Roots = Context.tlsRoots
	newConf.TLSAllowUnencryptedDoH = tlsConf.AllowUnencryptedDoH

	newConf.FilterHandler = applyAdditionalFiltering
	newConf.GetCustomUpstreamByClient = Context.clients.findUpstreams

	newConf.LocalPTRResolvers = dnsConf.LocalPTRResolvers
	newConf.UpstreamTimeout = dnsConf.UpstreamTimeout.Duration

	newConf.ResolveClients = config.Clients.Sources.RDNS
	newConf.UsePrivateRDNS = dnsConf.UsePrivateRDNS
	newConf.ServeHTTP3 = dnsConf.ServeHTTP3
	newConf.UseHTTP3Upstreams = dnsConf.UseHTTP3Upstreams

	return newConf, nil
}

func newDNSCrypt(hosts []netip.Addr, tlsConf tlsConfigSettings) (dnscc dnsforward.DNSCryptConfig, err error) {
	if tlsConf.DNSCryptConfigFile == "" {
		return dnscc, errors.Error("no dnscrypt_config_file")
	}

	f, err := os.Open(tlsConf.DNSCryptConfigFile)
	if err != nil {
		return dnscc, fmt.Errorf("opening dnscrypt config: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	rc := &dnscrypt.ResolverConfig{}
	err = yaml.NewDecoder(f).Decode(rc)
	if err != nil {
		return dnscc, fmt.Errorf("decoding dnscrypt config: %w", err)
	}

	cert, err := rc.CreateCert()
	if err != nil {
		return dnscc, fmt.Errorf("creating dnscrypt cert: %w", err)
	}

	return dnsforward.DNSCryptConfig{
		ResolverCert:   cert,
		ProviderName:   rc.ProviderName,
		UDPListenAddrs: ipsToUDPAddrs(hosts, tlsConf.PortDNSCrypt),
		TCPListenAddrs: ipsToTCPAddrs(hosts, tlsConf.PortDNSCrypt),
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

	if tlsConf.Enabled && len(tlsConf.ServerName) != 0 {
		hostname := tlsConf.ServerName
		if tlsConf.PortHTTPS != 0 {
			addr := hostname
			if tlsConf.PortHTTPS != defaultPortHTTPS {
				addr = netutil.JoinHostPort(addr, tlsConf.PortHTTPS)
			}

			de.https = (&url.URL{
				Scheme: "https",
				Host:   addr,
				Path:   "/dns-query",
			}).String()
		}

		if tlsConf.PortDNSOverTLS != 0 {
			de.tls = (&url.URL{
				Scheme: "tls",
				Host:   netutil.JoinHostPort(hostname, tlsConf.PortDNSOverTLS),
			}).String()
		}

		if tlsConf.PortDNSOverQUIC != 0 {
			de.quic = (&url.URL{
				Scheme: "quic",
				Host:   netutil.JoinHostPort(hostname, tlsConf.PortDNSOverQUIC),
			}).String()
		}
	}

	return de
}

// applyAdditionalFiltering adds additional client information and settings if
// the client has them.
func applyAdditionalFiltering(clientIP net.IP, clientID string, setts *filtering.Settings) {
	// pref is a prefix for logging messages around the scope.
	const pref = "applying filters"

	Context.filters.ApplyBlockedServices(setts)

	log.Debug("%s: looking for client with ip %s and clientid %q", pref, clientIP, clientID)

	if clientIP == nil {
		return
	}

	setts.ClientIP = clientIP

	c, ok := Context.clients.Find(clientID)
	if !ok {
		c, ok = Context.clients.Find(clientIP.String())
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
	setts.SafeSearchEnabled = c.safeSearchConf.Enabled
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

	Context.clients.Start()

	err := Context.dnsServer.Start()
	if err != nil {
		return fmt.Errorf("couldn't start forwarding DNS server: %w", err)
	}

	Context.filters.Start()
	Context.stats.Start()
	Context.queryLog.Start()

	const topClientsNumber = 100 // the number of clients to get
	for _, ip := range Context.stats.TopClientsIP(topClientsNumber) {
		srcs := config.Clients.Sources
		if srcs.RDNS && !ip.IsLoopback() {
			Context.rdns.Begin(ip)
		}

		Context.whoisCh <- ip
	}

	return nil
}

func reconfigureDNSServer() (err error) {
	var newConf dnsforward.ServerConfig

	tlsConf := &tlsConfigSettings{}
	Context.tls.WriteDiskConfig(tlsConf)

	newConf, err = generateServerConfig(tlsConf, httpRegister)
	if err != nil {
		return fmt.Errorf("generating forwarding dns server config: %w", err)
	}

	err = Context.dnsServer.Reconfigure(&newConf)
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

	err = Context.clients.close()
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

	Context.filters.Close()

	if Context.stats != nil {
		err := Context.stats.Close()
		if err != nil {
			log.Debug("closing stats: %s", err)
		}

		// TODO(e.burkov):  Find out if it's safe.
		Context.stats = nil
	}

	if Context.queryLog != nil {
		Context.queryLog.Close()
		Context.queryLog = nil
	}

	log.Debug("all dns modules are closed")
}

// safeSearchResolver is a [filtering.Resolver] implementation used for safe
// search.
type safeSearchResolver struct{}

// type check
var _ filtering.Resolver = safeSearchResolver{}

// LookupIP implements [filtering.Resolver] interface for safeSearchResolver.
// It returns the slice of net.IP with IPv4 and IPv6 instances.
//
// TODO(a.garipov): Support network.
func (r safeSearchResolver) LookupIP(_ context.Context, _, host string) (ips []net.IP, err error) {
	addrs, err := Context.dnsServer.Resolve(host)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("couldn't lookup host: %s", host)
	}

	for _, a := range addrs {
		ips = append(ips, a.IP)
	}

	return ips, nil
}
