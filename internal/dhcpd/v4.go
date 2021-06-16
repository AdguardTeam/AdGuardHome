//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/go-ping/ping"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
)

// v4Server is a DHCPv4 server.
//
// TODO(a.garipov): Think about unifying this and v6Server.
type v4Server struct {
	conf V4ServerConf
	srv  *server4.Server

	// leasedOffsets contains offsets from conf.ipRange.start that have been
	// leased.
	leasedOffsets *bitSet

	// leaseHosts is the set of all hostnames of all known DHCP clients.
	leaseHosts *aghstrings.Set

	// leases contains all dynamic and static leases.
	leases []*Lease

	// leasesLock protects leases, leaseHosts, and leasedOffsets.
	leasesLock sync.Mutex
}

// WriteDiskConfig4 - write configuration
func (s *v4Server) WriteDiskConfig4(c *V4ServerConf) {
	*c = s.conf
}

// WriteDiskConfig6 - write configuration
func (s *v4Server) WriteDiskConfig6(c *V6ServerConf) {
}

// normalizeHostname normalizes a hostname sent by the client.  If err is not
// nil, norm is an empty string.
func normalizeHostname(hostname string) (norm string, err error) {
	defer func() { err = errors.Annotate(err, "normalizing %q: %w", hostname) }()

	if hostname == "" {
		return "", nil
	}

	norm = strings.ToLower(hostname)
	parts := strings.FieldsFunc(norm, func(c rune) (ok bool) {
		return c != '.' && !aghnet.IsValidHostOuterRune(c)
	})

	if len(parts) == 0 {
		return "", fmt.Errorf("no valid parts")
	}

	norm = strings.Join(parts, "-")
	norm = strings.TrimSuffix(norm, "-")

	return norm, nil
}

// validHostnameForClient accepts the hostname sent by the client and its IP and
// returns either a normalized version of that hostname, or a new hostname
// generated from the IP address, or an empty string.
func (s *v4Server) validHostnameForClient(cliHostname string, ip net.IP) (hostname string) {
	hostname, err := normalizeHostname(cliHostname)
	if err != nil {
		log.Info("dhcpv4: %s", err)
	}

	if hostname == "" {
		hostname = aghnet.GenerateHostname(ip)
	}

	err = aghnet.ValidateDomainName(hostname)
	if err != nil {
		log.Info("dhcpv4: %s", err)
		hostname = ""
	}

	return hostname
}

// ResetLeases resets leases.
func (s *v4Server) ResetLeases(leases []*Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if !s.conf.Enabled {
		return
	}

	s.leasedOffsets = newBitSet()
	s.leaseHosts = aghstrings.NewSet()
	s.leases = nil

	for _, l := range leases {
		if !l.IsStatic() {
			l.Hostname = s.validHostnameForClient(l.Hostname, l.IP)
		}
		err = s.addLease(l)
		if err != nil {
			// TODO(a.garipov): Wrap and bubble up the error.
			log.Error(
				"dhcpv4: reset: re-adding a lease for %s (%s): %s",
				l.IP,
				l.HWAddr,
				err,
			)

			continue
		}
	}

	return nil
}

// getLeasesRef returns the actual leases slice.  For internal use only.
func (s *v4Server) getLeasesRef() []*Lease {
	return s.leases
}

// isBlocklisted returns true if this lease holds a blocklisted IP.
//
// TODO(a.garipov): Make a method of *Lease?
func (s *v4Server) isBlocklisted(l *Lease) (ok bool) {
	if len(l.HWAddr) == 0 {
		return false
	}

	ok = true
	for _, b := range l.HWAddr {
		if b != 0 {
			ok = false

			break
		}
	}

	return ok
}

