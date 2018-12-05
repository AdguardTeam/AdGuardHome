package dnsforward

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

// Server is the main way to start a DNS server.
//
// Example:
//  s := dnsforward.Server{}
//  err := s.Start(nil) // will start a DNS server listening on default port 53, in a goroutine
//  err := s.Reconfigure(ServerConfig{UDPListenAddr: &net.UDPAddr{Port: 53535}}) // will reconfigure running DNS server to listen on UDP port 53535
//  err := s.Stop() // will stop listening on port 53535 and cancel all goroutines
//  err := s.Start(nil) // will start listening again, on port 53535, in a goroutine
//
// The zero Server is empty and ready for use.
type Server struct {
	udpListen *net.UDPConn

	dnsFilter *dnsfilter.Dnsfilter

	cache cache

	sync.RWMutex
	ServerConfig
}

// uncomment this block to have tracing of locks
/*
func (s *Server) Lock() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> Lock() -> in progress\n", path.Base(file), line, path.Base(f.Name()))
	s.RWMutex.Lock()
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> Lock() -> done\n", path.Base(file), line, path.Base(f.Name()))
}
func (s *Server) RLock() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> RLock() -> in progress\n", path.Base(file), line, path.Base(f.Name()))
	s.RWMutex.RLock()
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> RLock() -> done\n", path.Base(file), line, path.Base(f.Name()))
}
func (s *Server) Unlock() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> Unlock() -> in progress\n", path.Base(file), line, path.Base(f.Name()))
	s.RWMutex.Unlock()
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> Unlock() -> done\n", path.Base(file), line, path.Base(f.Name()))
}
func (s *Server) RUnlock() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> RUnlock() -> in progress\n", path.Base(file), line, path.Base(f.Name()))
	s.RWMutex.RUnlock()
	fmt.Fprintf(os.Stderr, "%s:%d %s() -> RUnlock() -> done\n", path.Base(file), line, path.Base(f.Name()))
}
*/

type FilteringConfig struct {
	ProtectionEnabled  bool   `yaml:"protection_enabled"`
	FilteringEnabled   bool   `yaml:"filtering_enabled"`
	BlockedResponseTTL uint32 `yaml:"blocked_response_ttl"` // if 0, then default is used (3600)

	dnsfilter.Config `yaml:",inline"`
}

// The zero ServerConfig is empty and ready for use.
type ServerConfig struct {
	UDPListenAddr *net.UDPAddr // if nil, then default is is used (port 53 on *)
	Upstreams     []Upstream
	Filters       []dnsfilter.Filter

	FilteringConfig
}

var defaultValues = ServerConfig{
	UDPListenAddr:   &net.UDPAddr{Port: 53},
	FilteringConfig: FilteringConfig{BlockedResponseTTL: 3600},
	Upstreams: []Upstream{
		//// dns over HTTPS
		// &dnsOverHTTPS{address: "https://1.1.1.1/dns-query"},
		// &dnsOverHTTPS{address: "https://dns.google.com/experimental"},
		// &dnsOverHTTPS{address: "https://doh.cleanbrowsing.org/doh/security-filter/"},
		// &dnsOverHTTPS{address: "https://dns10.quad9.net/dns-query"},
		// &dnsOverHTTPS{address: "https://doh.powerdns.org"},
		// &dnsOverHTTPS{address: "https://doh.securedns.eu/dns-query"},

		//// dns over TLS
		// &dnsOverTLS{address: "tls://8.8.8.8:853"},
		// &dnsOverTLS{address: "tls://8.8.4.4:853"},
		// &dnsOverTLS{address: "tls://1.1.1.1:853"},
		// &dnsOverTLS{address: "tls://1.0.0.1:853"},

		//// plainDNS
		&plainDNS{address: "8.8.8.8:53"},
		&plainDNS{address: "8.8.4.4:53"},
		&plainDNS{address: "1.1.1.1:53"},
		&plainDNS{address: "1.0.0.1:53"},
	},
}

