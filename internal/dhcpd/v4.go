// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/go-ping/ping"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
)

// v4Server - DHCPv4 server
type v4Server struct {
	srv        *server4.Server
	leasesLock sync.Mutex
	leases     []*Lease
	ipAddrs    [256]byte

	conf V4ServerConf
}

// WriteDiskConfig4 - write configuration
func (s *v4Server) WriteDiskConfig4(c *V4ServerConf) {
	*c = s.conf
}

// WriteDiskConfig6 - write configuration
func (s *v4Server) WriteDiskConfig6(c *V6ServerConf) {
}

// Return TRUE if IP address is within range [start..stop]
func ip4InRange(start, stop, ip net.IP) bool {
	if len(start) != 4 || len(stop) != 4 {
		return false
	}
	from := binary.BigEndian.Uint32(start)
	to := binary.BigEndian.Uint32(stop)
	check := binary.BigEndian.Uint32(ip)
	return from <= check && check <= to
}

// ResetLeases - reset leases
func (s *v4Server) ResetLeases(leases []*Lease) {
	s.leases = nil

	for _, l := range leases {

		if l.Expiry.Unix() != leaseExpireStatic &&
			!ip4InRange(s.conf.ipStart, s.conf.ipEnd, l.IP) {

			log.Debug("dhcpv4: skipping a lease with IP %v: not within current IP range", l.IP)
			continue
		}

		s.addLease(l)
	}
}

// GetLeasesRef - get leases
func (s *v4Server) GetLeasesRef() []*Lease {
	return s.leases
}

// Return TRUE if this lease holds a blacklisted IP
func (s *v4Server) blacklisted(l *Lease) bool {
	return l.HWAddr.String() == "00:00:00:00:00:00"
}

// GetLeases returns the list of current DHCP leases (thread-safe)
func (s *v4Server) GetLeases(flags int) []Lease {
	var result []Lease
	now := time.Now().Unix()

	s.leasesLock.Lock()
	for _, lease := range s.leases {
		if ((flags&LeasesDynamic) != 0 && lease.Expiry.Unix() > now && !s.blacklisted(lease)) ||
			((flags&LeasesStatic) != 0 && lease.Expiry.Unix() == leaseExpireStatic) {
			result = append(result, *lease)
		}
	}
	s.leasesLock.Unlock()

	return result
}

// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
func (s *v4Server) FindMACbyIP(ip net.IP) net.HardwareAddr {
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
func (s *v4Server) blacklistLease(lease *Lease) {
	hw := make(net.HardwareAddr, 6)
	lease.HWAddr = hw
	lease.Hostname = ""
	lease.Expiry = time.Now().Add(s.conf.leaseTime)
}

// Remove (swap) lease by index
func (s *v4Server) leaseRemoveSwapByIndex(i int) {
	s.ipAddrs[s.leases[i].IP[3]] = 0
	log.Debug("dhcpv4: removed lease %s", s.leases[i].HWAddr)

	n := len(s.leases)
	if i != n-1 {
		s.leases[i] = s.leases[n-1] // swap with the last element
	}
	s.leases = s.leases[:n-1]
}

// Remove a dynamic lease with the same properties
// Return error if a static lease is found
func (s *v4Server) rmDynamicLease(lease Lease) error {
	for i := 0; i < len(s.leases); i++ {
		l := s.leases[i]

		if bytes.Equal(l.HWAddr, lease.HWAddr) {

			if l.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.leaseRemoveSwapByIndex(i)
			if i == len(s.leases) {
				break
			}
			l = s.leases[i]
		}

		if net.IP.Equal(l.IP, lease.IP) {

			if l.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.leaseRemoveSwapByIndex(i)
		}
	}
	return nil
}

// Add a lease
func (s *v4Server) addLease(l *Lease) {
	s.leases = append(s.leases, l)
	s.ipAddrs[l.IP[3]] = 1
	log.Debug("dhcpv4: added lease %s <-> %s", l.IP, l.HWAddr)
}

// Remove a lease with the same properties
func (s *v4Server) rmLease(lease Lease) error {
	for i, l := range s.leases {
		if net.IP.Equal(l.IP, lease.IP) {

			if !bytes.Equal(l.HWAddr, lease.HWAddr) ||
				l.Hostname != lease.Hostname {
				return fmt.Errorf("lease not found")
			}

			s.leaseRemoveSwapByIndex(i)
			return nil
		}
	}
	return fmt.Errorf("lease not found")
}

// AddStaticLease adds a static lease (thread-safe)
func (s *v4Server) AddStaticLease(lease Lease) error {
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
func (s *v4Server) RemoveStaticLease(l Lease) error {
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
func (s *v4Server) addrAvailable(target net.IP) bool {
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
	log.Debug("dhcpv4: Sending ICMP Echo to %v", target)

	err = pinger.Run()
	if err != nil {
		log.Error("pinger.Run(): %v", err)
		return true
	}

	if reply {
		log.Info("dhcpv4: IP conflict: %v is already used by another device", target)
		return false
	}

	log.Debug("dhcpv4: ICMP procedure is complete: %v", target)
	return true
}

// Find lease by MAC
func (s *v4Server) findLease(mac net.HardwareAddr) *Lease {
	for i := range s.leases {
		if bytes.Equal(mac, s.leases[i].HWAddr) {
			return s.leases[i]
		}
	}
	return nil
}

// Get next free IP
func (s *v4Server) findFreeIP() net.IP {
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
func (s *v4Server) findExpiredLease() int {
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
func (s *v4Server) reserveLease(mac net.HardwareAddr) *Lease {
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

func (s *v4Server) commitLease(l *Lease) {
	l.Expiry = time.Now().Add(s.conf.leaseTime)

	s.leasesLock.Lock()
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()

	s.conf.notify(LeaseChangedAdded)
}

// Process Discover request and return lease
func (s *v4Server) processDiscover(req, resp *dhcpv4.DHCPv4) *Lease {
	mac := req.ClientHWAddr

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	lease := s.findLease(mac)
	if lease == nil {
		toStore := false
		for lease == nil {
			lease = s.reserveLease(mac)
			if lease == nil {
				log.Debug("dhcpv4: No more IP addresses")
				if toStore {
					s.conf.notify(LeaseChangedDBStore)
				}
				return nil
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

	} else {
		reqIP := req.Options.Get(dhcpv4.OptionRequestedIPAddress)
		if len(reqIP) != 0 &&
			!bytes.Equal(reqIP, lease.IP) {
			log.Debug("dhcpv4: different RequestedIP: %v != %v", reqIP, lease.IP)
		}
	}

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	return lease
}

type optFQDN struct {
	name string
}

func (o *optFQDN) String() string {
	return "optFQDN"
}

// flags[1]
// A-RR[1]
// PTR-RR[1]
// name[]
func (o *optFQDN) ToBytes() []byte {
	b := make([]byte, 3+len(o.name))
	i := 0

	b[i] = 0x03 // f_server_overrides | f_server
	i++

	b[i] = 255 // A-RR
	i++

	b[i] = 255 // PTR-RR
	i++

	copy(b[i:], []byte(o.name))
	return b
}

// Process Request request and return lease
// Return false if we don't need to reply
func (s *v4Server) processRequest(req, resp *dhcpv4.DHCPv4) (*Lease, bool) {
	var lease *Lease
	mac := req.ClientHWAddr
	hostname := req.Options.Get(dhcpv4.OptionHostName)
	reqIP := req.Options.Get(dhcpv4.OptionRequestedIPAddress)
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	sid := req.Options.Get(dhcpv4.OptionServerIdentifier)
	if len(sid) != 0 &&
		!bytes.Equal(sid, s.conf.dnsIPAddrs[0]) {
		log.Debug("dhcpv4: Bad OptionServerIdentifier in Request message for %s", mac)
		return nil, false
	}

	if len(reqIP) != 4 {
		log.Debug("dhcpv4: Bad OptionRequestedIPAddress in Request message for %s", mac)
		return nil, false
	}

	s.leasesLock.Lock()
	for _, l := range s.leases {
		if bytes.Equal(l.HWAddr, mac) {
			if !bytes.Equal(l.IP, reqIP) {
				s.leasesLock.Unlock()
				log.Debug("dhcpv4: Mismatched OptionRequestedIPAddress in Request message for %s", mac)
				return nil, true
			}

			lease = l
			break
		}
	}
	s.leasesLock.Unlock()

	if lease == nil {
		log.Debug("dhcpv4: No lease for %s", mac)
		return nil, true
	}

	if lease.Expiry.Unix() != leaseExpireStatic {
		lease.Hostname = string(hostname)
		s.commitLease(lease)
	} else if len(lease.Hostname) != 0 {
		o := &optFQDN{
			name: lease.Hostname,
		}
		fqdn := dhcpv4.Option{
			Code:  dhcpv4.OptionFQDN,
			Value: o,
		}
		resp.UpdateOption(fqdn)
	}

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	return lease, true
}

// Find a lease associated with MAC and prepare response
// Return 1: OK
// Return 0: error; reply with Nak
// Return -1: error; don't reply
func (s *v4Server) process(req, resp *dhcpv4.DHCPv4) int {
	var lease *Lease

	resp.UpdateOption(dhcpv4.OptServerIdentifier(s.conf.dnsIPAddrs[0]))

	switch req.MessageType() {

	case dhcpv4.MessageTypeDiscover:
		lease = s.processDiscover(req, resp)
		if lease == nil {
			return 0
		}

	case dhcpv4.MessageTypeRequest:
		var toReply bool
		lease, toReply = s.processRequest(req, resp)
		if lease == nil {
			if toReply {
				return 0
			}
			return -1 // drop packet
		}
	}

	resp.YourIPAddr = make([]byte, 4)
	copy(resp.YourIPAddr, lease.IP)

	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(s.conf.leaseTime))
	resp.UpdateOption(dhcpv4.OptRouter(s.conf.routerIP))
	resp.UpdateOption(dhcpv4.OptSubnetMask(s.conf.subnetMask))
	resp.UpdateOption(dhcpv4.OptDNS(s.conf.dnsIPAddrs...))

	for _, opt := range s.conf.options {
		resp.Options[opt.code] = opt.val
	}
	return 1
}

// client(0.0.0.0:68) -> (Request:ClientMAC,Type=Discover,ClientID,ReqIP,HostName) -> server(255.255.255.255:67)
// client(255.255.255.255:68) <- (Reply:YourIP,ClientMAC,Type=Offer,ServerID,SubnetMask,LeaseTime) <- server(<IP>:67)
// client(0.0.0.0:68) -> (Request:ClientMAC,Type=Request,ClientID,ReqIP||ClientIP,HostName,ServerID,ParamReqList) -> server(255.255.255.255:67)
// client(255.255.255.255:68) <- (Reply:YourIP,ClientMAC,Type=ACK,ServerID,SubnetMask,LeaseTime) <- server(<IP>:67)
func (s *v4Server) packetHandler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	log.Debug("dhcpv4: received message: %s", req.Summary())

	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover,
		dhcpv4.MessageTypeRequest:
		//

	default:
		log.Debug("dhcpv4: unsupported message type %d", req.MessageType())
		return
	}

	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		log.Debug("dhcpv4: dhcpv4.New: %s", err)
		return
	}

	if len(req.ClientHWAddr) != 6 {
		log.Debug("dhcpv4: Invalid ClientHWAddr")
		return
	}

	r := s.process(req, resp)
	if r < 0 {
		return
	} else if r == 0 {
		resp.Options.Update(dhcpv4.OptMessageType(dhcpv4.MessageTypeNak))
	}

	log.Debug("dhcpv4: sending: %s", resp.Summary())

	_, err = conn.WriteTo(resp.ToBytes(), peer)
	if err != nil {
		log.Error("dhcpv4: conn.Write to %s failed: %s", peer, err)
		return
	}
}

// ifaceIPv4Addrs returns the interface's IPv4 addresses.
func ifaceIPv4Addrs(iface *net.Interface) (ips []net.IP, err error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}

		if ip := ipnet.IP.To4(); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// Start starts the IPv4 DHCP server.
func (s *v4Server) Start() error {
	if !s.conf.Enabled {
		return nil
	}

	ifaceName := s.conf.InterfaceName
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("dhcpv4: finding interface %s by name: %w", ifaceName, err)
	}

	log.Debug("dhcpv4: starting...")

	dnsIPAddrs, err := ifaceIPv4Addrs(iface)
	if err != nil {
		return fmt.Errorf("dhcpv4: getting ipv4 addrs for iface %s: %w", ifaceName, err)
	}

	switch len(dnsIPAddrs) {
	case 0:
		log.Debug("dhcpv4: no ipv4 address for interface %s", iface.Name)

		return nil
	case 1:
		// Some Android devices use 8.8.8.8 if there is no secondary DNS
		// server.  Fix that by setting the secondary DNS address to our
		// address as well.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/1708.
		log.Debug("dhcpv4: setting secondary dns ip to iself for interface %s", iface.Name)
		dnsIPAddrs = append(dnsIPAddrs, dnsIPAddrs[0])
	default:
		// Go on.
	}

	s.conf.dnsIPAddrs = dnsIPAddrs

	laddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: dhcpv4.ServerPort,
	}
	s.srv, err = server4.NewServer(iface.Name, laddr, s.packetHandler, server4.WithDebugLogger())
	if err != nil {
		return err
	}

	log.Info("dhcpv4: listening")

	go func() {
		err = s.srv.Serve()
		log.Debug("dhcpv4: srv.Serve: %s", err)
	}()

	return nil
}

// Stop - stop server
func (s *v4Server) Stop() {
	if s.srv == nil {
		return
	}

	log.Debug("dhcpv4: stopping")
	err := s.srv.Close()
	if err != nil {
		log.Error("dhcpv4: srv.Close: %s", err)
	}
	// now s.srv.Serve() will return
	s.srv = nil
}

// Create DHCPv4 server
func v4Create(conf V4ServerConf) (DHCPServer, error) {
	s := &v4Server{}
	s.conf = conf

	if !conf.Enabled {
		return s, nil
	}

	var err error
	s.conf.routerIP, err = parseIPv4(s.conf.GatewayIP)
	if err != nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}

	subnet, err := parseIPv4(s.conf.SubnetMask)
	if err != nil || !isValidSubnetMask(subnet) {
		return s, fmt.Errorf("dhcpv4: invalid subnet mask: %s", s.conf.SubnetMask)
	}
	s.conf.subnetMask = make([]byte, 4)
	copy(s.conf.subnetMask, subnet)

	s.conf.ipStart, err = parseIPv4(conf.RangeStart)
	if s.conf.ipStart == nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}
	if s.conf.ipStart[0] == 0 {
		return s, fmt.Errorf("dhcpv4: invalid range start IP")
	}

	s.conf.ipEnd, err = parseIPv4(conf.RangeEnd)
	if s.conf.ipEnd == nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}
	if !net.IP.Equal(s.conf.ipStart[:3], s.conf.ipEnd[:3]) ||
		s.conf.ipStart[3] > s.conf.ipEnd[3] {
		return s, fmt.Errorf("dhcpv4: range end IP should match range start IP")
	}

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 24
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	for _, o := range conf.Options {
		code, val := parseOptionString(o)
		if code == 0 {
			log.Debug("dhcpv4: bad option string: %s", o)
			continue
		}

		opt := dhcpOption{
			code: code,
			val:  val,
		}
		s.conf.options = append(s.conf.options, opt)
	}

	return s, nil
}