// GetLeases returns the list of current DHCP leases.  It is safe for concurrent
// use.
func (s *v4Server) GetLeases(flags GetLeasesFlags) (leases []*Lease) {
	// The function shouldn't return nil, because zero-length slice behaves
	// differently in cases like marshalling.  Our front-end also requires
	// a non-nil value in the response.
	leases = []*Lease{}

	getDynamic := flags&LeasesDynamic != 0
	getStatic := flags&LeasesStatic != 0

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	now := time.Now()
	for _, l := range s.leases {
		if getDynamic && l.Expiry.After(now) && !s.isBlocklisted(l) {
			leases = append(leases, l.Clone())

			continue
		}

		if getStatic && l.IsStatic() {
			leases = append(leases, l.Clone())
		}
	}

	return leases
}

// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
func (s *v4Server) FindMACbyIP(ip net.IP) net.HardwareAddr {
	now := time.Now()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}

	for _, l := range s.leases {
		if l.IP.Equal(ip4) {
			if l.Expiry.After(now) || l.IsStatic() {
				return l.HWAddr
			}
		}
	}

	return nil
}

// defaultHwAddrLen is the default length of a hardware (MAC) address.
const defaultHwAddrLen = 6

// Add the specified IP to the black list for a time period
func (s *v4Server) blocklistLease(l *Lease) {
	l.HWAddr = make(net.HardwareAddr, defaultHwAddrLen)
	l.Hostname = ""
	l.Expiry = time.Now().Add(s.conf.leaseTime)
}

// rmLeaseByIndex removes a lease by its index in the leases slice.
func (s *v4Server) rmLeaseByIndex(i int) {
	n := len(s.leases)
	if i >= n {
		// TODO(a.garipov): Better error handling.
		log.Debug("dhcpv4: can't remove lease at index %d: no such lease", i)

		return
	}

	l := s.leases[i]
	s.leases = append(s.leases[:i], s.leases[i+1:]...)

	r := s.conf.ipRange
	offset, ok := r.offset(l.IP)
	if ok {
		s.leasedOffsets.set(offset, false)
	}

	s.leaseHosts.Del(l.Hostname)

	log.Debug("dhcpv4: removed lease %s (%s)", l.IP, l.HWAddr)
}

// Remove a dynamic lease with the same properties
// Return error if a static lease is found
func (s *v4Server) rmDynamicLease(lease *Lease) (err error) {
	for i := 0; i < len(s.leases); i++ {
		l := s.leases[i]

		if bytes.Equal(l.HWAddr, lease.HWAddr) {
			if l.IsStatic() {
				return errors.Error("static lease already exists")
			}

			s.rmLeaseByIndex(i)
			if i == len(s.leases) {
				break
			}

			l = s.leases[i]
		}

		if l.IP.Equal(lease.IP) {
			if l.IsStatic() {
				return errors.Error("static lease already exists")
			}

			s.rmLeaseByIndex(i)
			if i == len(s.leases) {
				break
			}

			l = s.leases[i]
		}

		if l.Hostname == lease.Hostname {
			l.Hostname = ""
		}
	}

	return nil
}

// addLease adds a dynamic or static lease.
func (s *v4Server) addLease(l *Lease) (err error) {
	r := s.conf.ipRange
	offset, inOffset := r.offset(l.IP)

	if l.IsStatic() {
		if sn := s.conf.subnet; !sn.Contains(l.IP) {
			return fmt.Errorf("subnet %s does not contain the ip %q", sn, l.IP)
		}
	} else if !inOffset {
		return fmt.Errorf("lease %s (%s) out of range, not adding", l.IP, l.HWAddr)
	}

	s.leases = append(s.leases, l)
	s.leasedOffsets.set(offset, true)

	if l.Hostname != "" {
		s.leaseHosts.Add(l.Hostname)
	}

	return nil
}

// rmLease removes a lease with the same properties.
func (s *v4Server) rmLease(lease *Lease) (err error) {
	if len(s.leases) == 0 {
		return nil
	}

	for i, l := range s.leases {
		if l.IP.Equal(lease.IP) {
			if !bytes.Equal(l.HWAddr, lease.HWAddr) || l.Hostname != lease.Hostname {
				return fmt.Errorf("lease for ip %s is different: %+v", lease.IP, l)
			}

			s.rmLeaseByIndex(i)

			return nil
		}
	}

	return errors.Error("lease not found")
}