//
// packet loop
//
func (s *Server) packetLoop() {
	log.Printf("Entering packet handle loop")
	b := make([]byte, dns.MaxMsgSize)
	for {
		s.RLock()
		conn := s.udpListen
		s.RUnlock()
		if conn == nil {
			log.Printf("udp socket has disappeared, exiting loop")
			break
		}
		n, addr, err := conn.ReadFrom(b)
		// documentation says to handle the packet even if err occurs, so do that first
		if n > 0 {
			// make a copy of all bytes because ReadFrom() will overwrite contents of b on next call
			// we need the contents to survive the call because we're handling them in goroutine
			p := make([]byte, n)
			copy(p, b)
			go s.handlePacket(p, addr, conn) // ignore errors
		}
		if err != nil {
			if isConnClosed(err) {
				log.Printf("ReadFrom() returned because we're reading from a closed connection, exiting loop")
				// don't try to nullify s.udpListen here, because s.udpListen could be already re-bound to listen
				break
			}
			log.Printf("Got error when reading from udp listen: %s", err)
		}
	}
}

//
// Control functions
//

func (s *Server) Start(config *ServerConfig) error {
	s.Lock()
	defer s.Unlock()
	if config != nil {
		s.ServerConfig = *config
	}
	// TODO: handle being called Start() second time after Stop()
	if s.udpListen == nil {
		log.Printf("Creating UDP socket")
		var err error
		addr := s.UDPListenAddr
		if addr == nil {
			addr = defaultValues.UDPListenAddr
		}
		s.udpListen, err = net.ListenUDP("udp", addr)
		if err != nil {
			s.udpListen = nil
			return errorx.Decorate(err, "Couldn't listen to UDP socket")
		}
		log.Println(s.udpListen.LocalAddr(), s.UDPListenAddr)
	}

	if s.dnsFilter == nil {
		log.Printf("Creating dnsfilter")
		s.dnsFilter = dnsfilter.New(&s.Config)
		// add rules only if they are enabled
		if s.FilteringEnabled {
			s.dnsFilter.AddRules(s.Filters)
		}
	}

	log.Printf("Loading stats from querylog")
	err := fillStatsFromQueryLog()
	if err != nil {
		log.Printf("Failed to load stats from querylog: %s", err)
		return err
	}

	once.Do(func() {
		go periodicQueryLogRotate()
		go periodicHourlyTopRotate()
		go statsRotator()
	})

	go s.packetLoop()

	return nil
}

func (s *Server) Stop() error {
	s.Lock()
	defer s.Unlock()
	if s.udpListen != nil {
		err := s.udpListen.Close()
		s.udpListen = nil
		if err != nil {
			return errorx.Decorate(err, "Couldn't close UDP listening socket")
		}
	}

	// flush remainder to file
	logBufferLock.Lock()
	flushBuffer := logBuffer
	logBuffer = nil
	logBufferLock.Unlock()
	err := flushToFile(flushBuffer)
	if err != nil {
		log.Printf("Saving querylog to file failed: %s", err)
		return err
	}

	return nil
}

func (s *Server) IsRunning() bool {
	s.RLock()
	isRunning := true
	if s.udpListen == nil {
		isRunning = false
	}
	s.RUnlock()
	return isRunning
}

//
// Server reconfigure
//

func (s *Server) reconfigureListenAddr(new ServerConfig) error {
	oldAddr := s.UDPListenAddr
	if oldAddr == nil {
		oldAddr = defaultValues.UDPListenAddr
	}
	newAddr := new.UDPListenAddr
	if newAddr == nil {
		newAddr = defaultValues.UDPListenAddr
	}
	if newAddr.Port == 0 {
		return errorx.IllegalArgument.New("new port cannot be 0")
	}
	if reflect.DeepEqual(oldAddr, newAddr) {
		// do nothing, the addresses are exactly the same
		log.Printf("Not going to rebind because addresses are same: %v -> %v", oldAddr, newAddr)
		return nil
	}

	// rebind, using a strategy:
	// * if ports are different, bind new first, then close old
	// * if ports are same, close old first, then bind new
	var newListen *net.UDPConn
	var err error
	if oldAddr.Port != newAddr.Port {
		log.Printf("Rebinding -- ports are different so bind first then close")
		newListen, err = net.ListenUDP("udp", newAddr)
		if err != nil {
			return errorx.Decorate(err, "Couldn't bind to %v", newAddr)
		}
		s.Lock()
		if s.udpListen != nil {
			err = s.udpListen.Close()
			s.udpListen = nil
		}
		s.Unlock()
		if err != nil {
			return errorx.Decorate(err, "Couldn't close UDP listening socket")
		}
	} else {
		log.Printf("Rebinding -- ports are same so close first then bind")
		s.Lock()
		if s.udpListen != nil {
			err = s.udpListen.Close()
			s.udpListen = nil
		}
		s.Unlock()
		if err != nil {
			return errorx.Decorate(err, "Couldn't close UDP listening socket")
		}
		newListen, err = net.ListenUDP("udp", newAddr)
		if err != nil {
			return errorx.Decorate(err, "Couldn't bind to %v", newAddr)
		}
	}
	s.Lock()
	s.udpListen = newListen
	s.UDPListenAddr = new.UDPListenAddr
	s.Unlock()
	log.Println(s.udpListen.LocalAddr(), s.UDPListenAddr)

	go s.packetLoop() // the old one has quit, use new one

	return nil
}

