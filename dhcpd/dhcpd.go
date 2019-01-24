package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hmage/golibs/log"
	"github.com/krolaw/dhcp4"
)

const defaultDiscoverTime = time.Second * 3

// Lease contains the necessary information about a DHCP lease
// field ordering is important -- yaml fields will mirror ordering from here
type Lease struct {
	HWAddr   net.HardwareAddr `json:"mac" yaml:"hwaddr"`
	IP       net.IP           `json:"ip"`
	Hostname string           `json:"hostname"`
	Expiry   time.Time        `json:"expires"`
}

// ServerConfig - DHCP server configuration
// field ordering is important -- yaml fields will mirror ordering from here
type ServerConfig struct {
	Enabled       bool   `json:"enabled" yaml:"enabled"`
	InterfaceName string `json:"interface_name" yaml:"interface_name"` // eth0, en0 and so on
	GatewayIP     string `json:"gateway_ip" yaml:"gateway_ip"`
	SubnetMask    string `json:"subnet_mask" yaml:"subnet_mask"`
	RangeStart    string `json:"range_start" yaml:"range_start"`
	RangeEnd      string `json:"range_end" yaml:"range_end"`
	LeaseDuration uint   `json:"lease_duration" yaml:"lease_duration"` // in seconds
}

// Server - the current state of the DHCP server
type Server struct {
	conn *filterConn // listening UDP socket

	ipnet *net.IPNet // if interface name changes, this needs to be reset

	// leases
	leases       []*Lease
	leaseStart   net.IP        // parsed from config RangeStart
	leaseStop    net.IP        // parsed from config RangeEnd
	leaseTime    time.Duration // parsed from config LeaseDuration
	leaseOptions dhcp4.Options // parsed from config GatewayIP and SubnetMask

	// IP address pool -- if entry is in the pool, then it's attached to a lease
	IPpool map[[4]byte]net.HardwareAddr

	ServerConfig
	sync.RWMutex
}

// Start will listen on port 67 and serve DHCP requests.
// Even though config can be nil, it is not optional (at least for now), since there are no default values (yet).
func (s *Server) Start(config *ServerConfig) error {
	if config != nil {
		s.ServerConfig = *config
	}

	iface, err := net.InterfaceByName(s.InterfaceName)
	if err != nil {
		s.closeConn() // in case it was already started
		return wrapErrPrint(err, "Couldn't find interface by name %s", s.InterfaceName)
	}

	// get ipv4 address of an interface
	s.ipnet = getIfaceIPv4(iface)
	if s.ipnet == nil {
		s.closeConn() // in case it was already started
		return wrapErrPrint(err, "Couldn't find IPv4 address of interface %s %+v", s.InterfaceName, iface)
	}

	if s.LeaseDuration == 0 {
		s.leaseTime = time.Hour * 2
		s.LeaseDuration = uint(s.leaseTime.Seconds())
	} else {
		s.leaseTime = time.Second * time.Duration(s.LeaseDuration)
	}

	s.leaseStart, err = parseIPv4(s.RangeStart)
	if err != nil {
		s.closeConn() // in case it was already started
		return wrapErrPrint(err, "Failed to parse range start address %s", s.RangeStart)
	}

	s.leaseStop, err = parseIPv4(s.RangeEnd)
	if err != nil {
		s.closeConn() // in case it was already started
		return wrapErrPrint(err, "Failed to parse range end address %s", s.RangeEnd)
	}

	subnet, err := parseIPv4(s.SubnetMask)
	if err != nil {
		s.closeConn() // in case it was already started
		return wrapErrPrint(err, "Failed to parse subnet mask %s", s.SubnetMask)
	}

	// if !bytes.Equal(subnet, s.ipnet.Mask) {
	// 	s.closeConn() // in case it was already started
	// 	return wrapErrPrint(err, "specified subnet mask %s does not meatch interface %s subnet mask %s", s.SubnetMask, s.InterfaceName, s.ipnet.Mask)
	// }

	router, err := parseIPv4(s.GatewayIP)
	if err != nil {
		s.closeConn() // in case it was already started
		return wrapErrPrint(err, "Failed to parse gateway IP %s", s.GatewayIP)
	}

	s.leaseOptions = dhcp4.Options{
		dhcp4.OptionSubnetMask:       subnet,
		dhcp4.OptionRouter:           router,
		dhcp4.OptionDomainNameServer: s.ipnet.IP,
	}

	// TODO: don't close if interface and addresses are the same
	if s.conn != nil {
		s.closeConn()
	}

	c, err := newFilterConn(*iface, ":67") // it has to be bound to 0.0.0.0:67, otherwise it won't see DHCP discover/request packets
	if err != nil {
		return wrapErrPrint(err, "Couldn't start listening socket on 0.0.0.0:67")
	}

	s.conn = c

	go func() {
		// operate on c instead of c.conn because c.conn can change over time
		err := dhcp4.Serve(c, s)
		if err != nil {
			log.Printf("dhcp4.Serve() returned with error: %s", err)
		}
		c.Close() // in case Serve() exits for other reason than listening socket closure
	}()

	return nil
}

