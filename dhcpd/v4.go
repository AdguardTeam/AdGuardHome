package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/sparrc/go-ping"
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
	Enabled       bool   `json:"enabled" yaml:"enabled"`
	InterfaceName string `json:"interface_name" yaml:"interface_name"` // eth0, en0 and so on
	GatewayIP     string `json:"gateway_ip" yaml:"gateway_ip"`
	SubnetMask    string `json:"subnet_mask" yaml:"subnet_mask"`
	RangeStart    string `json:"range_start" yaml:"range_start"`
	RangeEnd      string `json:"range_end" yaml:"range_end"`
	LeaseDuration uint32 `json:"lease_duration" yaml:"lease_duration"` // in seconds

	// IP conflict detector: time (ms) to wait for ICMP reply.
	// 0: disable
	ICMPTimeout uint32 `json:"icmp_timeout_msec" yaml:"icmp_timeout_msec"`

	ipStart    net.IP
	ipStop     net.IP
	leaseTime  time.Duration
	dnsIPAddrs []net.IP // IPv4 addresses to return to DHCP clients as DNS server addresses
	routerIP   net.IP
	subnetMask net.IPMask

	notify func(uint32)
}

// WriteDiskConfig - write configuration
func (s *V4Server) WriteDiskConfig(c *V4ServerConf) {
	*c = s.conf
}

func ipInRange(start, stop, ip net.IP) bool {
	if len(start) != len(stop) ||
		len(start) != len(ip) {
		return false
	}
	// return dhcp4.IPInRange(start, stop, ip)
	return false
}

// ResetLeases - reset leases
func (s *V4Server) ResetLeases(ll []*Lease) {
	s.leases = nil
	s.IPpool = make(map[[4]byte]net.HardwareAddr)
	for _, l := range ll {

		if l.Expiry.Unix() != leaseExpireStatic &&
			!ipInRange(s.conf.ipStart, s.conf.ipStop, l.IP) {

			log.Tracef("DHCPv4: skipping a lease with IP %v: not within current IP range", l.IP)
			continue
		}

		s.leases = append(s.leases, l)
		s.reserveIP(l.IP, l.HWAddr)
	}
}

// GetLeases returns the list of current DHCP leases (thread-safe)
func (s *V4Server) GetLeases(flags int) []Lease {
	var result []Lease
	now := time.Now().Unix()
	s.leasesLock.Lock()
	for _, lease := range s.leases {
		if ((flags&LeasesDynamic) != 0 && lease.Expiry.Unix() > now) ||
			((flags&LeasesStatic) != 0 && lease.Expiry.Unix() == leaseExpireStatic) {
			result = append(result, *lease)
		}
	}
	s.leasesLock.Unlock()
	return result
}

// FindMACbyIP4 - find a MAC address by IP address in the currently active DHCP leases
func (s *V4Server) FindMACbyIP4(ip net.IP) net.HardwareAddr {
	now := time.Now().Unix()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}

	for _, l := range s.leases {
		if l.IP.Equal(ip4) {
			unix := l.Expiry.Unix()
			if unix > now || unix == leaseExpireStatic {
				return l.HWAddr
			}
		}
	}
	return nil
}

func (s *V4Server) reserveIP(ip net.IP, hwaddr net.HardwareAddr) {
	rawIP := []byte(ip)
	IP4 := [4]byte{rawIP[0], rawIP[1], rawIP[2], rawIP[3]}
	s.IPpool[IP4] = hwaddr
}

func (s *V4Server) unreserveIP(ip net.IP) {
	rawIP := []byte(ip)
	IP4 := [4]byte{rawIP[0], rawIP[1], rawIP[2], rawIP[3]}
	delete(s.IPpool, IP4)
}

func (s *V4Server) findReservedHWaddr(ip net.IP) net.HardwareAddr {
	rawIP := []byte(ip)
	IP4 := [4]byte{rawIP[0], rawIP[1], rawIP[2], rawIP[3]}
	return s.IPpool[IP4]
}

// Add the specified IP to the black list for a time period
func (s *V4Server) blacklistLease(lease *Lease) {
	hw := make(net.HardwareAddr, 6)
	s.leasesLock.Lock()
	s.reserveIP(lease.IP, hw)
	lease.HWAddr = hw
	lease.Hostname = ""
	lease.Expiry = time.Now().Add(s.conf.leaseTime)
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()
	s.conf.notify(LeaseChangedBlacklisted)
}