func (s *Server) reconfigureBlockedResponseTTL(new ServerConfig) {
	newVal := new.BlockedResponseTTL
	if newVal == 0 {
		newVal = defaultValues.BlockedResponseTTL
	}
	oldVal := s.BlockedResponseTTL
	if oldVal == 0 {
		oldVal = defaultValues.BlockedResponseTTL
	}
	if newVal != oldVal {
		s.BlockedResponseTTL = new.BlockedResponseTTL
	}
}

func (s *Server) reconfigureUpstreams(new ServerConfig) {
	newVal := new.Upstreams
	if len(newVal) == 0 {
		newVal = defaultValues.Upstreams
	}
	oldVal := s.Upstreams
	if len(oldVal) == 0 {
		oldVal = defaultValues.Upstreams
	}
	if reflect.DeepEqual(newVal, oldVal) {
		// they're exactly the same, do nothing
		return
	}
	s.Upstreams = new.Upstreams
}

func (s *Server) reconfigureFiltering(new ServerConfig) {
	newFilters := new.Filters
	if len(newFilters) == 0 {
		newFilters = defaultValues.Filters
	}
	oldFilters := s.Filters
	if len(oldFilters) == 0 {
		oldFilters = defaultValues.Filters
	}

	needUpdate := false
	if !reflect.DeepEqual(newFilters, oldFilters) {
		needUpdate = true
	}

	if !reflect.DeepEqual(new.FilteringConfig, s.FilteringConfig) {
		needUpdate = true
	}

	if !needUpdate {
		// nothing to do, everything is same
		return
	}

	// TODO: instead of creating new dnsfilter, change existing one's settings and filters
	dnsFilter := dnsfilter.New(&new.Config) // sets safebrowsing, safesearch and parental

	// add rules only if they are enabled
	if new.FilteringEnabled {
		dnsFilter.AddRules(newFilters)
	}

	s.Lock()
	oldDNSFilter := s.dnsFilter
	s.dnsFilter = dnsFilter
	s.FilteringConfig = new.FilteringConfig
	s.Unlock()

	oldDNSFilter.Destroy()
}

func (s *Server) Reconfigure(new ServerConfig) error {
	s.reconfigureBlockedResponseTTL(new)
	s.reconfigureUpstreams(new)
	s.reconfigureFiltering(new)

	err := s.reconfigureListenAddr(new)
	if err != nil {
		return errorx.Decorate(err, "Couldn't reconfigure to new listening address %+v", new.UDPListenAddr)
	}
	return nil
}

//
// packet handling functions
//