// Stop closes the listening UDP socket
func (s *Server) Stop() error {
	if s.conn == nil {
		// nothing to do, return silently
		return nil
	}
	err := s.closeConn()
	if err != nil {
		return wrapErrPrint(err, "Couldn't close UDP listening socket")
	}

	return nil
}

// closeConn will close the connection and set it to zero
func (s *Server) closeConn() error {
	if s.conn == nil {
		return nil
	}
	err := s.conn.Close()
	s.conn = nil
	return err
}

func (s *Server) reserveLease(p dhcp4.Packet) (*Lease, error) {
	// WARNING: do not remove copy()
	// the given hwaddr by p.CHAddr() in the packet survives only during ServeDHCP() call
	// since we need to retain it we need to make our own copy
	hwaddrCOW := p.CHAddr()
	hwaddr := make(net.HardwareAddr, len(hwaddrCOW))
	copy(hwaddr, hwaddrCOW)
	foundLease := s.locateLease(p)
	if foundLease != nil {
		// log.Tracef("found lease for %s: %+v", hwaddr, foundLease)
		return foundLease, nil
	}
	// not assigned a lease, create new one, find IP from LRU
	log.Tracef("Lease not found for %s: creating new one", hwaddr)
	ip, err := s.findFreeIP(p, hwaddr)
	if err != nil {
		return nil, wrapErrPrint(err, "Couldn't find free IP for the lease %s", hwaddr.String())
	}
	log.Tracef("Assigning to %s IP address %s", hwaddr, ip.String())
	hostname := p.ParseOptions()[dhcp4.OptionHostName]
	lease := &Lease{HWAddr: hwaddr, IP: ip, Hostname: string(hostname)}
	s.Lock()
	s.leases = append(s.leases, lease)
	s.Unlock()
	return lease, nil
}

func (s *Server) locateLease(p dhcp4.Packet) *Lease {
	hwaddr := p.CHAddr()
	for i := range s.leases {
		if bytes.Equal([]byte(hwaddr), []byte(s.leases[i].HWAddr)) {
			// log.Tracef("bytes.Equal(%s, %s) returned true", hwaddr, s.leases[i].hwaddr)
			return s.leases[i]
		}
	}
	return nil
}

