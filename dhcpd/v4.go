package dhcpd

import (
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/krolaw/dhcp4"
)

// V4Server - DHCPv4 server
type V4Server struct {
	srv        *server4.Server
	leasesLock sync.Mutex
	leases     []*Lease
	// IP address pool -- if entry is in the pool, then it's attached to a lease
	IPpool map[[4]byte]net.HardwareAddr

	conf V4ServerConf
}

// V4ServerConf - server configuration
type V4ServerConf struct {
	Enabled bool `yaml:"enabled"`
	// RangeStart    string `yaml:"range_start"`
	LeaseDuration uint32 `yaml:"lease_duration"` // in seconds

	// ipStart    net.IP
	leaseTime time.Duration
	// dnsIPAddrs []net.IP // IPv6 addresses to return to DHCP clients as DNS server addresses
	// sid        dhcpv6.Duid

	// notify func(uint32)
}

// Start - start server
func (s *V4Server) Start(iface net.Interface) error {
	if s.conn != nil {
		_ = s.closeConn()
	}

	c, err := newFilterConn(iface, ":67") // it has to be bound to 0.0.0.0:67, otherwise it won't see DHCP discover/request packets
	if err != nil {
		return wrapErrPrint(err, "Couldn't start listening socket on 0.0.0.0:67")
	}
	log.Info("DHCP: listening on 0.0.0.0:67")

	s.conn = c
	s.cond = sync.NewCond(&s.mutex)

	s.running = true
	go func() {
		// operate on c instead of c.conn because c.conn can change over time
		err := dhcp4.Serve(c, s)
		if err != nil && !s.stopping {
			log.Printf("dhcp4.Serve() returned with error: %s", err)
		}
		_ = c.Close() // in case Serve() exits for other reason than listening socket closure
		s.running = false
		s.cond.Signal()
	}()
	return nil
}

// Reset - stop server
func (s *V4Server) Reset() {
	s.leasesLock.Lock()
	s.leases = nil
	s.IPpool = make(map[[4]byte]net.HardwareAddr)
	s.leasesLock.Unlock()
}

// Stop - stop server
func (s *V4Server) Stop() {
}

// Create DHCPv6 server
func v4Create(conf V4ServerConf) (*V4Server, error) {
	s := &V4Server{}
	s.conf = conf

	if !conf.Enabled {
		return s, nil
	}

	// s.conf.ipStart = net.ParseIP(conf.RangeStart)
	// if s.conf.ipStart == nil {
	// 	return nil, fmt.Errorf("DHCPv6: invalid range-start IP: %s", conf.RangeStart)
	// }

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 2
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s, nil
}