// handlePacketInternal processes the incoming packet bytes and returns with an optional response packet.
//
// If an empty dns.Msg is returned, do not try to send anything back to client, otherwise send contents of dns.Msg.
//
// If an error is returned, log it, don't try to generate data based on that error.
func (s *Server) handlePacketInternal(msg *dns.Msg, addr net.Addr, conn *net.UDPConn) (*dns.Msg, *dnsfilter.Result, Upstream, error) {
	// log.Printf("Got packet %d bytes from %s: %v", len(p), addr, p)
	//
	// DNS packet byte format is valid
	//
	// any errors below here require a response to client
	// log.Printf("Unpacked: %v", msg.String())
	if len(msg.Question) != 1 {
		log.Printf("Got invalid number of questions: %v", len(msg.Question))
		return s.genServerFailure(msg), nil, nil, nil
	}

	// use dnsfilter before cache -- changed settings or filters would require cache invalidation otherwise
	host := strings.TrimSuffix(msg.Question[0].Name, ".")
	res, err := s.dnsFilter.CheckHost(host)
	if err != nil {
		log.Printf("dnsfilter failed to check host '%s': %s", host, err)
		return s.genServerFailure(msg), &res, nil, err
	} else if res.IsFiltered {
		log.Printf("Host %s is filtered, reason - '%s', matched rule: '%s'", host, res.Reason, res.Rule)
		return s.genNXDomain(msg), &res, nil, nil
	}

	{
		val, ok := s.cache.Get(msg)
		if ok && val != nil {
			return val, &res, nil, nil
		}
	}

	// TODO: replace with single-socket implementation
	upstream := s.chooseUpstream()
	reply, err := upstream.Exchange(msg)
	if err != nil {
		log.Printf("talking to upstream failed for host '%s': %s", host, err)
		return s.genServerFailure(msg), &res, upstream, err
	}
	if reply == nil {
		log.Printf("SHOULD NOT HAPPEN upstream returned empty message for host '%s'. Request is %v", host, msg.String())
		return s.genServerFailure(msg), &res, upstream, nil
	}

	s.cache.Set(reply)

	return reply, &res, upstream, nil
}

func (s *Server) handlePacket(p []byte, addr net.Addr, conn *net.UDPConn) {
	start := time.Now()

	msg := &dns.Msg{}
	err := msg.Unpack(p)
	if err != nil {
		log.Printf("got invalid DNS packet: %s", err)
		return // do nothing
	}

	reply, result, upstream, err := s.handlePacketInternal(msg, addr, conn)
	if reply != nil {
		rerr := s.respond(reply, addr, conn)
		if rerr != nil {
			log.Printf("Couldn't respond to UDP packet: %s", err)
		}
	}

	// query logging and stats counters
	elapsed := time.Since(start)
	upstreamAddr := ""
	if upstream != nil {
		upstreamAddr = upstream.Address()
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		log.Printf("Failed to split %v into host/port: %s", addr, err)
	}
	logRequest(msg, reply, result, elapsed, host, upstreamAddr)
}

//
// packet sending functions
//

func (s *Server) respond(resp *dns.Msg, addr net.Addr, conn *net.UDPConn) error {
	// log.Printf("Replying to %s with %s", addr, resp)
	resp.Compress = true
	bytes, err := resp.Pack()
	if err != nil {
		return errorx.Decorate(err, "Couldn't convert message into wire format")
	}
	n, err := conn.WriteTo(bytes, addr)
	if n == 0 && isConnClosed(err) {
		return err
	}
	if n != len(bytes) {
		return fmt.Errorf("WriteTo() returned with %d != %d", n, len(bytes))
	}
	if err != nil {
		return errorx.Decorate(err, "WriteTo() returned error")
	}
	return nil
}

func (s *Server) genServerFailure(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeServerFailure)
	return &resp
}

func (s *Server) genNXDomain(request *dns.Msg) *dns.Msg {
	resp := dns.Msg{}
	resp.SetRcode(request, dns.RcodeNameError)
	resp.Ns = s.genSOA(request)
	return &resp
}

func (s *Server) genSOA(request *dns.Msg) []dns.RR {
	zone := ""
	if len(request.Question) > 0 {
		zone = request.Question[0].Name
	}

	soa := dns.SOA{
		// values copied from verisign's nonexistent .com domain
		// their exact values are not important in our use case because they are used for domain transfers between primary/secondary DNS servers
		Refresh: 1800,
		Retry:   900,
		Expire:  604800,
		Minttl:  86400,
		// copied from AdGuard DNS
		Ns:     "fake-for-negative-caching.adguard.com.",
		Serial: 100500,
		// rest is request-specific
		Hdr: dns.RR_Header{
			Name:   zone,
			Rrtype: dns.TypeSOA,
			Ttl:    s.BlockedResponseTTL,
			Class:  dns.ClassINET,
		},
		Mbox: "hostmaster.", // zone will be appended later if it's not empty or "."
	}
	if soa.Hdr.Ttl == 0 {
		soa.Hdr.Ttl = defaultValues.BlockedResponseTTL
	}
	if len(zone) > 0 && zone[0] != '.' {
		soa.Mbox += zone
	}
	return []dns.RR{&soa}
}

var once sync.Once
