package dhcpd

import (
	"bytes"
	"encoding/binary"
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
	ipAddrs    [256]byte

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
	ipEnd      net.IP
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

// Return TRUE if IP address is within range [start..stop]
func ipInRange(start net.IP, stop net.IP, ip net.IP) bool {
	from := binary.BigEndian.Uint32(start)
	to := binary.BigEndian.Uint32(stop)
	check := binary.BigEndian.Uint32(ip)
	return from <= check && check <= to
}

// ResetLeases - reset leases
func (s *V4Server) ResetLeases(leases []*Lease) {
	s.leases = nil

	for _, l := range leases {

		if l.Expiry.Unix() != leaseExpireStatic &&
			!ipInRange(s.conf.ipStart, s.conf.ipEnd, l.IP) {

			log.Debug("DHCPv4: skipping a lease with IP %v: not within current IP range", l.IP)
			continue
		}

		s.addLease(l)
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

// Add the specified IP to the black list for a time period
func (s *V4Server) blacklistLease(lease *Lease) {
	hw := make(net.HardwareAddr, 6)
	lease.HWAddr = hw
	lease.Hostname = ""
	lease.Expiry = time.Now().Add(s.conf.leaseTime)
}

// Remove (swap) lease by index
func (s *V4Server) leaseRemoveSwapByIndex(i int) {
	s.ipAddrs[s.leases[i].IP[3]] = 0
	log.Debug("DHCPv4: removed lease %s", s.leases[i].HWAddr)

	n := len(s.leases)
	if i != n-1 {
		s.leases[i] = s.leases[n-1] // swap with the last element
	}
	s.leases = s.leases[:n-1]
}

// Remove a dynamic lease with the same properties
// Return error if a static lease is found
func (s *V4Server) rmDynamicLease(lease Lease) error {
	for i := 0; i < len(s.leases); i++ {
		l := s.leases[i]

		if bytes.Equal(l.HWAddr, lease.HWAddr) {

			if l.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.leaseRemoveSwapByIndex(i)
			l = s.leases[i]
		}

		if bytes.Equal(l.IP, lease.IP) {

			if l.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.leaseRemoveSwapByIndex(i)
		}
	}
	return nil
}

// Add a lease
func (s *V4Server) addLease(l *Lease) {
	s.leases = append(s.leases, l)
	s.ipAddrs[l.IP[3]] = 1
	log.Debug("DHCPv4: added lease %s <-> %s", l.IP, l.HWAddr)
}

// Remove a lease with the same properies
func (s *V4Server) rmLease(lease Lease) error {
	for i, l := range s.leases {
		if bytes.Equal(l.IP, lease.IP) {

			if !bytes.Equal(l.HWAddr, lease.HWAddr) ||
				l.Hostname != lease.Hostname {

				return fmt.Errorf("Lease not found")
			}

			s.leaseRemoveSwapByIndex(i)
			return nil
		}
	}
	return fmt.Errorf("lease not found")
}

// AddStaticLease adds a static lease (thread-safe)
func (s *V4Server) AddStaticLease(lease Lease) error {
	if len(lease.IP) != 4 {
		return fmt.Errorf("invalid IP")
	}
	if len(lease.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}
	lease.Expiry = time.Unix(leaseExpireStatic, 0)

	s.leasesLock.Lock()
	err := s.rmDynamicLease(lease)
	if err != nil {
		s.leasesLock.Unlock()
		return err
	}
	s.addLease(&lease)
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
		reply = true
	}
	log.Debug("DHCPv4: Sending ICMP Echo to %v", target)
	pinger.Run()

	if reply {
		log.Info("DHCPv4: IP conflict: %v is already used by another device", target)
		return false
	}

	log.Debug("DHCPv4: ICMP procedure is complete: %v", target)
	return true
}

// Find lease by MAC
func (s *V4Server) findLease(mac net.HardwareAddr) *Lease {
	for i := range s.leases {
		if bytes.Equal(mac, s.leases[i].HWAddr) {
			return s.leases[i]
		}
	}
	return nil
}

// Get next free IP
func (s *V4Server) findFreeIP() net.IP {
	for i := s.conf.ipStart[3]; ; i++ {
		if s.ipAddrs[i] == 0 {
			ip := make([]byte, 4)
			copy(ip, s.conf.ipStart)
			ip[3] = i
			return ip
		}
		if i == s.conf.ipEnd[3] {
			break
		}
	}
	return nil
}

// Find an expired lease and return its index or -1
func (s *V4Server) findExpiredLease() int {
	now := time.Now().Unix()
	for i, lease := range s.leases {
		if lease.Expiry.Unix() != leaseExpireStatic &&
			lease.Expiry.Unix() <= now {
			return i
		}
	}
	return -1
}

// Reserve lease for MAC
func (s *V4Server) reserveLease(mac net.HardwareAddr) *Lease {
	l := Lease{}
	l.HWAddr = make([]byte, 6)
	copy(l.HWAddr, mac)

	l.IP = s.findFreeIP()
	if l.IP == nil {
		i := s.findExpiredLease()
		if i < 0 {
			return nil
		}
		copy(s.leases[i].HWAddr, mac)
		return s.leases[i]
	}

	s.addLease(&l)
	return &l
}

// Find a lease associated with MAC and prepare response
func (s *V4Server) process(req *dhcpv4.DHCPv4, resp *dhcpv4.DHCPv4) int {

	var lease *Lease
	mac := req.ClientHWAddr
	if len(mac) != 6 {
		log.Debug("DHCPv4: Invalid ClientHWAddr")
		return -1
	}
	hostname := req.Options.Get(dhcpv4.OptionHostName)
	reqIP := req.Options.Get(dhcpv4.OptionRequestedIPAddress)

	resp.UpdateOption(dhcpv4.OptServerIdentifier(s.conf.dnsIPAddrs[0]))

	switch req.MessageType() {

	case dhcpv4.MessageTypeDiscover:

		s.leasesLock.Lock()
		defer s.leasesLock.Unlock()

		lease = s.findLease(mac)
		if lease == nil {
			toStore := false
			for lease == nil {
				lease = s.reserveLease(mac)
				if lease == nil {
					log.Debug("DHCPv4: No more IP addresses")
					if toStore {
						s.conf.notify(LeaseChangedDBStore)
					}
					return 0
				}

				toStore = true

				if !s.addrAvailable(lease.IP) {
					s.blacklistLease(lease)
					lease = nil
					continue
				}
				break
			}

			s.conf.notify(LeaseChangedDBStore)

			// s.conf.notify(LeaseChangedBlacklisted)

		} else {
			if len(reqIP) != 0 &&
				!bytes.Equal(reqIP, lease.IP) {
				log.Debug("DHCPv4: different RequestedIP: %v != %v", reqIP, lease.IP)
			}
		}

		lease.Hostname = string(hostname)

		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))

	case dhcpv4.MessageTypeRequest:

		sid := req.Options.Get(dhcpv4.OptionServerIdentifier)
		if len(sid) == 0 {
			log.Debug("DHCPv4: No OptionServerIdentifier in Request message for %s", mac)
			return -1
		}
		if !bytes.Equal(sid, s.conf.dnsIPAddrs[0]) {
			log.Debug("DHCPv4: Bad OptionServerIdentifier in Request message for %s", mac)
			return -1
		}

		if len(reqIP) != 4 {
			log.Debug("DHCPv4: Bad OptionRequestedIPAddress in Request message for %s", mac)
			return -1
		}

		s.leasesLock.Lock()
		for _, l := range s.leases {
			if bytes.Equal(l.HWAddr, mac) {
				if !bytes.Equal(l.IP, reqIP) {
					s.leasesLock.Unlock()
					log.Debug("DHCPv4: Mismatched OptionRequestedIPAddress in Request message for %s", mac)
					return -1
				}

				if !bytes.Equal([]byte(l.Hostname), hostname) {
					s.leasesLock.Unlock()
					log.Debug("DHCPv4: Mismatched OptionHostName in Request message for %s", mac)
					return -1
				}

				lease = l
				break
			}
		}
		s.leasesLock.Unlock()

		if lease == nil {
			log.Debug("DHCPv4: No lease for %s", mac)
			return 0
		}

		if lease.Expiry.Unix() != leaseExpireStatic {

			lease.Expiry = time.Now().Add(s.conf.leaseTime)

			s.leasesLock.Lock()
			s.conf.notify(LeaseChangedDBStore)
			s.leasesLock.Unlock()

			s.conf.notify(LeaseChangedAdded)
		}

		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	}

	resp.YourIPAddr = make([]byte, 4)
	copy(resp.YourIPAddr, lease.IP)

	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(s.conf.leaseTime))
	resp.UpdateOption(dhcpv4.OptRouter(s.conf.routerIP))
	resp.UpdateOption(dhcpv4.OptSubnetMask(s.conf.subnetMask))
	resp.UpdateOption(dhcpv4.OptDNS(s.conf.dnsIPAddrs...))
	return 1
}