// Remove a dynamic lease by IP address
func (s *V4Server) rmDynamicLeaseWithIP(ip net.IP) error {
	var newLeases []*Lease
	for _, lease := range s.leases {
		if net.IP.Equal(lease.IP.To4(), ip) {
			if lease.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease with the same IP already exists")
			}
			continue
		}
		newLeases = append(newLeases, lease)
	}
	s.leases = newLeases
	s.unreserveIP(ip)
	return nil
}

// Remove a dynamic lease by IP address
func (s *V4Server) rmDynamicLeaseWithMAC(mac net.HardwareAddr) error {
	var newLeases []*Lease
	for _, lease := range s.leases {
		if bytes.Equal(lease.HWAddr, mac) {
			if lease.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease with the same IP already exists")
			}
			s.unreserveIP(lease.IP)
			continue
		}
		newLeases = append(newLeases, lease)
	}
	s.leases = newLeases
	return nil
}

// Remove a lease
func (s *V4Server) rmLease(l Lease) error {
	var newLeases []*Lease
	for _, lease := range s.leases {
		if net.IP.Equal(lease.IP.To4(), l.IP) {
			if !bytes.Equal(lease.HWAddr, l.HWAddr) ||
				lease.Hostname != l.Hostname {
				return fmt.Errorf("Lease not found")
			}
			continue
		}
		newLeases = append(newLeases, lease)
	}
	s.leases = newLeases
	s.unreserveIP(l.IP)
	return nil
}

// AddStaticLease adds a static lease (thread-safe)
func (s *V4Server) AddStaticLease(l Lease) error {
	if len(l.IP) != 4 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}
	l.Expiry = time.Unix(leaseExpireStatic, 0)

	s.leasesLock.Lock()

	if s.findReservedHWaddr(l.IP) != nil {
		err := s.rmDynamicLeaseWithIP(l.IP)
		if err != nil {
			s.leasesLock.Unlock()
			return err
		}
	} else {
		err := s.rmDynamicLeaseWithMAC(l.HWAddr)
		if err != nil {
			s.leasesLock.Unlock()
			return err
		}
	}
	s.leases = append(s.leases, &l)
	s.reserveIP(l.IP, l.HWAddr)
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()
	s.conf.notify(LeaseChangedAddedStatic)
	return nil
}

// RemoveStaticLease removes a static lease (thread-safe)
func (s *V4Server) RemoveStaticLease(l Lease) error {
	if len(l.IP) != 4 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}

	s.leasesLock.Lock()

	if s.findReservedHWaddr(l.IP) == nil {
		s.leasesLock.Unlock()
		return fmt.Errorf("lease not found")
	}

	err := s.rmLease(l)
	if err != nil {
		s.leasesLock.Unlock()
		return err
	}
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()
	s.conf.notify(LeaseChangedRemovedStatic)
	return nil
}

// Send ICMP to the specified machine
// Return TRUE if it doesn't reply, which probably means that the IP is available
func (s *V4Server) addrAvailable(target net.IP) bool {

	if s.conf.ICMPTimeout == 0 {
		return true
	}

	pinger, err := ping.NewPinger(target.String())
	if err != nil {
		log.Error("ping.NewPinger(): %v", err)
		return true
	}

	pinger.SetPrivileged(true)
	pinger.Timeout = time.Duration(s.conf.ICMPTimeout) * time.Millisecond
	pinger.Count = 1
	reply := false
	pinger.OnRecv = func(pkt *ping.Packet) {
		// log.Tracef("Received ICMP Reply from %v", target)
		reply = true
	}
	log.Tracef("Sending ICMP Echo to %v", target)
	pinger.Run()

	if reply {
		log.Info("DHCP: IP conflict: %v is already used by another device", target)
		return false
	}

	log.Tracef("ICMP procedure is complete: %v", target)
	return true
}

func (s *V4Server) findLease(mac net.HardwareAddr) *Lease {
	return nil
}

func (s *V4Server) reserveLease(mac net.HardwareAddr) *Lease {
	return nil
}