func (s *Server) findFreeIP(p dhcp4.Packet, hwaddr net.HardwareAddr) (net.IP, error) {
	// if IP pool is nil, lazy initialize it
	if s.IPpool == nil {
		s.IPpool = make(map[[4]byte]net.HardwareAddr)
	}

	// go from start to end, find unreserved IP
	var foundIP net.IP
	for i := 0; i < dhcp4.IPRange(s.leaseStart, s.leaseStop); i++ {
		newIP := dhcp4.IPAdd(s.leaseStart, i)
		foundHWaddr := s.getIPpool(newIP)
		log.Tracef("tried IP %v, got hwaddr %v", newIP, foundHWaddr)
		if foundHWaddr != nil && len(foundHWaddr) != 0 {
			// if !bytes.Equal(foundHWaddr, hwaddr) {
			// 	log.Tracef("SHOULD NOT HAPPEN: hwaddr in IP pool %s is not equal to hwaddr in lease %s", foundHWaddr, hwaddr)
			// }
			log.Tracef("will try again")
			continue
		}
		foundIP = newIP
		break
	}

	if foundIP == nil {
		// TODO: LRU
		return nil, fmt.Errorf("Couldn't find free entry in IP pool")
	}

	s.reserveIP(foundIP, hwaddr)

	return foundIP, nil
}

func (s *Server) getIPpool(ip net.IP) net.HardwareAddr {
	rawIP := []byte(ip)
	IP4 := [4]byte{rawIP[0], rawIP[1], rawIP[2], rawIP[3]}
	return s.IPpool[IP4]
}

func (s *Server) reserveIP(ip net.IP, hwaddr net.HardwareAddr) {
	rawIP := []byte(ip)
	IP4 := [4]byte{rawIP[0], rawIP[1], rawIP[2], rawIP[3]}
	s.IPpool[IP4] = hwaddr
}

func (s *Server) unreserveIP(ip net.IP) {
	rawIP := []byte(ip)
	IP4 := [4]byte{rawIP[0], rawIP[1], rawIP[2], rawIP[3]}
	delete(s.IPpool, IP4)
}