// AddStaticLease adds a static lease.  It is safe for concurrent use.
func (s *v4Server) AddStaticLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: adding static lease: %w") }()

	if ip4 := l.IP.To4(); ip4 == nil {
		return fmt.Errorf("invalid ip %q, only ipv4 is supported", l.IP)
	}

	l.Expiry = time.Unix(leaseExpireStatic, 0)

	err = aghnet.ValidateHardwareAddress(l.HWAddr)
	if err != nil {
		return err
	}

	if hostname := l.Hostname; hostname != "" {
		hostname, err = normalizeHostname(hostname)
		if err != nil {
			return err
		}

		err = aghnet.ValidateDomainName(hostname)
		if err != nil {
			return fmt.Errorf("validating hostname: %w", err)
		}

		// Don't check for hostname uniqueness, since we try to emulate
		// dnsmasq here, which means that rmDynamicLease below will
		// simply empty the hostname of the dynamic lease if there even
		// is one.

		l.Hostname = hostname
	}

	// Perform the following actions in an anonymous function to make sure
	// that the lock gets unlocked before the notification step.
	func() {
		s.leasesLock.Lock()
		defer s.leasesLock.Unlock()

		err = s.rmDynamicLease(l)
		if err != nil {
			err = fmt.Errorf(
				"removing dynamic leases for %s (%s): %w",
				l.IP,
				l.HWAddr,
				err,
			)

			return
		}

		err = s.addLease(l)
		if err != nil {
			err = fmt.Errorf("adding static lease for %s (%s): %w", l.IP, l.HWAddr, err)

			return
		}
	}()
	if err != nil {
		return err
	}

	s.conf.notify(LeaseChangedDBStore)
	s.conf.notify(LeaseChangedAddedStatic)

	return nil
}

// RemoveStaticLease removes a static lease.  It is safe for concurrent use.
func (s *v4Server) RemoveStaticLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if len(l.IP) != 4 {
		return fmt.Errorf("invalid IP")
	}

	err = aghnet.ValidateHardwareAddress(l.HWAddr)
	if err != nil {
		return fmt.Errorf("validating lease: %w", err)
	}

	s.leasesLock.Lock()
	err = s.rmLease(l)
	if err != nil {
		s.leasesLock.Unlock()

		return err
	}
	s.leasesLock.Unlock()

	s.conf.notify(LeaseChangedDBStore)
	s.conf.notify(LeaseChangedRemovedStatic)

	return nil
}

// addrAvailable sends an ICP request to the specified IP address.  It returns
// true if the remote host doesn't reply, which probably means that the IP
// address is available.
//
// TODO(a.garipov): I'm not sure that this is the best way to do this.
func (s *v4Server) addrAvailable(target net.IP) (avail bool) {
	if s.conf.ICMPTimeout == 0 {
		return true
	}

	pinger, err := ping.NewPinger(target.String())
	if err != nil {
		log.Error("dhcpv4: ping.NewPinger(): %s", err)

		return true
	}

	pinger.SetPrivileged(true)
	pinger.Timeout = time.Duration(s.conf.ICMPTimeout) * time.Millisecond
	pinger.Count = 1
	reply := false
	pinger.OnRecv = func(_ *ping.Packet) {
		reply = true
	}

	log.Debug("dhcpv4: sending icmp echo to %s", target)

	err = pinger.Run()
	if err != nil {
		log.Error("dhcpv4: pinger.Run(): %s", err)

		return true
	}

	if reply {
		log.Info("dhcpv4: ip conflict: %s is already used by another device", target)

		return false
	}

	log.Debug("dhcpv4: icmp procedure is complete: %q", target)

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

		return !s.leasedOffsets.isSet(offset)
	})

	return ip.To4()
}

