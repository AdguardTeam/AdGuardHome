// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/go-ping/ping"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/willf/bitset"
)

// v4Server is a DHCPv4 server.
//
// TODO(a.garipov): Think about unifying this and v6Server.
type v4Server struct {
	conf V4ServerConf
	srv  *server4.Server

	// leasedOffsets contains offsets from conf.ipRange.start that have been
	// leased.
	leasedOffsets *bitset.BitSet

	// leases contains all dynamic and static leases.
	leases []*Lease

	// leasesLock protects leases and leasedOffsets.
	leasesLock sync.Mutex
}

// WriteDiskConfig4 - write configuration
func (s *v4Server) WriteDiskConfig4(c *V4ServerConf) {
	*c = s.conf
}

// WriteDiskConfig6 - write configuration
func (s *v4Server) WriteDiskConfig6(c *V6ServerConf) {
}

// ResetLeases - reset leases
func (s *v4Server) ResetLeases(leases []*Lease) {
	s.leases = nil

	for _, l := range leases {
		if l.Expiry.Unix() != leaseExpireStatic && !s.conf.ipRange.contains(l.IP) {
			log.Debug("dhcpv4: skipping a lease with ip %v: not within current ip range", l.IP)

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
	// The function shouldn't return nil value because zero-length slice
	// behaves differently in cases like marshalling.  Our front-end also
	// requires non-nil value in the response.
	result := []Lease{}
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

// rmLeaseByIndex removes a lease by its index in the leases slice.
func (s *v4Server) rmLeaseByIndex(i int) {
	l := s.leases[i]
	s.leases = append(s.leases[:i], s.leases[i+1:]...)

	r := s.conf.ipRange
	offset, ok := r.offset(l.IP)
	if ok {
		s.leasedOffsets.Clear(offset)
	}

	log.Debug("dhcpv4: removed lease %s (%s)", l.IP, l.HWAddr)
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

			s.rmLeaseByIndex(i)
			if i == len(s.leases) {
				break
			}

			l = s.leases[i]
		}

		if net.IP.Equal(l.IP, lease.IP) {
			if l.Expiry.Unix() == leaseExpireStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.rmLeaseByIndex(i)
		}
	}
	return nil
}

// addLease adds a lease.
func (s *v4Server) addLease(l *Lease) {
	r := s.conf.ipRange
	offset, ok := r.offset(l.IP)
	if !ok {
		// TODO(a.garipov): Better error handling.
		log.Debug("dhcpv4: lease %s (%s) out of range, not adding", l.IP, l.HWAddr)

		return
	}

	s.leases = append(s.leases, l)
	s.leasedOffsets.Set(uint(offset))

	log.Debug("dhcpv4: added lease %s (%s)", l.IP, l.HWAddr)
}

// Remove a lease with the same properties
func (s *v4Server) rmLease(lease Lease) error {
	for i, l := range s.leases {
		if l.IP.Equal(lease.IP) {
			if !bytes.Equal(l.HWAddr, lease.HWAddr) ||
				l.Hostname != lease.Hostname {
				return fmt.Errorf("lease not found")
			}

			s.rmLeaseByIndex(i)

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
	pinger.OnRecv = func(_ *ping.Packet) {
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

// findLease finds a lease by its MAC-address.
func (s *v4Server) findLease(mac net.HardwareAddr) (l *Lease) {
	for _, l = range s.leases {
		if bytes.Equal(mac, l.HWAddr) {
			return l
		}
	}

	return nil
}

// nextIP generates a new free IP.
func (s *v4Server) nextIP() (ip net.IP) {
	r := s.conf.ipRange
	ip = r.find(func(next net.IP) (ok bool) {
		offset, ok := r.offset(next)
		if !ok {
			// Shouldn't happen.
			return false
		}

		return !s.leasedOffsets.Test(uint(offset))
	})

	return ip.To4()
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

// reserveLease reserves a lease for a client by its MAC-address.  It returns
// nil if it couldn't allocate a new lease.
func (s *v4Server) reserveLease(mac net.HardwareAddr) (l *Lease) {
	l = &Lease{
		HWAddr: make([]byte, 6),
	}

	copy(l.HWAddr, mac)

	l.IP = s.nextIP()
	if l.IP == nil {
		i := s.findExpiredLease()
		if i < 0 {
			return nil
		}

		copy(s.leases[i].HWAddr, mac)

		return s.leases[i]
	}

	s.addLease(l)

	return l
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
		reqIP := req.RequestedIPAddress()
		if len(reqIP) != 0 && !reqIP.Equal(lease.IP) {
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
	reqIP := req.RequestedIPAddress()
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	sid := req.ServerIdentifier()
	if len(sid) != 0 && !sid.Equal(s.conf.dnsIPAddrs[0]) {
		log.Debug("dhcpv4: bad OptionServerIdentifier in request message for %s", mac)

		return nil, false
	}

	if ip4 := reqIP.To4(); ip4 == nil {
		log.Debug("dhcpv4: bad OptionRequestedIPAddress in request message for %s", mac)

		return nil, false
	}

	s.leasesLock.Lock()
	for _, l := range s.leases {
		if bytes.Equal(l.HWAddr, mac) {
			if !l.IP.Equal(reqIP) {
				s.leasesLock.Unlock()
				log.Debug("dhcpv4: mismatched OptionRequestedIPAddress in request message for %s", mac)

				return nil, true
			}

			lease = l

			break
		}
	}
	s.leasesLock.Unlock()

	if lease == nil {
		log.Debug("dhcpv4: no lease for %s", mac)

		return nil, true
	}

	if lease.Expiry.Unix() != leaseExpireStatic {
		lease.Hostname = req.HostName()
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
		resp.Options[opt.code] = opt.data
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

	dnsIPAddrs, err := ifaceDNSIPAddrs(iface, ipVersion4, defaultMaxAttempts, defaultBackoff)
	if err != nil {
		return fmt.Errorf("dhcpv4: interface %s: %w", ifaceName, err)
	}

	if len(dnsIPAddrs) == 0 {
		// No available IP addresses which may appear later.
		return nil
	}

	s.conf.dnsIPAddrs = dnsIPAddrs

	laddr := &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
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
func v4Create(conf V4ServerConf) (srv DHCPServer, err error) {
	s := &v4Server{}
	s.conf = conf

	if !conf.Enabled {
		return s, nil
	}

	s.conf.routerIP, err = tryTo4(s.conf.GatewayIP)
	if err != nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}

	if s.conf.SubnetMask == nil {
		return s, fmt.Errorf("dhcpv4: invalid subnet mask: %v", s.conf.SubnetMask)
	}
	s.conf.subnetMask = make([]byte, 4)
	copy(s.conf.subnetMask, s.conf.SubnetMask.To4())

	s.conf.ipRange, err = newIPRange(conf.RangeStart, conf.RangeEnd)
	if err != nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}

	s.leasedOffsets = &bitset.BitSet{}

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 24
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	p := newDHCPOptionParser()

	for i, o := range conf.Options {
		var code uint8
		var data []byte
		code, data, err = p.parse(o)
		if err != nil {
			log.Error("dhcpv4: bad option string at index %d: %s", i, err)

			continue
		}

		opt := dhcpOption{
			code: code,
			data: data,
		}

		s.conf.options = append(s.conf.options, opt)
	}

	return s, nil
}