// ServeDHCP handles an incoming DHCP request
func (s *Server) ServeDHCP(p dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) dhcp4.Packet {
	log.Tracef("Got %v message", msgType)
	log.Tracef("Leases:")
	for i, lease := range s.leases {
		log.Tracef("Lease #%d: hwaddr %s, ip %s, expiry %s", i, lease.HWAddr, lease.IP, lease.Expiry)
	}
	log.Tracef("IP pool:")
	for ip, hwaddr := range s.IPpool {
		log.Tracef("IP pool entry %s -> %s", net.IPv4(ip[0], ip[1], ip[2], ip[3]), hwaddr)
	}

	switch msgType {
	case dhcp4.Discover: // Broadcast Packet From Client - Can I have an IP?
		// find a lease, but don't update lease time
		log.Tracef("Got from client: Discover")
		lease, err := s.reserveLease(p)
		if err != nil {
			log.Tracef("Couldn't find free lease: %s", err)
			// couldn't find lease, don't respond
			return nil
		}
		reply := dhcp4.ReplyPacket(p, dhcp4.Offer, s.ipnet.IP, lease.IP, s.leaseTime, s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
		log.Tracef("Replying with offer: offered IP %v for %v with options %+v", lease.IP, s.leaseTime, reply.ParseOptions())
		return reply
	case dhcp4.Request: // Broadcast From Client - I'll take that IP (Also start for renewals)
		// start/renew a lease -- update lease time
		// some clients (OSX) just go right ahead and do Request first from previously known IP, if they get NAK, they restart full cycle with Discover then Request
		return s.handleDHCP4Request(p, msgType, options)
	case dhcp4.Decline: // Broadcast From Client - Sorry I can't use that IP
		log.Tracef("Got from client: Decline")

	case dhcp4.Release: // From Client, I don't need that IP anymore
		log.Tracef("Got from client: Release")

	case dhcp4.Inform: // From Client, I have this IP and there's nothing you can do about it
		log.Tracef("Got from client: Inform")
		// do nothing

	// from server -- ignore those but enumerate just in case
	case dhcp4.Offer: // Broadcast From Server - Here's an IP
		log.Printf("SHOULD NOT HAPPEN -- FROM ANOTHER DHCP SERVER: Offer")
	case dhcp4.ACK: // From Server, Yes you can have that IP
		log.Printf("SHOULD NOT HAPPEN -- FROM ANOTHER DHCP SERVER: ACK")
	case dhcp4.NAK: // From Server, No you cannot have that IP
		log.Printf("SHOULD NOT HAPPEN -- FROM ANOTHER DHCP SERVER: NAK")
	default:
		log.Printf("Unknown DHCP packet detected, ignoring: %v", msgType)
		return nil
	}
	return nil
}

func (s *Server) handleDHCP4Request(p dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) dhcp4.Packet {
	log.Tracef("Got from client: Request")
	if server, ok := options[dhcp4.OptionServerIdentifier]; ok && !net.IP(server).Equal(s.ipnet.IP) {
		log.Tracef("Request message not for this DHCP server (%v vs %v)", server, s.ipnet.IP)
		return nil // Message not for this dhcp server
	}

	reqIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
	if reqIP == nil {
		reqIP = net.IP(p.CIAddr())
	}

	if reqIP.To4() == nil {
		log.Tracef("Replying with NAK: request IP isn't valid IPv4: %s", reqIP)
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	}

	if reqIP.Equal(net.IPv4zero) {
		log.Tracef("Replying with NAK: request IP is 0.0.0.0")
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	}

	log.Tracef("requested IP is %s", reqIP)
	lease, err := s.reserveLease(p)
	if err != nil {
		log.Tracef("Couldn't find free lease: %s", err)
		// couldn't find lease, don't respond
		return nil
	}

	if lease.IP.Equal(reqIP) {
		// IP matches lease IP, nothing else to do
		lease.Expiry = time.Now().Add(s.leaseTime)
		log.Tracef("Replying with ACK: request IP matches lease IP, nothing else to do. IP %v for %v", lease.IP, p.CHAddr())
		return dhcp4.ReplyPacket(p, dhcp4.ACK, s.ipnet.IP, lease.IP, s.leaseTime, s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
	}

	//
	// requested IP different from lease
	//

	log.Tracef("lease IP is different from requested IP: %s vs %s", lease.IP, reqIP)

	hwaddr := s.getIPpool(reqIP)
	if hwaddr == nil {
		// not in pool, check if it's in DHCP range
		if dhcp4.IPInRange(s.leaseStart, s.leaseStop, reqIP) {
			// okay, we can give it to our client -- it's in our DHCP range and not taken, so let them use their IP
			log.Tracef("Replying with ACK: request IP %v is not taken, so assigning lease IP %v to it, for %v", reqIP, lease.IP, p.CHAddr())
			s.unreserveIP(lease.IP)
			lease.IP = reqIP
			s.reserveIP(reqIP, p.CHAddr())
			lease.Expiry = time.Now().Add(s.leaseTime)
			return dhcp4.ReplyPacket(p, dhcp4.ACK, s.ipnet.IP, lease.IP, s.leaseTime, s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
		}
	}

	if hwaddr != nil && !bytes.Equal(hwaddr, lease.HWAddr) {
		log.Printf("SHOULD NOT HAPPEN: IP pool hwaddr does not match lease hwaddr: %s vs %s", hwaddr, lease.HWAddr)
	}

	// requsted IP is not sufficient, reply with NAK
	if hwaddr != nil {
		log.Tracef("Replying with NAK: request IP %s is taken, asked by %v", reqIP, p.CHAddr())
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	}

	// requested IP is outside of DHCP range
	log.Tracef("Replying with NAK: request IP %s is outside of DHCP range [%s, %s], asked by %v", reqIP, s.leaseStart, s.leaseStop, p.CHAddr())
	return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
}

// Leases returns the list of current DHCP leases
func (s *Server) Leases() []*Lease {
	s.RLock()
	result := s.leases
	s.RUnlock()
	return result
}
