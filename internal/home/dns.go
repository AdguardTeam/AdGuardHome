package home

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/ameshkov/dnscrypt/v2"
	yaml "gopkg.in/yaml.v2"
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
	_ = config.write()
}

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer() (err error) {
	baseDir := Context.getDataDir()

	var anonFunc aghnet.IPMutFunc
	if config.DNS.AnonymizeClientIP {
		anonFunc = querylog.AnonymizeIP
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

	conf := querylog.Config{
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
	Context.queryLog = querylog.New(conf)

	filterConf := config.DNS.DnsfilterConf
	filterConf.EtcHosts = Context.etcHosts
	filterConf.ConfigModified = onConfigModified
	filterConf.HTTPRegister = httpRegister
	Context.dnsFilter = filtering.New(&filterConf, nil)

	var privateNets netutil.SubnetSet
	switch len(config.DNS.PrivateNets) {
	case 0:
		// Use an optimized locally-served matcher.
		privateNets = netutil.SubnetSetFunc(netutil.IsLocallyServed)
	case 1:
		var n *net.IPNet
		n, err = netutil.ParseSubnet(config.DNS.PrivateNets[0])
		if err != nil {
			return fmt.Errorf("preparing the set of private subnets: %w", err)
		}

		privateNets = n
	default:
		var nets []*net.IPNet
		nets, err = netutil.ParseSubnets(config.DNS.PrivateNets...)
		if err != nil {
			return fmt.Errorf("preparing the set of private subnets: %w", err)
		}

		privateNets = netutil.SliceSubnetSet(nets)
	}

	p := dnsforward.DNSCreateParams{
		DNSFilter:   Context.dnsFilter,
		Stats:       Context.stats,
		QueryLog:    Context.queryLog,
		PrivateNets: privateNets,
		Anonymizer:  anonymizer,
		LocalDomain: config.DHCP.LocalDomainName,
	}
	if Context.dhcpServer != nil {
		p.DHCPServer = Context.dhcpServer
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

	Context.filters.Init()
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

func ipsToTCPAddrs(ips []net.IP, port int) (tcpAddrs []*net.TCPAddr) {
	if ips == nil {
		return nil
	}

	tcpAddrs = make([]*net.TCPAddr, len(ips))
	for i, ip := range ips {
		tcpAddrs[i] = &net.TCPAddr{
			IP:   ip,
			Port: port,
		}
	}

	return tcpAddrs
}

func ipsToUDPAddrs(ips []net.IP, port int) (udpAddrs []*net.UDPAddr) {
	if ips == nil {
		return nil
	}

	udpAddrs = make([]*net.UDPAddr, len(ips))
	for i, ip := range ips {
		udpAddrs[i] = &net.UDPAddr{
			IP:   ip,
			Port: port,
		}
	}

	return udpAddrs
}

func generateServerConfig() (newConf dnsforward.ServerConfig, err error) {
	dnsConf := config.DNS
	hosts := dnsConf.BindHosts
	if len(hosts) == 0 {
		hosts = []net.IP{{127, 0, 0, 1}}
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

	newConf.ResolveClients = config.Clients.Sources.RDNS
	newConf.UsePrivateRDNS = dnsConf.UsePrivateRDNS
	newConf.LocalPTRResolvers = dnsConf.LocalPTRResolvers
	newConf.UpstreamTimeout = dnsConf.UpstreamTimeout.Duration

	return newConf, nil
}

func newDNSCrypt(hosts []net.IP, tlsConf tlsConfigSettings) (dnscc dnsforward.DNSCryptConfig, err error) {
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
		UDPListenAddrs: ipsToUDPAddrs(hosts, tlsConf.PortDNSCrypt),
		TCPListenAddrs: ipsToTCPAddrs(hosts, tlsConf.PortDNSCrypt),
		ResolverCert:   cert,
		ProviderName:   rc.ProviderName,
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
	Context.dnsFilter.ApplyBlockedServices(setts, nil, true)

	log.Debug("looking up settings for client with ip %s and clientid %q", clientIP, clientID)

	if clientIP == nil {
		return
	}

	setts.ClientIP = clientIP

	c, ok := Context.clients.Find(clientID)
	if !ok {
		c, ok = Context.clients.Find(clientIP.String())
		if !ok {
			log.Debug("client with ip %s and clientid %q not found", clientIP, clientID)

			return
		}
	}

	log.Debug("using settings for client %q with ip %s and clientid %q", c.Name, clientIP, clientID)

	if c.UseOwnBlockedServices {
		Context.dnsFilter.ApplyBlockedServices(setts, c.BlockedServices, false)
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

	enableFiltersLocked(false)

	Context.clients.Start()

	err := Context.dnsServer.Start()
	if err != nil {
		return fmt.Errorf("couldn't start forwarding DNS server: %w", err)
	}

	Context.dnsFilter.Start()
	Context.filters.Start()
	Context.stats.Start()
	Context.queryLog.Start()

	const topClientsNumber = 100 // the number of clients to get
	for _, ip := range Context.stats.GetTopClientsIP(topClientsNumber) {
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

func stopDNSServer() error {
	if !isRunning() {
		return nil
	}

	err := Context.dnsServer.Stop()
	if err != nil {
		return fmt.Errorf("couldn't stop forwarding DNS server: %w", err)
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

	if Context.dnsFilter != nil {
		Context.dnsFilter.Close()
		Context.dnsFilter = nil
	}

	if Context.stats != nil {
		Context.stats.Close()
		Context.stats = nil
	}

	if Context.queryLog != nil {
		Context.queryLog.Close()
		Context.queryLog = nil
	}

	Context.filters.Close()

	log.Debug("Closed all DNS modules")
}