// Find an expired lease and return its index or -1
func (s *v4Server) findExpiredLease() int {
	now := time.Now()
	for i, lease := range s.leases {
		if !lease.IsStatic() && lease.Expiry.Before(now) {
			return i
		}
	}

	return -1
}

// reserveLease reserves a lease for a client by its MAC-address.  It returns
// nil if it couldn't allocate a new lease.
func (s *v4Server) reserveLease(mac net.HardwareAddr) (l *Lease, err error) {
	l = &Lease{
		HWAddr: make([]byte, len(mac)),
	}

	copy(l.HWAddr, mac)

	l.IP = s.nextIP()
	if l.IP == nil {
		i := s.findExpiredLease()
		if i < 0 {
			return nil, nil
		}

		copy(s.leases[i].HWAddr, mac)

		return s.leases[i], nil
	}

	err = s.addLease(l)
	if err != nil {
		return nil, err
	}

	return l, nil
}

func (s *v4Server) commitLease(l *Lease) {
	l.Expiry = time.Now().Add(s.conf.leaseTime)

	func() {
		s.leasesLock.Lock()
		defer s.leasesLock.Unlock()

		s.conf.notify(LeaseChangedDBStore)

		if l.Hostname != "" {
			s.leaseHosts.Add(l.Hostname)
		}
	}()

	s.conf.notify(LeaseChangedAdded)
}

// allocateLease allocates a new lease for the MAC address.  If there are no IP
// addresses left, both l and err are nil.
func (s *v4Server) allocateLease(mac net.HardwareAddr) (l *Lease, err error) {
	for {
		l, err = s.reserveLease(mac)
		if err != nil {
			return nil, fmt.Errorf("reserving a lease: %w", err)
		} else if l == nil {
			return nil, nil
		}

		if s.addrAvailable(l.IP) {
			return l, nil
		}

		s.blocklistLease(l)
	}
}

// processDiscover is the handler for the DHCP Discover request.
func (s *v4Server) processDiscover(req, resp *dhcpv4.DHCPv4) (l *Lease, err error) {
	mac := req.ClientHWAddr

	defer s.conf.notify(LeaseChangedDBStore)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	l = s.findLease(mac)
	if l != nil {
		reqIP := req.RequestedIPAddress()
		if len(reqIP) != 0 && !reqIP.Equal(l.IP) {
			log.Debug("dhcpv4: different RequestedIP: %s != %s", reqIP, l.IP)
		}

		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))

		return l, nil
	}

	l, err = s.allocateLease(mac)
	if err != nil {
		return nil, err
	} else if l == nil {
		log.Debug("dhcpv4: no more ip addresses")

		return nil, nil
	}

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))

	return l, nil
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

// checkLease checks if the pair of mac and ip is already leased.  The mismatch
// is true when the existing lease has the same hardware address but differs in
// its IP address.
func (s *v4Server) checkLease(mac net.HardwareAddr, ip net.IP) (lease *Lease, mismatch bool) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if !bytes.Equal(l.HWAddr, mac) {
			continue
		}

		if l.IP.Equal(ip) {
			return l, false
		}

		log.Debug(
			`dhcpv4: mismatched OptionRequestedIPAddress in req msg for %s`,
			mac,
		)

		return nil, true
	}

	return nil, false
}

