package dhcpd

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/krolaw/dhcp4"
)

const defaultDiscoverTime = time.Second * 3

// field ordering is important -- yaml fields will mirror ordering from here
type Lease struct {
	HWAddr net.HardwareAddr `json:"mac" yaml:"hwaddr"`
	IP     net.IP           `json:"ip"`
	Expiry time.Time        `json:"expires"`
}

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
		// trace("found lease for %s: %+v", hwaddr, foundLease)
		return foundLease, nil
	}
	// not assigned a lease, create new one, find IP from LRU
	trace("Lease not found for %s: creating new one", hwaddr)
	ip, err := s.findFreeIP(p, hwaddr)
	if err != nil {
		return nil, wrapErrPrint(err, "Couldn't find free IP for the lease %s", hwaddr.String())
	}
	trace("Assigning to %s IP address %s", hwaddr, ip.String())
	lease := &Lease{HWAddr: hwaddr, IP: ip}
	s.leases = append(s.leases, lease)
	return lease, nil
}

func (s *Server) locateLease(p dhcp4.Packet) *Lease {
	hwaddr := p.CHAddr()
	for i := range s.leases {
		if bytes.Equal([]byte(hwaddr), []byte(s.leases[i].HWAddr)) {
			// trace("bytes.Equal(%s, %s) returned true", hwaddr, s.leases[i].hwaddr)
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
		trace("tried IP %v, got hwaddr %v", newIP, foundHWaddr)
		if foundHWaddr != nil && len(foundHWaddr) != 0 {
			// if !bytes.Equal(foundHWaddr, hwaddr) {
			// 	trace("SHOULD NOT HAPPEN: hwaddr in IP pool %s is not equal to hwaddr in lease %s", foundHWaddr, hwaddr)
			// }
			trace("will try again")
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

func (s *Server) ServeDHCP(p dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) dhcp4.Packet {
	trace("Got %v message", msgType)
	trace("Leases:")
	for i, lease := range s.leases {
		trace("Lease #%d: hwaddr %s, ip %s, expiry %s", i, lease.HWAddr, lease.IP, lease.Expiry)
	}
	trace("IP pool:")
	for ip, hwaddr := range s.IPpool {
		trace("IP pool entry %s -> %s", net.IPv4(ip[0], ip[1], ip[2], ip[3]), hwaddr)
	}
	// spew.Dump(s.leases, s.IPpool)
	// log.Printf("Called with msgType = %v, options = %+v", msgType, options)
	// spew.Dump(p)
	// log.Printf("%14s %v", "p.Broadcast", p.Broadcast())       // false
	// log.Printf("%14s %v", "p.CHAddr", p.CHAddr())             // 2c:f0:a2:f2:31:00
	// log.Printf("%14s %v", "p.CIAddr", p.CIAddr())             // 0.0.0.0
	// log.Printf("%14s %v", "p.Cookie", p.Cookie())             // [99 130 83 99]
	// log.Printf("%14s %v", "p.File", p.File())                 // []
	// log.Printf("%14s %v", "p.Flags", p.Flags())               // [0 0]
	// log.Printf("%14s %v", "p.GIAddr", p.GIAddr())             // 0.0.0.0
	// log.Printf("%14s %v", "p.HLen", p.HLen())                 // 6
	// log.Printf("%14s %v", "p.HType", p.HType())               // 1
	// log.Printf("%14s %v", "p.Hops", p.Hops())                 // 0
	// log.Printf("%14s %v", "p.OpCode", p.OpCode())             // BootRequest
	// log.Printf("%14s %v", "p.Options", p.Options())           // [53 1 1 55 10 1 121 3 6 15 119 252 95 44 46 57 2 5 220 61 7 1 44 240 162 242 49 0 51 4 0 118 167 0 12 4 119 104 109 100 255 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
	// log.Printf("%14s %v", "p.ParseOptions", p.ParseOptions()) // map[OptionParameterRequestList:[1 121 3 6 15 119 252 95 44 46] OptionDHCPMessageType:[1] OptionMaximumDHCPMessageSize:[5 220] OptionClientIdentifier:[1 44 240 162 242 49 0] OptionIPAddressLeaseTime:[0 118 167 0] OptionHostName:[119 104 109 100]]
	// log.Printf("%14s %v", "p.SIAddr", p.SIAddr())             // 0.0.0.0
	// log.Printf("%14s %v", "p.SName", p.SName())               // []
	// log.Printf("%14s %v", "p.Secs", p.Secs())                 // [0 8]
	// log.Printf("%14s %v", "p.XId", p.XId())                   // [211 184 20 44]
	// log.Printf("%14s %v", "p.YIAddr", p.YIAddr())             // 0.0.0.0

	switch msgType {
	case dhcp4.Discover: // Broadcast Packet From Client - Can I have an IP?
		// find a lease, but don't update lease time
		trace("Got from client: Discover")
		lease, err := s.reserveLease(p)
		if err != nil {
			trace("Couldn't find free lease: %s", err)
			// couldn't find lease, don't respond
			return nil
		}
		reply := dhcp4.ReplyPacket(p, dhcp4.Offer, s.ipnet.IP, lease.IP, s.leaseTime, s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
		trace("Replying with offer: offered IP %v for %v with options %+v", lease.IP, s.leaseTime, reply.ParseOptions())
		return reply
	case dhcp4.Request: // Broadcast From Client - I'll take that IP (Also start for renewals)
		// start/renew a lease -- update lease time
		// some clients (OSX) just go right ahead and do Request first from previously known IP, if they get NAK, they restart full cycle with Discover then Request
		trace("Got from client: Request")
		if server, ok := options[dhcp4.OptionServerIdentifier]; ok && !net.IP(server).Equal(s.ipnet.IP) {
			trace("Request message not for this DHCP server (%v vs %v)", p, server, s.ipnet.IP)
			return nil // Message not for this dhcp server
		}

		reqIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
		if reqIP == nil {
			reqIP = net.IP(p.CIAddr())
		}

		if reqIP.To4() == nil {
			trace("Replying with NAK: request IP isn't valid IPv4: %s", reqIP)
			return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
		}

		if reqIP.Equal(net.IPv4zero) {
			trace("Replying with NAK: request IP is 0.0.0.0")
			return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
		}

		trace("requested IP is %s", reqIP)
		lease, err := s.reserveLease(p)
		if err != nil {
			trace("Couldn't find free lease: %s", err)
			// couldn't find lease, don't respond
			return nil
		}

		if lease.IP.Equal(reqIP) {
			// IP matches lease IP, nothing else to do
			lease.Expiry = time.Now().Add(s.leaseTime)
			trace("Replying with ACK: request IP matches lease IP, nothing else to do. IP %v for %v", lease.IP, p.CHAddr())
			return dhcp4.ReplyPacket(p, dhcp4.ACK, s.ipnet.IP, lease.IP, s.leaseTime, s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
		}

		//
		// requested IP different from lease
		//

		trace("lease IP is different from requested IP: %s vs %s", lease.IP, reqIP)

		hwaddr := s.getIPpool(reqIP)
		if hwaddr == nil {
			// not in pool, check if it's in DHCP range
			if dhcp4.IPInRange(s.leaseStart, s.leaseStop, reqIP) {
				// okay, we can give it to our client -- it's in our DHCP range and not taken, so let them use their IP
				trace("Replying with ACK: request IP %v is not taken, so assigning lease IP %v to it, for %v", reqIP, lease.IP, p.CHAddr())
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
			trace("Replying with NAK: request IP %s is taken, asked by %v", reqIP, p.CHAddr())
			return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
		}

		// requested IP is outside of DHCP range
		trace("Replying with NAK: request IP %s is outside of DHCP range [%s, %s], asked by %v", reqIP, s.leaseStart, s.leaseStop, p.CHAddr())
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	case dhcp4.Decline: // Broadcast From Client - Sorry I can't use that IP
		trace("Got from client: Decline")

	case dhcp4.Release: // From Client, I don't need that IP anymore
		trace("Got from client: Release")

	case dhcp4.Inform: // From Client, I have this IP and there's nothing you can do about it
		trace("Got from client: Inform")
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

func (s *Server) Leases() []*Lease {
	return s.leases
}
