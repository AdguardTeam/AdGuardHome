package home

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
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
func initDNSServer() error {
	baseDir := Context.getDataDir()

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return fmt.Errorf("Cannot create DNS data dir at %s: %s", baseDir, err)
	}

	statsConf := stats.Config{
		Filename:       filepath.Join(baseDir, "stats.db"),
		LimitDays:      config.DNS.StatsInterval,
		ConfigModified: onConfigModified,
		HTTPRegister:   httpRegister,
	}
	Context.stats, err = stats.New(statsConf)
	if err != nil {
		return fmt.Errorf("Couldn't initialize statistics module")
	}
	conf := querylog.Config{
		Enabled:        config.DNS.QueryLogEnabled,
		BaseDir:        baseDir,
		Interval:       config.DNS.QueryLogInterval,
		MemSize:        config.DNS.QueryLogMemSize,
		ConfigModified: onConfigModified,
		HTTPRegister:   httpRegister,
	}
	Context.queryLog = querylog.New(conf)

	filterConf := config.DNS.DnsfilterConf
	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	filterConf.ResolverAddress = fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)
	filterConf.ConfigModified = onConfigModified
	filterConf.HTTPRegister = httpRegister
	Context.dnsFilter = dnsfilter.New(&filterConf, nil)

	Context.dnsServer = dnsforward.NewServer(Context.dnsFilter, Context.stats, Context.queryLog)
	dnsConfig := generateServerConfig()
	err = Context.dnsServer.Prepare(&dnsConfig)
	if err != nil {
		closeDNSServer()
		return fmt.Errorf("dnsServer.Prepare: %s", err)
	}

	sessFilename := filepath.Join(baseDir, "sessions.db")
	Context.auth = InitAuth(sessFilename, config.Users, config.WebSessionTTLHours*60*60)
	if Context.auth == nil {
		closeDNSServer()
		return fmt.Errorf("Couldn't initialize Auth module")
	}
	config.Users = nil

	Context.rdns = InitRDNS(Context.dnsServer, &Context.clients)
	Context.whois = initWhois(&Context.clients)

	initFiltering()
	return nil
}

func isRunning() bool {
	return Context.dnsServer != nil && Context.dnsServer.IsRunning()
}

// nolint (gocyclo)
// Return TRUE if IP is within public Internet IP range
func isPublicIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 != nil {
		switch ip4[0] {
		case 0:
			return false //software
		case 10:
			return false //private network
		case 127:
			return false //loopback
		case 169:
			if ip4[1] == 254 {
				return false //link-local
			}
		case 172:
			if ip4[1] >= 16 && ip4[1] <= 31 {
				return false //private network
			}
		case 192:
			if (ip4[1] == 0 && ip4[2] == 0) || //private network
				(ip4[1] == 0 && ip4[2] == 2) || //documentation
				(ip4[1] == 88 && ip4[2] == 99) || //reserved
				(ip4[1] == 168) { //private network
				return false
			}
		case 198:
			if (ip4[1] == 18 || ip4[2] == 19) || //private network
				(ip4[1] == 51 || ip4[2] == 100) { //documentation
				return false
			}
		case 203:
			if ip4[1] == 0 && ip4[2] == 113 { //documentation
				return false
			}
		case 224:
			if ip4[1] == 0 && ip4[2] == 0 { //multicast
				return false
			}
		case 255:
			if ip4[1] == 255 && ip4[2] == 255 && ip4[3] == 255 { //subnet
				return false
			}
		}
	} else {
		if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
			return false
		}
	}

	return true
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
	if isPublicIP(ipAddr) {
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

	if config.TLS.Enabled {
		newconfig.TLSConfig = config.TLS.TLSConfig
		if config.TLS.PortDNSOverTLS != 0 {
			newconfig.TLSListenAddr = &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.TLS.PortDNSOverTLS}
		}
	}
	newconfig.TLSv12Roots = Context.tlsRoots

	newconfig.FilterHandler = applyAdditionalFiltering
	newconfig.GetUpstreamsByClient = getUpstreamsByClient
	return newconfig
}

func getUpstreamsByClient(clientAddr string) []upstream.Upstream {
	return Context.clients.FindUpstreams(clientAddr)
}

// If a client has his own settings, apply them
func applyAdditionalFiltering(clientAddr string, setts *dnsfilter.RequestFilteringSettings) {
	ApplyBlockedServices(setts, config.DNS.BlockedServices)

	if len(clientAddr) == 0 {
		return
	}

	c, ok := Context.clients.Find(clientAddr)
	if !ok {
		return
	}

	log.Debug("Using settings for client with IP %s", clientAddr)

	if c.UseOwnBlockedServices {
		ApplyBlockedServices(setts, c.BlockedServices)
	}

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

	err := Context.dnsServer.Start()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	Context.dnsFilter.Start()
	startFiltering()
	Context.stats.Start()
	Context.queryLog.Start()

	const topClientsNumber = 100 // the number of clients to get
	topClients := Context.stats.GetTopClientsIP(topClientsNumber)
	for _, ip := range topClients {
		ipAddr := net.ParseIP(ip)
		if !ipAddr.IsLoopback() {
			Context.rdns.Begin(ip)
		}
		if isPublicIP(ipAddr) {
			Context.whois.Begin(ip)
		}
	}

	return nil
}

func reconfigureDNSServer() error {
	newconfig := generateServerConfig()
	err := Context.dnsServer.Reconfigure(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func stopDNSServer() error {
	if !isRunning() {
		return nil
	}

	err := Context.dnsServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop forwarding DNS server")
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

	if Context.auth != nil {
		Context.auth.Close()
		Context.auth = nil
	}

	log.Debug("Closed all DNS modules")
}