// processRequest is the handler for the DHCP Request request.
func (s *v4Server) processRequest(req, resp *dhcpv4.DHCPv4) (lease *Lease, needsReply bool) {
	mac := req.ClientHWAddr
	reqIP := req.RequestedIPAddress()
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	sid := req.ServerIdentifier()
	if len(sid) != 0 && !sid.Equal(s.conf.dnsIPAddrs[0]) {
		log.Debug("dhcpv4: bad OptionServerIdentifier in req msg for %s", mac)

		return nil, false
	}

	if ip4 := reqIP.To4(); ip4 == nil {
		log.Debug("dhcpv4: bad OptionRequestedIPAddress in req msg for %s", mac)

		return nil, false
	}

	var mismatch bool
	if lease, mismatch = s.checkLease(mac, reqIP); mismatch {
		return nil, true
	}

	if lease == nil {
		log.Debug("dhcpv4: no reserved lease for %s", mac)

		return nil, true
	}

	if !lease.IsStatic() {
		cliHostname := req.HostName()
		hostname := s.validHostnameForClient(cliHostname, reqIP)
		if hostname != lease.Hostname && s.leaseHosts.Has(hostname) {
			log.Info("dhcpv4: hostname %q already exists", hostname)
			lease.Hostname = ""
		} else {
			lease.Hostname = hostname
		}

		s.commitLease(lease)
	} else if lease.Hostname != "" {
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

// processRequest is the handler for the DHCP Decline request.
func (s *v4Server) processDecline(req, resp *dhcpv4.DHCPv4) (err error) {
	s.conf.notify(LeaseChangedDBStore)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	mac := req.ClientHWAddr
	reqIP := req.RequestedIPAddress()
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	var oldLease *Lease
	for _, l := range s.leases {
		if bytes.Equal(l.HWAddr, mac) && l.IP.Equal(reqIP) {
			oldLease = l

			break
		}
	}

	if oldLease == nil {
		log.Info("dhcpv4: lease with ip %s for %s not found", reqIP, mac)

		return nil
	}

	err = s.rmDynamicLease(oldLease)
	if err != nil {
		return fmt.Errorf("removing old lease for %s: %w", mac, err)
	}

	newLease, err := s.allocateLease(mac)
	if err != nil {
		return fmt.Errorf("allocating new lease for %s: %w", mac, err)
	} else if newLease == nil {
		log.Info("dhcpv4: allocating new lease for %s: no more ip addresses", mac)

		resp.YourIPAddr = make([]byte, 4)
		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

		return nil
	}

	newLease.Hostname = oldLease.Hostname
	newLease.Expiry = time.Now().Add(s.conf.leaseTime)

	err = s.addLease(newLease)
	if err != nil {
		return fmt.Errorf("adding new lease for %s: %w", mac, err)
	}

	log.Info("dhcpv4: changed ip from %s to %s for %s", reqIP, newLease.IP, mac)

	resp.YourIPAddr = make([]byte, 4)
	copy(resp.YourIPAddr, newLease.IP)

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

	return nil
}

// processRelease is the handler for the DHCP Release request.
func (s *v4Server) processRelease(req, resp *dhcpv4.DHCPv4) (err error) {
	mac := req.ClientHWAddr
	reqIP := req.RequestedIPAddress()
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	// TODO(a.garipov): Add a separate notification type for dynamic lease
	// removal?
	defer s.conf.notify(LeaseChangedDBStore)

	n := 0
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if !bytes.Equal(l.HWAddr, mac) || !l.IP.Equal(reqIP) {
			continue
		}

		err = s.rmDynamicLease(l)
		if err != nil {
			err = fmt.Errorf("removing dynamic lease for %s: %w", mac, err)

			return
		}

		n++
	}

	log.Info("dhcpv4: released %d dynamic leases for %s", n, mac)

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

	return nil
}

// Find a lease associated with MAC and prepare response
// Return 1: OK
// Return 0: error; reply with Nak
// Return -1: error; don't reply
func (s *v4Server) process(req, resp *dhcpv4.DHCPv4) int {
	var err error

	resp.UpdateOption(dhcpv4.OptServerIdentifier(s.conf.dnsIPAddrs[0]))

	// TODO(a.garipov): Refactor this into handlers.
	var l *Lease
	switch req.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		l, err = s.processDiscover(req, resp)
		if err != nil {
			log.Error("dhcpv4: processing discover: %s", err)

			return 0
		}

		if l == nil {
			return 0
		}
	case dhcpv4.MessageTypeRequest:
		var toReply bool
		l, toReply = s.processRequest(req, resp)
		if l == nil {
			if toReply {
				return 0
			}
			return -1 // drop packet
		}
	case dhcpv4.MessageTypeDecline:
		err = s.processDecline(req, resp)
		if err != nil {
			log.Error("dhcpv4: processing decline: %s", err)

			return 0
		}
	case dhcpv4.MessageTypeRelease:
		err = s.processRelease(req, resp)
		if err != nil {
			log.Error("dhcpv4: processing release: %s", err)

			return 0
		}
	}

	if l != nil {
		resp.YourIPAddr = make([]byte, 4)
		copy(resp.YourIPAddr, l.IP)
	}

	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(s.conf.leaseTime))
	resp.UpdateOption(dhcpv4.OptRouter(s.conf.subnet.IP))
	resp.UpdateOption(dhcpv4.OptSubnetMask(s.conf.subnet.Mask))
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
		dhcpv4.MessageTypeRequest,
		dhcpv4.MessageTypeDecline,
		dhcpv4.MessageTypeRelease:
		// Go on.
	default:
		log.Debug("dhcpv4: unsupported message type %d", req.MessageType())

		return
	}

	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		log.Debug("dhcpv4: dhcpv4.New: %s", err)
		return
	}

	err = aghnet.ValidateHardwareAddress(req.ClientHWAddr)
	if err != nil {
		log.Error("dhcpv4: invalid ClientHWAddr: %s", err)

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
func (s *v4Server) Start() (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if !s.conf.Enabled {
		return nil
	}

	ifaceName := s.conf.InterfaceName
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("finding interface %s by name: %w", ifaceName, err)
	}

	log.Debug("dhcpv4: starting...")

	dnsIPAddrs, err := ifaceDNSIPAddrs(iface, ipVersion4, defaultMaxAttempts, defaultBackoff)
	if err != nil {
		return fmt.Errorf("interface %s: %w", ifaceName, err)
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
		if serr := s.srv.Serve(); errors.Is(serr, net.ErrClosed) {
			log.Info("dhcpv4: server is closed")

			return
		} else if serr != nil {
			log.Error("dhcpv4: srv.Serve: %s", serr)
		}
	}()

	// Signal to the clients containers in packages home and dnsforward that
	// it should reload the DHCP clients.
	s.conf.notify(LeaseChangedAdded)

	return nil
}

