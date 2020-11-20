package home

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/AdGuardHome/internal/util"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
)

// Called by other modules when configuration is changed
func onConfigModified() {
	_ = config.write()
}

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer() error {
	var err error
	baseDir := Context.getDataDir()

	statsConf := stats.Config{
		Filename:          filepath.Join(baseDir, "stats.db"),
		LimitDays:         config.DNS.StatsInterval,
		AnonymizeClientIP: config.DNS.AnonymizeClientIP,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
	}
	Context.stats, err = stats.New(statsConf)
	if err != nil {
		return fmt.Errorf("couldn't initialize statistics module")
	}
	conf := querylog.Config{
		Enabled:           config.DNS.QueryLogEnabled,
		FileEnabled:       config.DNS.QueryLogFileEnabled,
		BaseDir:           baseDir,
		Interval:          config.DNS.QueryLogInterval,
		MemSize:           config.DNS.QueryLogMemSize,
		AnonymizeClientIP: config.DNS.AnonymizeClientIP,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
	}
	Context.queryLog = querylog.New(conf)

	filterConf := config.DNS.DnsfilterConf
	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	filterConf.ResolverAddress = fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)
	filterConf.AutoHosts = &Context.autoHosts
	filterConf.ConfigModified = onConfigModified
	filterConf.HTTPRegister = httpRegister
	Context.dnsFilter = dnsfilter.New(&filterConf, nil)

	p := dnsforward.DNSCreateParams{
		DNSFilter: Context.dnsFilter,
		Stats:     Context.stats,
		QueryLog:  Context.queryLog,
	}
	if Context.dhcpServer != nil {
		p.DHCPServer = Context.dhcpServer
	}
	Context.dnsServer = dnsforward.NewServer(p)
	Context.clients.dnsServer = Context.dnsServer
	dnsConfig := generateServerConfig()
	err = Context.dnsServer.Prepare(&dnsConfig)
	if err != nil {
		closeDNSServer()
		return fmt.Errorf("dnsServer.Prepare: %w", err)
	}

	Context.rdns = InitRDNS(Context.dnsServer, &Context.clients)
	Context.whois = initWhois(&Context.clients)

	Context.filters.Init()
	return nil
}

func isRunning() bool {
	return Context.dnsServer != nil && Context.dnsServer.IsRunning()
}

func onDNSRequest(d *proxy.DNSContext) {
	ip := dnsforward.GetIPString(d.Addr)
	if ip == "" {
		// This would be quite weird if we get here
		return
	}

	ipAddr := net.ParseIP(ip)
	if !ipAddr.IsLoopback() {
		Context.rdns.Begin(ip)
	}
	if !Context.ipDetector.detectSpecialNetwork(ipAddr) {
		Context.whois.Begin(ip)
	}
}

func generateServerConfig() dnsforward.ServerConfig {
	newconfig := dnsforward.ServerConfig{
		UDPListenAddr:   &net.UDPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		TCPListenAddr:   &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		FilteringConfig: config.DNS.FilteringConfig,
		ConfigModified:  onConfigModified,
		HTTPRegister:    httpRegister,
		OnDNSRequest:    onDNSRequest,
	}

	tlsConf := tlsConfigSettings{}
	Context.tls.WriteDiskConfig(&tlsConf)
	if tlsConf.Enabled {
		newconfig.TLSConfig = tlsConf.TLSConfig

		if tlsConf.PortDNSOverTLS != 0 {
			newconfig.TLSListenAddr = &net.TCPAddr{
				IP:   net.ParseIP(config.DNS.BindHost),
				Port: tlsConf.PortDNSOverTLS,
			}
		}

		if tlsConf.PortDNSOverQUIC != 0 {
			newconfig.QUICListenAddr = &net.UDPAddr{
				IP:   net.ParseIP(config.DNS.BindHost),
				Port: int(tlsConf.PortDNSOverQUIC),
			}
		}
	}
	newconfig.TLSv12Roots = Context.tlsRoots
	newconfig.TLSCiphers = Context.tlsCiphers
	newconfig.TLSAllowUnencryptedDOH = tlsConf.AllowUnencryptedDOH

	newconfig.FilterHandler = applyAdditionalFiltering
	newconfig.GetCustomUpstreamByClient = Context.clients.FindUpstreams
	return newconfig
}

