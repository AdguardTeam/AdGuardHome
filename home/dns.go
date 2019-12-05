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
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

type dnsContext struct {
	rdns  *RDNS
	whois *Whois
}

// Called by other modules when configuration is changed
func onConfigModified() {
	_ = config.write()
}

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer() {
	baseDir := config.getDataDir()

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		log.Fatalf("Cannot create DNS data dir at %s: %s", baseDir, err)
	}

	statsConf := stats.Config{
		Filename:       filepath.Join(baseDir, "stats.db"),
		LimitDays:      config.DNS.StatsInterval,
		ConfigModified: onConfigModified,
		HTTPRegister:   httpRegister,
	}
	config.stats, err = stats.New(statsConf)
	if err != nil {
		log.Fatal("Couldn't initialize statistics module")
	}
	conf := querylog.Config{
		Enabled:        config.DNS.QueryLogEnabled,
		BaseDir:        baseDir,
		Interval:       config.DNS.QueryLogInterval,
		MemSize:        config.DNS.QueryLogMemSize,
		ConfigModified: onConfigModified,
		HTTPRegister:   httpRegister,
	}
	config.queryLog = querylog.New(conf)

	filterConf := config.DNS.DnsfilterConf
	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	filterConf.ResolverAddress = fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)
	filterConf.ConfigModified = onConfigModified
	filterConf.HTTPRegister = httpRegister
	config.dnsFilter = dnsfilter.New(&filterConf, nil)

	config.dnsServer = dnsforward.NewServer(config.dnsFilter, config.stats, config.queryLog)

	sessFilename := filepath.Join(baseDir, "sessions.db")
	config.auth = InitAuth(sessFilename, config.Users, config.WebSessionTTLHours*60*60)
	config.Users = nil

	config.dnsctx.rdns = InitRDNS(&config.clients)
	config.dnsctx.whois = initWhois(&config.clients)

	initFiltering()
}

func isRunning() bool {
	return config.dnsServer != nil && config.dnsServer.IsRunning()
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
		config.dnsctx.rdns.Begin(ip)
	}
	if isPublicIP(ipAddr) {
		config.dnsctx.whois.Begin(ip)
	}
}

func generateServerConfig() (dnsforward.ServerConfig, error) {
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

	newconfig.FilterHandler = applyAdditionalFiltering
	newconfig.GetUpstreamsByClient = getUpstreamsByClient
	return newconfig, nil
}

func getUpstreamsByClient(clientAddr string) []string {
	c, ok := config.clients.Find(clientAddr)
	if !ok {
		return []string{}
	}
	log.Debug("Using upstreams %v for client %s (IP: %s)", c.Upstreams, c.Name, clientAddr)
	return c.Upstreams
}

// If a client has his own settings, apply them
func applyAdditionalFiltering(clientAddr string, setts *dnsfilter.RequestFilteringSettings) {

	ApplyBlockedServices(setts, config.DNS.BlockedServices)

	if len(clientAddr) == 0 {
		return
	}

	c, ok := config.clients.Find(clientAddr)
	if !ok {
		return
	}

	log.Debug("Using settings for client with IP %s", clientAddr)

	if c.UseOwnBlockedServices {
		ApplyBlockedServices(setts, c.BlockedServices)
	}

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

	newconfig, err := generateServerConfig()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	err = config.dnsServer.Start(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	startFiltering()

	const topClientsNumber = 100 // the number of clients to get
	topClients := config.stats.GetTopClientsIP(topClientsNumber)
	for _, ip := range topClients {
		ipAddr := net.ParseIP(ip)
		if !ipAddr.IsLoopback() {
			config.dnsctx.rdns.Begin(ip)
		}
		if isPublicIP(ipAddr) {
			config.dnsctx.whois.Begin(ip)
		}
	}

	return nil
}

func reconfigureDNSServer() error {
	newconfig, err := generateServerConfig()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}
	err = config.dnsServer.Reconfigure(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func stopDNSServer() error {
	if !isRunning() {
		return nil
	}

	err := config.dnsServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop forwarding DNS server")
	}

	// DNS forward module must be closed BEFORE stats or queryLog because it depends on them
	config.dnsServer.Close()

	config.dnsFilter.Close()
	config.dnsFilter = nil

	config.stats.Close()
	config.stats = nil

	config.queryLog.Close()
	config.queryLog = nil

	config.auth.Close()
	config.auth = nil
	return nil
}