// Stop - stop server
func (s *v4Server) Stop() (err error) {
	if s.srv == nil {
		return
	}

	log.Debug("dhcpv4: stopping")
	err = s.srv.Close()
	if err != nil {
		return fmt.Errorf("closing dhcpv4 srv: %w", err)
	}

	// Signal to the clients containers in packages home and dnsforward that
	// it should remove all DHCP clients.
	s.conf.notify(LeaseChangedRemovedAll)

	s.srv = nil

	return nil
}

// Create DHCPv4 server
func v4Create(conf V4ServerConf) (srv DHCPServer, err error) {
	s := &v4Server{}
	s.conf = conf
	s.leaseHosts = aghstrings.NewSet()

	// TODO(a.garipov): Don't use a disabled server in other places or just
	// use an interface.
	if !conf.Enabled {
		return s, nil
	}

	var routerIP net.IP
	routerIP, err = tryTo4(s.conf.GatewayIP)
	if err != nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}

	if s.conf.SubnetMask == nil {
		return s, fmt.Errorf("dhcpv4: invalid subnet mask: %v", s.conf.SubnetMask)
	}

	subnetMask := make([]byte, 4)
	copy(subnetMask, s.conf.SubnetMask.To4())

	s.conf.subnet = &net.IPNet{
		IP:   routerIP,
		Mask: subnetMask,
	}

	s.conf.ipRange, err = newIPRange(conf.RangeStart, conf.RangeEnd)
	if err != nil {
		return s, fmt.Errorf("dhcpv4: %w", err)
	}

	s.leasedOffsets = newBitSet()

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
