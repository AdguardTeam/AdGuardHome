package home

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

// Called by other modules when configuration is changed
func onConfigModified() {
	_ = config.write()
}

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func (c *homeContext) initDNSServer() error {
	var err error
	baseDir := c.getDataDir()

	statsConf := stats.Config{
		Filename:          filepath.Join(baseDir, "stats.db"),
		LimitDays:         config.DNS.StatsInterval,
		AnonymizeClientIP: config.DNS.AnonymizeClientIP,
		ConfigModified:    onConfigModified,
		HTTPRegister:      httpRegister,
	}
	c.stats, err = stats.New(statsConf)
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
	c.queryLog = querylog.New(conf)

	filterConf := config.DNS.DnsfilterConf
	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	filterConf.ResolverAddress = fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)
	filterConf.AutoHosts = &c.autoHosts
	filterConf.ConfigModified = onConfigModified
	filterConf.HTTPRegister = httpRegister
	c.dnsFilter = dnsfilter.New(&filterConf, nil)

	p := dnsforward.DNSCreateParams{
		DNSFilter: c.dnsFilter,
		Stats:     c.stats,
		QueryLog:  c.queryLog,
	}
	if c.dhcpServer != nil {
		p.DHCPServer = c.dhcpServer
	}
	c.dnsServer = dnsforward.NewServer(p)
	dnsConfig := c.generateServerConfig()
	err = c.dnsServer.Prepare(&dnsConfig)
	if err != nil {
		c.closeDNSServer()
		return fmt.Errorf("dnsServer.Prepare: %s", err)
	}

	c.rdns = InitRDNS(c.dnsServer, &c.clients)
	c.whois = initWhois(&c.clients)

	c.filters.Init()
	return nil
}

func (c *homeContext) isRunning() bool {
	return c.dnsServer != nil && c.dnsServer.IsRunning()
}

func (c *homeContext) onDNSRequest(d *proxy.DNSContext) {
	ip := dnsforward.GetIPString(d.Addr)
	if ip == "" {
		// This would be quite weird if we get here
		return
	}

	ipAddr := net.ParseIP(ip)
	if !ipAddr.IsLoopback() {
		c.rdns.Begin(ip)
	}
	if util.IsPublicIP(ipAddr) {
		c.whois.Begin(ip)
	}
}

func (c *homeContext) generateServerConfig() dnsforward.ServerConfig {
	newconfig := dnsforward.ServerConfig{
		UDPListenAddr:   &net.UDPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		TCPListenAddr:   &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		FilteringConfig: config.DNS.FilteringConfig,
		ConfigModified:  onConfigModified,
		HTTPRegister:    httpRegister,
		OnDNSRequest:    c.onDNSRequest,
	}

	tlsConf := tlsConfigSettings{}
	c.tls.WriteDiskConfig(&tlsConf)
	if tlsConf.Enabled {
		newconfig.TLSConfig = tlsConf.TLSConfig
		if tlsConf.PortDNSOverTLS != 0 {
			newconfig.TLSListenAddr = &net.TCPAddr{
				IP:   net.ParseIP(config.DNS.BindHost),
				Port: tlsConf.PortDNSOverTLS,
			}
		}
	}
	newconfig.TLSv12Roots = c.tlsRoots
	newconfig.TLSCiphers = c.tlsCiphers
	newconfig.TLSAllowUnencryptedDOH = tlsConf.AllowUnencryptedDOH

	newconfig.FilterHandler = c.applyAdditionalFiltering
	newconfig.GetCustomUpstreamByClient = c.clients.FindUpstreams
	return newconfig
}

// Get the list of DNS addresses the server is listening on
func (c *homeContext) getDNSAddresses() []string {
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

	tlsConf := tlsConfigSettings{}
	c.tls.WriteDiskConfig(&tlsConf)
	if tlsConf.Enabled && len(tlsConf.ServerName) != 0 {

		if tlsConf.PortHTTPS != 0 {
			addr := tlsConf.ServerName
			if tlsConf.PortHTTPS != 443 {
				addr = fmt.Sprintf("%s:%d", addr, tlsConf.PortHTTPS)
			}
			addr = fmt.Sprintf("https://%s/dns-query", addr)
			dnsAddresses = append(dnsAddresses, addr)
		}

		if tlsConf.PortDNSOverTLS != 0 {
			addr := fmt.Sprintf("tls://%s:%d", tlsConf.ServerName, tlsConf.PortDNSOverTLS)
			dnsAddresses = append(dnsAddresses, addr)
		}
	}

	return dnsAddresses
}

// If a client has his own settings, apply them
func (c *homeContext) applyAdditionalFiltering(clientAddr string, setts *dnsfilter.RequestFilteringSettings) {
	c.dnsFilter.ApplyBlockedServices(setts, nil, true)

	if len(clientAddr) == 0 {
		return
	}
	setts.ClientIP = clientAddr

	cl, ok := c.clients.Find(clientAddr)
	if !ok {
		return
	}

	log.Debug("Using settings for client %s with IP %s", cl.Name, clientAddr)

	if cl.UseOwnBlockedServices {
		c.dnsFilter.ApplyBlockedServices(setts, cl.BlockedServices, false)
	}

	setts.ClientName = cl.Name
	setts.ClientTags = cl.Tags

	if !cl.UseOwnSettings {
		return
	}

	setts.FilteringEnabled = cl.FilteringEnabled
	setts.SafeSearchEnabled = cl.SafeSearchEnabled
	setts.SafeBrowsingEnabled = cl.SafeBrowsingEnabled
	setts.ParentalEnabled = cl.ParentalEnabled
}

func (c *homeContext) startDNSServer() error {
	if c.isRunning() {
		return fmt.Errorf("unable to start forwarding DNS server: Already running")
	}

	enableFilters(false)

	c.clients.Start()

	err := c.dnsServer.Start()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	c.dnsFilter.Start()
	c.filters.Start()
	c.stats.Start()
	c.queryLog.Start()

	const topClientsNumber = 100 // the number of clients to get
	topClients := c.stats.GetTopClientsIP(topClientsNumber)
	for _, ip := range topClients {
		ipAddr := net.ParseIP(ip)
		if !ipAddr.IsLoopback() {
			c.rdns.Begin(ip)
		}
		if util.IsPublicIP(ipAddr) {
			c.whois.Begin(ip)
		}
	}

	return nil
}

func (c *homeContext) reconfigureDNSServer() error {
	newconfig := c.generateServerConfig()
	err := c.dnsServer.Reconfigure(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func (c *homeContext) stopDNSServer() error {
	if !c.isRunning() {
		return nil
	}

	err := c.dnsServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop forwarding DNS server")
	}

	c.closeDNSServer()
	return nil
}

func (c *homeContext) closeDNSServer() {
	// DNS forward module must be closed BEFORE stats or queryLog because it depends on them
	if c.dnsServer != nil {
		c.dnsServer.Close()
		c.dnsServer = nil
	}

	if c.dnsFilter != nil {
		c.dnsFilter.Close()
		c.dnsFilter = nil
	}

	if c.stats != nil {
		c.stats.Close()
		c.stats = nil
	}

	if c.queryLog != nil {
		c.queryLog.Close()
		c.queryLog = nil
	}

	c.filters.Close()
	log.Debug("Closed all DNS modules")
}