// client(0.0.0.0:68) -> (Request:ClientMAC,Discover,ClientID,ReqIP,HostName) -> server(255.255.255.255:67)
// client(255.255.255.255:68) <- (Reply:YourIP,ClientMAC,Offer,ServerID,SubnetMask,LeaseTime) <- server(<IP>:67)
// client(0.0.0.0:68) -> (Request:ClientMAC,Request,ClientID,ReqIP,HostName,ServerID,ParamReqList) -> server(255.255.255.255:67)
// client(255.255.255.255:68) <- (Reply:YourIP,ClientMAC,ACK,ServerID,SubnetMask,LeaseTime) <- server(<IP>:67)
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

	r := s.process(req, resp)
	if r < 0 {
		return
	} else if r == 0 {
		resp.Options.Update(dhcpv4.OptMessageType(dhcpv4.MessageTypeNak))
	}

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
			res = append(res, ipnet.IP.To4())
		}
	}
	return res
}

// Start - start server
func (s *V4Server) Start() error {
	if !s.conf.Enabled {
		return nil
	}

	iface, err := net.InterfaceByName(s.conf.InterfaceName)
	if err != nil {
		return fmt.Errorf("DHCPv4: Couldn't find interface by name %s: %s", s.conf.InterfaceName, err)
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

	var err error
	s.conf.routerIP, err = parseIPv4(s.conf.GatewayIP)
	if err != nil {
		return nil, fmt.Errorf("DHCPv4: %s", err)
	}

	subnet, err := parseIPv4(s.conf.SubnetMask)
	if err != nil || !isValidSubnetMask(subnet) {
		return nil, fmt.Errorf("DHCPv4: invalid subnet mask: %s", s.conf.SubnetMask)
	}
	s.conf.subnetMask = make([]byte, 4)
	copy(s.conf.subnetMask, subnet)

	s.conf.ipStart, err = parseIPv4(conf.RangeStart)
	if s.conf.ipStart == nil {
		return nil, fmt.Errorf("DHCPv4: %s", err)
	}
	if s.conf.ipStart[0] == 0 {
		return nil, fmt.Errorf("DHCPv4: invalid range start IP")
	}
	s.conf.ipEnd, err = parseIPv4(conf.RangeEnd)
	if s.conf.ipEnd == nil {
		return nil, fmt.Errorf("DHCPv4: %s", err)
	}
	if !bytes.Equal(s.conf.ipStart[:3], s.conf.ipEnd[:3]) ||
		s.conf.ipStart[3] > s.conf.ipEnd[3] {
		return nil, fmt.Errorf("DHCPv4: range end IP should match range start IP")
	}

	// s.conf.ICMPTimeout = 1000

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 24
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s, nil
}
