package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
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

	cond     *sync.Cond // Synchronize worker thread with main thread
	mutex    sync.Mutex // Mutex for 'cond'
	running  bool       // Set if the worker thread is running
	stopping bool       // Set if the worker thread should be stopped

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

// Print information about the available network interfaces
func printInterfaces() {
	ifaces, _ := net.Interfaces()
	var buf strings.Builder
	for i := range ifaces {
		buf.WriteString(fmt.Sprintf("\"%s\", ", ifaces[i].Name))
	}
	log.Info("Available network interfaces: %s", buf.String())
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
		printInterfaces()
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

	s.dbLoad()

	c, err := newFilterConn(*iface, ":67") // it has to be bound to 0.0.0.0:67, otherwise it won't see DHCP discover/request packets
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
		c.Close() // in case Serve() exits for other reason than listening socket closure
		s.running = false
		s.cond.Signal()
	}()

	return nil
}

// Stop closes the listening UDP socket
func (s *Server) Stop() error {
	if s.conn == nil {
		// nothing to do, return silently
		return nil
	}

	s.stopping = true

	err := s.closeConn()
	if err != nil {
		return wrapErrPrint(err, "Couldn't close UDP listening socket")
	}

	// We've just closed the listening socket.
	// Worker thread should exit right after it tries to read from the socket.
	s.mutex.Lock()
	for s.running {
		s.cond.Wait()
	}
	s.mutex.Unlock()

	s.dbStore()
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

// Reserve a lease for the client
func (s *Server) reserveLease(p dhcp4.Packet) (*Lease, error) {
	// WARNING: do not remove copy()
	// the given hwaddr by p.CHAddr() in the packet survives only during ServeDHCP() call
	// since we need to retain it we need to make our own copy
	hwaddrCOW := p.CHAddr()
	hwaddr := make(net.HardwareAddr, len(hwaddrCOW))
	copy(hwaddr, hwaddrCOW)
	// not assigned a lease, create new one, find IP from LRU
	log.Tracef("Lease not found for %s: creating new one", hwaddr)
	ip, err := s.findFreeIP(hwaddr)
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

// Find a lease for the client
func (s *Server) findLease(p dhcp4.Packet) *Lease {
	hwaddr := p.CHAddr()
	for i := range s.leases {
		if bytes.Equal([]byte(hwaddr), []byte(s.leases[i].HWAddr)) {
			// log.Tracef("bytes.Equal(%s, %s) returned true", hwaddr, s.leases[i].hwaddr)
			return s.leases[i]
		}
	}
	return nil
}

func (s *Server) findFreeIP(hwaddr net.HardwareAddr) (net.IP, error) {
	// if IP pool is nil, lazy initialize it
	if s.IPpool == nil {
		s.IPpool = make(map[[4]byte]net.HardwareAddr)
	}

	// go from start to end, find unreserved IP
	var foundIP net.IP
	for i := 0; i < dhcp4.IPRange(s.leaseStart, s.leaseStop); i++ {
		newIP := dhcp4.IPAdd(s.leaseStart, i)
		foundHWaddr := s.findReservedHWaddr(newIP)
		log.Tracef("tried IP %v, got hwaddr %v", newIP, foundHWaddr)
		if foundHWaddr != nil && len(foundHWaddr) != 0 {
			// if !bytes.Equal(foundHWaddr, hwaddr) {
			// 	log.Tracef("SHOULD NOT HAPPEN: hwaddr in IP pool %s is not equal to hwaddr in lease %s", foundHWaddr, hwaddr)
			// }
			continue
		}
		foundIP = newIP
		break
	}

	if foundIP == nil {
		// TODO: LRU
		return nil, fmt.Errorf("couldn't find free entry in IP pool")
	}

	s.reserveIP(foundIP, hwaddr)

	return foundIP, nil
}

func (s *Server) findReservedHWaddr(ip net.IP) net.HardwareAddr {
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
	s.printLeases()

	switch msgType {
	case dhcp4.Discover: // Broadcast Packet From Client - Can I have an IP?
		return s.handleDiscover(p, options)

	case dhcp4.Request: // Broadcast From Client - I'll take that IP (Also start for renewals)
		// start/renew a lease -- update lease time
		// some clients (OSX) just go right ahead and do Request first from previously known IP, if they get NAK, they restart full cycle with Discover then Request
		return s.handleDHCP4Request(p, options)

	case dhcp4.Decline: // Broadcast From Client - Sorry I can't use that IP
		return s.handleDecline(p, options)

	case dhcp4.Release: // From Client, I don't need that IP anymore
		return s.handleRelease(p, options)

	case dhcp4.Inform: // From Client, I have this IP and there's nothing you can do about it
		return s.handleInform(p, options)

	// from server -- ignore those but enumerate just in case
	case dhcp4.Offer: // Broadcast From Server - Here's an IP
		log.Printf("DHCP: received message from %s: Offer", p.CHAddr())

	case dhcp4.ACK: // From Server, Yes you can have that IP
		log.Printf("DHCP: received message from %s: ACK", p.CHAddr())

	case dhcp4.NAK: // From Server, No you cannot have that IP
		log.Printf("DHCP: received message from %s: NAK", p.CHAddr())

	default:
		log.Printf("DHCP: unknown packet %v from %s", msgType, p.CHAddr())
		return nil
	}
	return nil
}

// Return TRUE if DHCP packet is correct
func isValidPacket(p dhcp4.Packet) bool {
	hw := p.CHAddr()
	zeroes := make([]byte, len(hw))
	if bytes.Equal(hw, zeroes) {
		log.Tracef("Packet has empty CHAddr")
		return false
	}
	return true
}

func (s *Server) handleDiscover(p dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	// find a lease, but don't update lease time
	var lease *Lease
	var err error

	reqIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
	hostname := p.ParseOptions()[dhcp4.OptionHostName]
	log.Tracef("Message from client: Discover.  ReqIP: %s  HW: %s  Hostname: %s",
		reqIP, p.CHAddr(), hostname)

	if !isValidPacket(p) {
		return nil
	}

	lease = s.findLease(p)
	for lease == nil {
		lease, err = s.reserveLease(p)
		if err != nil {
			log.Error("Couldn't find free lease: %s", err)
			return nil
		}

		break
	}

	opt := s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList])
	reply := dhcp4.ReplyPacket(p, dhcp4.Offer, s.ipnet.IP, lease.IP, s.leaseTime, opt)
	log.Tracef("Replying with offer: offered IP %v for %v with options %+v", lease.IP, s.leaseTime, reply.ParseOptions())
	return reply
}

