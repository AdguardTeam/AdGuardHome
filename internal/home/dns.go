package home

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog/jsonfile"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog/logs"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/ameshkov/dnscrypt/v2"
	yaml "gopkg.in/yaml.v3"
)

// Default ports.
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

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer() (err error) {
	baseDir := Context.getDataDir()

	var anonFunc aghnet.IPMutFunc
	if config.DNS.AnonymizeClientIP {
		anonFunc = logs.AnonymizeIP
	}
	anonymizer := aghnet.NewIPMut(anonFunc)

	statsConf := stats.Config{
		Filename:       filepath.Join(baseDir, "stats.db"),
		LimitDays:      config.DNS.StatsInterval,
		ConfigModified: onConfigModified,
		HTTPRegister:   httpRegister,
	}
	Context.stats, err = stats.New(statsConf)
	if err != nil {
		return fmt.Errorf("init stats: %w", err)
	}

	conf := logs.Config{
		Anonymizer:        anonymizer,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
		FindClient:        Context.clients.findMultiple,
		BaseDir:           baseDir,
		RotationIvl:       config.DNS.QueryLogInterval.Duration,
		MemSize:           config.DNS.QueryLogMemSize,
		Enabled:           config.DNS.QueryLogEnabled,
		FileEnabled:       config.DNS.QueryLogFileEnabled,
		AnonymizeClientIP: config.DNS.AnonymizeClientIP,
	}
	Context.queryLog = jsonfile.New(conf)
	logs.RegisterHTTP(Context.queryLog, httpRegister)

	Context.filters, err = filtering.New(config.DNS.DnsfilterConf, nil)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	var privateNets netutil.SubnetSet
	switch len(config.DNS.PrivateNets) {
	case 0:
		// Use an optimized locally-served matcher.
		privateNets = netutil.SubnetSetFunc(netutil.IsLocallyServed)
	case 1:
		privateNets, err = netutil.ParseSubnet(config.DNS.PrivateNets[0])
		if err != nil {
			return fmt.Errorf("preparing the set of private subnets: %w", err)
		}
	default:
		var nets []*net.IPNet
		nets, err = netutil.ParseSubnets(config.DNS.PrivateNets...)
		if err != nil {
			return fmt.Errorf("preparing the set of private subnets: %w", err)
		}

		privateNets = netutil.SliceSubnetSet(nets)
	}

	p := dnsforward.DNSCreateParams{
		DNSFilter:   Context.filters,
		Stats:       Context.stats,
		QueryLog:    Context.queryLog,
		PrivateNets: privateNets,
		Anonymizer:  anonymizer,
		LocalDomain: config.DHCP.LocalDomainName,
		DHCPServer:  Context.dhcpServer,
	}

	Context.dnsServer, err = dnsforward.NewServer(p)
	if err != nil {
		closeDNSServer()

		return fmt.Errorf("dnsforward.NewServer: %w", err)
	}

	Context.clients.dnsServer = Context.dnsServer
	var dnsConfig dnsforward.ServerConfig
	dnsConfig, err = generateServerConfig()
	if err != nil {
		closeDNSServer()

		return fmt.Errorf("generateServerConfig: %w", err)
	}

	err = Context.dnsServer.Prepare(&dnsConfig)
	if err != nil {
		closeDNSServer()

		return fmt.Errorf("dnsServer.Prepare: %w", err)
	}

	if config.Clients.Sources.RDNS {
		Context.rdns = NewRDNS(Context.dnsServer, &Context.clients, config.DNS.UsePrivateRDNS)
	}

	if config.Clients.Sources.WHOIS {
		Context.whois = initWHOIS(&Context.clients)
	}

	return nil
}

func isRunning() bool {
	return Context.dnsServer != nil && Context.dnsServer.IsRunning()
}

func onDNSRequest(pctx *proxy.DNSContext) {
	ip, _ := netutil.IPAndPortFromAddr(pctx.Addr)
	if ip == nil {
		// This would be quite weird if we get here.
		return
	}

	srcs := config.Clients.Sources
	if srcs.RDNS && !ip.IsLoopback() {
		Context.rdns.Begin(ip)
	}
	if srcs.WHOIS && !netutil.IsSpecialPurpose(ip) {
		Context.whois.Begin(ip)
	}
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

func generateServerConfig() (newConf dnsforward.ServerConfig, err error) {
	dnsConf := config.DNS
	hosts := dnsConf.BindHosts
	if len(hosts) == 0 {
		hosts = []netip.Addr{aghnet.IPv4Localhost()}
	}

	newConf = dnsforward.ServerConfig{
		UDPListenAddrs:  ipsToUDPAddrs(hosts, dnsConf.Port),
		TCPListenAddrs:  ipsToTCPAddrs(hosts, dnsConf.Port),
		FilteringConfig: dnsConf.FilteringConfig,
		ConfigModified:  onConfigModified,
		HTTPRegister:    httpRegister,
		OnDNSRequest:    onDNSRequest,
	}

	tlsConf := tlsConfigSettings{}
	Context.tls.WriteDiskConfig(&tlsConf)
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
			newConf.DNSCryptConfig, err = newDNSCrypt(hosts, tlsConf)
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

	Context.filters.ApplyBlockedServices(setts, nil)

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
		svcs := c.BlockedServices
		if svcs == nil {
			svcs = []string{}
		}
		Context.filters.ApplyBlockedServices(setts, svcs)
		log.Debug("%s: services for client %q set: %s", pref, c.Name, svcs)
	}

	setts.ClientName = c.Name
	setts.ClientTags = c.Tags
	if !c.UseOwnSettings {
		return
	}

	setts.FilteringEnabled = c.FilteringEnabled
	setts.SafeSearchEnabled = c.SafeSearchEnabled
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
		if ip == nil {
			continue
		}

		srcs := config.Clients.Sources
		if srcs.RDNS && !ip.IsLoopback() {
			Context.rdns.Begin(ip)
		}
		if srcs.WHOIS && !netutil.IsSpecialPurpose(ip) {
			Context.whois.Begin(ip)
		}
	}

	return nil
}

func reconfigureDNSServer() (err error) {
	var newConf dnsforward.ServerConfig
	newConf, err = generateServerConfig()
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