type DNSEncryption struct {
	https string
	tls   string
	quic  string
}

func getDNSEncryption() DNSEncryption {
	dnsEncryption := DNSEncryption{}

	tlsConf := tlsConfigSettings{}

	Context.tls.WriteDiskConfig(&tlsConf)

	if tlsConf.Enabled && len(tlsConf.ServerName) != 0 {

		if tlsConf.PortHTTPS != 0 {
			addr := tlsConf.ServerName
			if tlsConf.PortHTTPS != 443 {
				addr = fmt.Sprintf("%s:%d", addr, tlsConf.PortHTTPS)
			}
			addr = fmt.Sprintf("https://%s/dns-query", addr)
			dnsEncryption.https = addr
		}

		if tlsConf.PortDNSOverTLS != 0 {
			addr := fmt.Sprintf("tls://%s:%d", tlsConf.ServerName, tlsConf.PortDNSOverTLS)
			dnsEncryption.tls = addr
		}

		if tlsConf.PortDNSOverQUIC != 0 {
			addr := fmt.Sprintf("quic://%s:%d", tlsConf.ServerName, tlsConf.PortDNSOverQUIC)
			dnsEncryption.quic = addr
		}
	}

	return dnsEncryption
}

// Get the list of DNS addresses the server is listening on
func getDNSAddresses() []string {
	dnsAddresses := []string{}

	if config.DNS.BindHost == "0.0.0.0" {
		ifaces, e := util.GetValidNetInterfacesForWeb()
		if e != nil {
			log.Error("Couldn't get network interfaces: %v", e)
			return []string{}
		}

		for _, iface := range ifaces {
			for _, addr := range iface.Addresses {
				addDNSAddress(&dnsAddresses, addr)
			}
		}
	} else {
		addDNSAddress(&dnsAddresses, config.DNS.BindHost)
	}

	dnsEncryption := getDNSEncryption()
	if dnsEncryption.https != "" {
		dnsAddresses = append(dnsAddresses, dnsEncryption.https)
	}
	if dnsEncryption.tls != "" {
		dnsAddresses = append(dnsAddresses, dnsEncryption.tls)
	}
	if dnsEncryption.quic != "" {
		dnsAddresses = append(dnsAddresses, dnsEncryption.quic)
	}

	return dnsAddresses
}

// If a client has his own settings, apply them
func applyAdditionalFiltering(clientAddr string, setts *dnsfilter.RequestFilteringSettings) {
	Context.dnsFilter.ApplyBlockedServices(setts, nil, true)

	if len(clientAddr) == 0 {
		return
	}
	setts.ClientIP = clientAddr

	c, ok := Context.clients.Find(clientAddr)
	if !ok {
		return
	}

	log.Debug("Using settings for client %s with IP %s", c.Name, clientAddr)

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
	if isRunning() {
		return fmt.Errorf("unable to start forwarding DNS server: Already running")
	}

	enableFilters(false)

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
	topClients := Context.stats.GetTopClientsIP(topClientsNumber)
	for _, ip := range topClients {
		ipAddr := net.ParseIP(ip)
		if !ipAddr.IsLoopback() {
			Context.rdns.Begin(ip)
		}
		if !Context.ipDetector.detectSpecialNetwork(ipAddr) {
			Context.whois.Begin(ip)
		}
	}

	return nil
}

func reconfigureDNSServer() error {
	newconfig := generateServerConfig()
	err := Context.dnsServer.Reconfigure(&newconfig)
	if err != nil {
		return fmt.Errorf("couldn't start forwarding DNS server: %w", err)
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