func (s *V4Server) commitLease(req *dhcpv4.DHCPv4, lease *Lease) time.Duration {
	return 0
}

func (s *V4Server) checkIA(req *dhcpv4.DHCPv4, lease *Lease) error {
	// req.GetOneOption(dhcpv4.OptionServerIdentifier)
	return nil
}

// Find a lease associated with MAC and prepare response
func (s *V4Server) process(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) bool {
	mac := req.ClientHWAddr

	// lock

	lease := s.findLease(mac)
	if lease == nil {
		log.Debug("DHCPv6: no lease for: %s", mac)

		switch req.MessageType() {

		case dhcpv4.MessageTypeDiscover:
			lease = s.reserveLease(mac)
			if lease == nil {
				return false
			}

		default:
			return false
		}
	}

	err := s.checkIA(req, lease)
	if err != nil {
		log.Debug("DHCPv4: %s", err)

		// return NAK

		return false
	}

	lifetime := s.commitLease(req, lease)
	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(lifetime))

	// resp.UpdateOption(dhcpv4.OptRequestedIPAddress(x))
	resp.UpdateOption(dhcpv4.OptServerIdentifier(x))
	resp.UpdateOption(dhcpv4.OptDNS(s.conf.dnsIPAddrs...))
	resp.UpdateOption(dhcpv4.OptRouter(s.conf.routerIP))
	resp.UpdateOption(dhcpv4.OptSubnetMask(s.conf.subnetMask))
	return true
}

// client -> Discover -> server
// client <- Reply <- server
// client -> Request -> server
// client <- Reply <- server
func (s *V4Server) packetHandler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	log.Debug("DHCPv4: received message: %s", req.Summary())

	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover,
		dhcpv4.MessageTypeRequest:
		//

	default:
		log.Debug("DHCPv4: unsupported message type %d", req.MessageType())
		return
	}

	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		log.Debug("DHCPv4: dhcpv4.New: %s", err)
		return
	}

	_ = s.process(req, resp)

	log.Debug("DHCPv4: sending: %s", resp.Summary())

	_, err = conn.WriteTo(resp.ToBytes(), peer)
	if err != nil {
		log.Error("DHCPv4: conn.Write to %s failed: %s", peer, err)
		return
	}
}

// Get IPv4 address list
func getIfaceIPv4(iface net.Interface) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}

	var res []net.IP
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if ipnet.IP.To4() != nil {
			res = append(res, ipnet.IP)
		}
	}
	return res
}

// Start - start server
func (s *V4Server) Start() error {
	iface, err := net.InterfaceByName(s.conf.InterfaceName)
	if err != nil {
		return wrapErrPrint(err, "DHCPv4: Couldn't find interface by name %s", s.conf.InterfaceName)
	}

	log.Debug("DHCPv4: starting...")
	s.conf.dnsIPAddrs = getIfaceIPv4(*iface)
	if len(s.conf.dnsIPAddrs) == 0 {
		return fmt.Errorf("DHCPv4: no IPv4 address for interface %s", iface.Name)
	}

	laddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: dhcpv4.ServerPort,
	}

	server, err := server4.NewServer(iface.Name, laddr, s.packetHandler, server4.WithDebugLogger())
	if err != nil {
		return err
	}

	log.Info("DHCPv4: listening")

	go func() {
		err = server.Serve()
		log.Error("DHCPv4: %s", err)
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
	if s.srv == nil {
		return
	}

	err := s.srv.Close()
	if err != nil {
		log.Error("DHCPv4: srv.Close: %s", err)
	}
	// now server.Serve() will return
}

// Create DHCPv6 server
func v4Create(conf V4ServerConf) (*V4Server, error) {
	s := &V4Server{}
	s.conf = conf

	if !conf.Enabled {
		return s, nil
	}

	s.conf.routerIP = x
	s.conf.subnetMask = x

	// s.conf.ipStart = net.ParseIP(conf.RangeStart)
	// if s.conf.ipStart == nil {
	// 	return nil, fmt.Errorf("DHCPv6: invalid range-start IP: %s", conf.RangeStart)
	// }

	// s.conf.ICMPTimeout = 1000

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 24
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s, nil
}
