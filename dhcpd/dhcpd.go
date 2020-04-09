package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/krolaw/dhcp4"
	ping "github.com/sparrc/go-ping"
)

const defaultDiscoverTime = time.Second * 3
const leaseExpireStatic = 1

var webHandlersRegistered = false

// Lease contains the necessary information about a DHCP lease
// field ordering is important -- yaml fields will mirror ordering from here
type Lease struct {
	HWAddr   net.HardwareAddr `json:"mac" yaml:"hwaddr"`
	IP       net.IP           `json:"ip"`
	Hostname string           `json:"hostname"`

	// Lease expiration time
	// 1: static lease
	Expiry time.Time `json:"expires"`
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
	LeaseDuration uint32 `json:"lease_duration" yaml:"lease_duration"` // in seconds

	// IP conflict detector: time (ms) to wait for ICMP reply.
	// 0: disable
	ICMPTimeout uint32 `json:"icmp_timeout_msec" yaml:"icmp_timeout_msec"`

	WorkDir    string `json:"-" yaml:"-"`
	DBFilePath string `json:"-" yaml:"-"` // path to DB file

	// Called when the configuration is changed by HTTP request
	ConfigModified func() `json:"-" yaml:"-"`

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request)) `json:"-" yaml:"-"`
}

type onLeaseChangedT func(flags int)

// flags for onLeaseChanged()
const (
	LeaseChangedAdded = iota
	LeaseChangedAddedStatic
	LeaseChangedRemovedStatic
	LeaseChangedBlacklisted
)

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
	leasesLock   sync.RWMutex
	leaseStart   net.IP        // parsed from config RangeStart
	leaseStop    net.IP        // parsed from config RangeEnd
	leaseTime    time.Duration // parsed from config LeaseDuration
	leaseOptions dhcp4.Options // parsed from config GatewayIP and SubnetMask

	// IP address pool -- if entry is in the pool, then it's attached to a lease
	IPpool map[[4]byte]net.HardwareAddr

	conf ServerConfig

	// Called when the leases DB is modified
	onLeaseChanged onLeaseChangedT
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

// CheckConfig checks the configuration
func (s *Server) CheckConfig(config ServerConfig) error {
	tmpServer := Server{}
	return tmpServer.setConfig(config)
}

// Create - create object
func Create(config ServerConfig) *Server {
	s := Server{}
	s.conf = config
	s.conf.DBFilePath = filepath.Join(config.WorkDir, dbFilename)
	if s.conf.Enabled {
		err := s.setConfig(config)
		if err != nil {
			log.Error("DHCP: %s", err)
			return nil
		}
	}

	if !webHandlersRegistered && s.conf.HTTPRegister != nil {
		webHandlersRegistered = true
		s.registerHandlers()
	}

	// we can't delay database loading until DHCP server is started,
	//  because we need static leases functionality available beforehand
	s.dbLoad()
	return &s
}

// Init checks the configuration and initializes the server
func (s *Server) Init(config ServerConfig) error {
	err := s.setConfig(config)
	if err != nil {
		return err
	}
	return nil
}

// SetOnLeaseChanged - set callback
func (s *Server) SetOnLeaseChanged(onLeaseChanged onLeaseChangedT) {
	s.onLeaseChanged = onLeaseChanged
}

func (s *Server) notify(flags int) {
	if s.onLeaseChanged == nil {
		return
	}
	s.onLeaseChanged(flags)
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *ServerConfig) {
	*c = s.conf
}

func (s *Server) setConfig(config ServerConfig) error {
	iface, err := net.InterfaceByName(config.InterfaceName)
	if err != nil {
		printInterfaces()
		return wrapErrPrint(err, "Couldn't find interface by name %s", config.InterfaceName)
	}

	// get ipv4 address of an interface
	s.ipnet = getIfaceIPv4(iface)
	if s.ipnet == nil {
		return wrapErrPrint(err, "Couldn't find IPv4 address of interface %s %+v", config.InterfaceName, iface)
	}

	if config.LeaseDuration == 0 {
		s.leaseTime = time.Hour * 2
	} else {
		s.leaseTime = time.Second * time.Duration(config.LeaseDuration)
	}

	s.leaseStart, err = parseIPv4(config.RangeStart)
	if err != nil {
		return wrapErrPrint(err, "Failed to parse range start address %s", config.RangeStart)
	}

	s.leaseStop, err = parseIPv4(config.RangeEnd)
	if err != nil {
		return wrapErrPrint(err, "Failed to parse range end address %s", config.RangeEnd)
	}
	if dhcp4.IPRange(s.leaseStart, s.leaseStop) <= 0 {
		return wrapErrPrint(err, "DHCP: Incorrect range_start/range_end values")
	}

	subnet, err := parseIPv4(config.SubnetMask)
	if err != nil || !isValidSubnetMask(subnet) {
		return wrapErrPrint(err, "Failed to parse subnet mask %s", config.SubnetMask)
	}

	// if !bytes.Equal(subnet, s.ipnet.Mask) {
	// 	return wrapErrPrint(err, "specified subnet mask %s does not meatch interface %s subnet mask %s", s.SubnetMask, s.InterfaceName, s.ipnet.Mask)
	// }

	router, err := parseIPv4(config.GatewayIP)
	if err != nil {
		return wrapErrPrint(err, "Failed to parse gateway IP %s", config.GatewayIP)
	}

	s.leaseOptions = dhcp4.Options{
		dhcp4.OptionSubnetMask:       subnet,
		dhcp4.OptionRouter:           router,
		dhcp4.OptionDomainNameServer: s.ipnet.IP,
	}

	oldconf := s.conf
	s.conf = config
	s.conf.WorkDir = oldconf.WorkDir
	s.conf.HTTPRegister = oldconf.HTTPRegister
	s.conf.ConfigModified = oldconf.ConfigModified
	s.conf.DBFilePath = oldconf.DBFilePath
	return nil
}

