package home

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

type dnsContext struct {
	rdnsChannel chan string // pass data from DNS request handling thread to rDNS thread
	// contains IP addresses of clients to be resolved by rDNS
	// if IP address couldn't be resolved, it stays here forever to prevent further attempts to resolve the same IP
	rdnsIP   map[string]bool
	rdnsLock sync.Mutex        // synchronize access to rdnsIP
	upstream upstream.Upstream // Upstream object for our own DNS server
}

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer(baseDir string) {
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		log.Fatalf("Cannot create DNS data dir at %s: %s", baseDir, err)
	}

	config.stats = stats.New("./data/stats.db", int(config.DNS.StatsInterval), nil)
	if config.stats == nil {
		log.Fatal("config.stats == nil")
	}
	config.dnsServer = dnsforward.NewServer(baseDir, config.stats)

	initRDNS()
}

func isRunning() bool {
	return config.dnsServer != nil && config.dnsServer.IsRunning()
}

func onDNSRequest(d *proxy.DNSContext) {
	qType := d.Req.Question[0].Qtype
	if qType != dns.TypeA && qType != dns.TypeAAAA {
		return
	}

	ip := dnsforward.GetIPString(d.Addr)
	if ip == "" {
		// This would be quite weird if we get here
		return
	}

	ipAddr := net.ParseIP(ip)
	if !ipAddr.IsLoopback() {
		beginAsyncRDNS(ip)
	}
}

func generateServerConfig() (dnsforward.ServerConfig, error) {
	filters := []dnsfilter.Filter{}
	userFilter := userFilter()
	filters = append(filters, dnsfilter.Filter{
		ID:   userFilter.ID,
		Data: userFilter.Data,
	})
	for _, filter := range config.Filters {
		if !filter.Enabled {
			continue
		}
		filters = append(filters, dnsfilter.Filter{
			ID:       filter.ID,
			FilePath: filter.Path(),
		})
	}

	newconfig := dnsforward.ServerConfig{
		UDPListenAddr:   &net.UDPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		TCPListenAddr:   &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		FilteringConfig: config.DNS.FilteringConfig,
		Filters:         filters,
	}
	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	newconfig.ResolverAddress = fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)

	if config.TLS.Enabled {
		newconfig.TLSConfig = config.TLS.TLSConfig
		if config.TLS.PortDNSOverTLS != 0 {
			newconfig.TLSListenAddr = &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.TLS.PortDNSOverTLS}
		}
	}

	upstreamConfig, err := proxy.ParseUpstreamsConfig(config.DNS.UpstreamDNS, config.DNS.BootstrapDNS, dnsforward.DefaultTimeout)
	if err != nil {
		return newconfig, fmt.Errorf("Couldn't get upstreams configuration cause: %s", err)
	}
	newconfig.Upstreams = upstreamConfig.Upstreams
	newconfig.DomainsReservedUpstreams = upstreamConfig.DomainReservedUpstreams
	newconfig.AllServers = config.DNS.AllServers
	newconfig.FilterHandler = applyAdditionalFiltering
	newconfig.OnDNSRequest = onDNSRequest
	return newconfig, nil
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

	newconfig, err := generateServerConfig()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}
	err = config.dnsServer.Start(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
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

	config.stats.Close()

	return nil
}