func (s *Server) handleDHCP4Request(p dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	var lease *Lease

	reqIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
	log.Tracef("Message from client: Request.  IP: %s  ReqIP: %s  HW: %s",
		p.CIAddr(), reqIP, p.CHAddr())

	if !isValidPacket(p) {
		return nil
	}

	server := options[dhcp4.OptionServerIdentifier]
	if server != nil && !net.IP(server).Equal(s.ipnet.IP) {
		log.Tracef("Request message not for this DHCP server (%v vs %v)", server, s.ipnet.IP)
		return nil // Message not for this dhcp server
	}

	if reqIP == nil {
		reqIP = p.CIAddr()

	} else if reqIP == nil || reqIP.To4() == nil {
		log.Tracef("Requested IP isn't a valid IPv4: %s", reqIP)
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	}

	lease = s.findLease(p)
	if lease == nil {
		log.Tracef("Lease for %s isn't found", p.CHAddr())
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	}

	if !lease.IP.Equal(reqIP) {
		log.Tracef("Lease for %s doesn't match requested/client IP: %s vs %s",
			lease.HWAddr, lease.IP, reqIP)
		return dhcp4.ReplyPacket(p, dhcp4.NAK, s.ipnet.IP, nil, 0, nil)
	}

	lease.Expiry = time.Now().Add(s.leaseTime)
	log.Tracef("Replying with ACK.  IP: %s  HW: %s  Expire: %s",
		lease.IP, lease.HWAddr, lease.Expiry)
	opt := s.leaseOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList])
	return dhcp4.ReplyPacket(p, dhcp4.ACK, s.ipnet.IP, lease.IP, s.leaseTime, opt)
}

func (s *Server) handleInform(p dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	log.Tracef("Message from client: Inform.  IP: %s  HW: %s",
		p.CIAddr(), p.CHAddr())

	return nil
}

func (s *Server) handleRelease(p dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	log.Tracef("Message from client: Release.  IP: %s  HW: %s",
		p.CIAddr(), p.CHAddr())

	return nil
}

func (s *Server) handleDecline(p dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	reqIP := net.IP(options[dhcp4.OptionRequestedIPAddress])
	log.Tracef("Message from client: Decline.  IP: %s  HW: %s",
		reqIP, p.CHAddr())

	return nil
}

// Leases returns the list of current DHCP leases (thread-safe)
func (s *Server) Leases() []*Lease {
	s.RLock()
	result := s.leases
	s.RUnlock()
	return result
}

// Print information about the current leases
func (s *Server) printLeases() {
	log.Tracef("Leases:")
	for i, lease := range s.leases {
		log.Tracef("Lease #%d: hwaddr %s, ip %s, expiry %s",
			i, lease.HWAddr, lease.IP, lease.Expiry)
	}
}

// Reset internal state
func (s *Server) reset() {
	s.Lock()
	s.leases = nil
	s.Unlock()
	s.IPpool = make(map[[4]byte]net.HardwareAddr)
}