// Start will listen on port 67 and serve DHCP requests.
func (s *Server) Start() error {
	// TODO: don't close if interface and addresses are the same
	if s.conn != nil {
		_ = s.closeConn()
	}

	iface, err := net.InterfaceByName(s.conf.InterfaceName)
	if err != nil {
		return wrapErrPrint(err, "Couldn't find interface by name %s", s.conf.InterfaceName)
	}

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
		_ = c.Close() // in case Serve() exits for other reason than listening socket closure
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
	hostname := p.ParseOptions()[dhcp4.OptionHostName]
	lease := &Lease{HWAddr: hwaddr, Hostname: string(hostname)}

	log.Tracef("Lease not found for %s: creating new one", hwaddr)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	ip, err := s.findFreeIP(hwaddr)
	if err != nil {
		i := s.findExpiredLease()
		if i < 0 {
			return nil, wrapErrPrint(err, "Couldn't find free IP for the lease %s", hwaddr.String())
		}

		log.Tracef("Assigning IP address %s to %s (lease for %s expired at %s)",
			s.leases[i].IP, hwaddr, s.leases[i].HWAddr, s.leases[i].Expiry)
		lease.IP = s.leases[i].IP
		s.leases[i] = lease

		s.reserveIP(lease.IP, hwaddr)
		return lease, nil
	}

	log.Tracef("Assigning to %s IP address %s", hwaddr, ip.String())
	lease.IP = ip
	s.leases = append(s.leases, lease)
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

// Find an expired lease and return its index or -1
func (s *Server) findExpiredLease() int {
	now := time.Now().Unix()
	for i, lease := range s.leases {
		if lease.Expiry.Unix() <= now && lease.Expiry.Unix() != leaseExpireStatic {
			return i
		}
	}
	return -1
}

func (s *Server) findFreeIP(hwaddr net.HardwareAddr) (net.IP, error) {
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

// Send ICMP to the specified machine
// Return TRUE if it doesn't reply, which probably means that the IP is available
func (s *Server) addrAvailable(target net.IP) bool {

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

// Add the specified IP to the black list for a time period
func (s *Server) blacklistLease(lease *Lease) {
	hw := make(net.HardwareAddr, 6)
	s.leasesLock.Lock()
	s.reserveIP(lease.IP, hw)
	lease.HWAddr = hw
	lease.Hostname = ""
	lease.Expiry = time.Now().Add(s.leaseTime)
	s.dbStore()
	s.leasesLock.Unlock()
	s.notify(LeaseChangedBlacklisted)
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

		if !s.addrAvailable(lease.IP) {
			s.blacklistLease(lease)
			lease = nil
			continue
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

	if lease.Expiry.Unix() != leaseExpireStatic {
		lease.Expiry = time.Now().Add(s.leaseTime)
		s.leasesLock.Lock()
		s.dbStore()
		s.leasesLock.Unlock()
		s.notify(LeaseChangedAdded) // Note: maybe we shouldn't call this function if only expiration time is updated
	}
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

// AddStaticLease adds a static lease (thread-safe)
func (s *Server) AddStaticLease(l Lease) error {
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
	s.dbStore()
	s.leasesLock.Unlock()
	s.notify(LeaseChangedAddedStatic)
	return nil
}

// Remove a dynamic lease by IP address
func (s *Server) rmDynamicLeaseWithIP(ip net.IP) error {
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
func (s *Server) rmDynamicLeaseWithMAC(mac net.HardwareAddr) error {
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
func (s *Server) rmLease(l Lease) error {
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

// RemoveStaticLease removes a static lease (thread-safe)
func (s *Server) RemoveStaticLease(l Lease) error {
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
	s.dbStore()
	s.leasesLock.Unlock()
	s.notify(LeaseChangedRemovedStatic)
	return nil
}

// flags for Leases() function
const (
	LeasesDynamic = 1
	LeasesStatic  = 2
	LeasesAll     = LeasesDynamic | LeasesStatic
)

// Leases returns the list of current DHCP leases (thread-safe)
func (s *Server) Leases(flags int) []Lease {
	var result []Lease
	now := time.Now().Unix()
	s.leasesLock.RLock()
	for _, lease := range s.leases {
		if ((flags&LeasesDynamic) != 0 && lease.Expiry.Unix() > now) ||
			((flags&LeasesStatic) != 0 && lease.Expiry.Unix() == leaseExpireStatic) {
			result = append(result, *lease)
		}
	}
	s.leasesLock.RUnlock()

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

// FindIPbyMAC finds an IP address by MAC address in the currently active DHCP leases
func (s *Server) FindIPbyMAC(mac net.HardwareAddr) net.IP {
	now := time.Now().Unix()
	s.leasesLock.RLock()
	defer s.leasesLock.RUnlock()
	for _, l := range s.leases {
		if l.Expiry.Unix() > now && bytes.Equal(mac, l.HWAddr) {
			return l.IP
		}
	}
	return nil
}

// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
func (s *Server) FindMACbyIP(ip net.IP) net.HardwareAddr {
	now := time.Now().Unix()

	s.leasesLock.RLock()
	defer s.leasesLock.RUnlock()

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

// Reset internal state
func (s *Server) reset() {
	s.leasesLock.Lock()
	s.leases = nil
	s.IPpool = make(map[[4]byte]net.HardwareAddr)
	s.leasesLock.Unlock()
}
