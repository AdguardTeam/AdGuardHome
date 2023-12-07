//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"
)

const valueIAID = "ADGH" // value for IANA.ID

// v6Server is a DHCPv6 server.
//
// TODO(a.garipov): Think about unifying this and v4Server.
type v6Server struct {
	ra   raCtx
	conf V6ServerConf
	sid  dhcpv6.DUID
	srv  *server6.Server

	leases     []*dhcpsvc.Lease
	leasesLock sync.Mutex
	ipAddrs    [256]byte
}

// WriteDiskConfig4 - write configuration
func (s *v6Server) WriteDiskConfig4(c *V4ServerConf) {
}

// WriteDiskConfig6 - write configuration
func (s *v6Server) WriteDiskConfig6(c *V6ServerConf) {
	*c = s.conf
}

// Return TRUE if IP address is within range [start..0xff]
func ip6InRange(start, ip net.IP) bool {
	if len(start) != 16 {
		return false
	}
	//lint:ignore SA1021 TODO(e.burkov): Ignore this for now, think about
	// using masks.
	if !bytes.Equal(start[:15], ip[:15]) {
		return false
	}
	return start[15] <= ip[15]
}

// HostByIP implements the [Interface] interface for *v6Server.
func (s *v6Server) HostByIP(ip netip.Addr) (host string) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if l.IP == ip {
			return l.Hostname
		}
	}

	return ""
}

// IPByHost implements the [Interface] interface for *v6Server.
func (s *v6Server) IPByHost(host string) (ip netip.Addr) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if l.Hostname == host {
			return l.IP
		}
	}

	return netip.Addr{}
}

// ResetLeases resets leases.
func (s *v6Server) ResetLeases(leases []*dhcpsvc.Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv6: %w") }()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	s.leases = nil
	for _, l := range leases {
		ip := net.IP(l.IP.AsSlice())
		if !l.IsStatic && !ip6InRange(s.conf.ipStart, ip) {

			log.Debug("dhcpv6: skipping a lease with IP %v: not within current IP range", l.IP)

			continue
		}

		s.addLease(l)
	}

	return nil
}

// GetLeases returns the list of current DHCP leases.  It is safe for concurrent
// use.
func (s *v6Server) GetLeases(flags GetLeasesFlags) (leases []*dhcpsvc.Lease) {
	// The function shouldn't return nil value because zero-length slice
	// behaves differently in cases like marshalling.  Our front-end also
	// requires non-nil value in the response.
	leases = []*dhcpsvc.Lease{}
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if l.IsStatic {
			if (flags & LeasesStatic) != 0 {
				leases = append(leases, l.Clone())
			}
		} else {
			if (flags & LeasesDynamic) != 0 {
				leases = append(leases, l.Clone())
			}
		}
	}

	return leases
}

// getLeasesRef returns the actual leases slice.  For internal use only.
func (s *v6Server) getLeasesRef() []*dhcpsvc.Lease {
	return s.leases
}

// FindMACbyIP implements the [Interface] for *v6Server.
func (s *v6Server) FindMACbyIP(ip netip.Addr) (mac net.HardwareAddr) {
	now := time.Now()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	if !ip.Is6() {
		return nil
	}

	for _, l := range s.leases {
		if l.IP == ip {
			if l.IsStatic || l.Expiry.After(now) {
				return l.HWAddr
			}
		}
	}

	return nil
}

// Remove (swap) lease by index
func (s *v6Server) leaseRemoveSwapByIndex(i int) {
	leaseIP := s.leases[i].IP.As16()
	s.ipAddrs[leaseIP[15]] = 0
	log.Debug("dhcpv6: removed lease %s", s.leases[i].HWAddr)

	n := len(s.leases)
	if i != n-1 {
		s.leases[i] = s.leases[n-1] // swap with the last element
	}
	s.leases = s.leases[:n-1]
}

// Remove a dynamic lease with the same properties
// Return error if a static lease is found
func (s *v6Server) rmDynamicLease(lease *dhcpsvc.Lease) (err error) {
	for i := 0; i < len(s.leases); i++ {
		l := s.leases[i]

		if bytes.Equal(l.HWAddr, lease.HWAddr) {
			if l.IsStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.leaseRemoveSwapByIndex(i)
			if i == len(s.leases) {
				break
			}

			l = s.leases[i]
		}

		if l.IP == lease.IP {
			if l.IsStatic {
				return fmt.Errorf("static lease already exists")
			}

			s.leaseRemoveSwapByIndex(i)
		}
	}

	return nil
}

// AddStaticLease adds a static lease.  It is safe for concurrent use.
func (s *v6Server) AddStaticLease(l *dhcpsvc.Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv6: %w") }()

	if !l.IP.Is6() {
		return fmt.Errorf("invalid IP")
	}

	err = netutil.ValidateMAC(l.HWAddr)
	if err != nil {
		return fmt.Errorf("validating lease: %w", err)
	}

	l.IsStatic = true

	s.leasesLock.Lock()
	err = s.rmDynamicLease(l)
	if err != nil {
		s.leasesLock.Unlock()

		return err
	}

	s.addLease(l)
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()

	s.conf.notify(LeaseChangedAddedStatic)

	return nil
}

// UpdateStaticLease updates IP, hostname of the static lease.
func (s *v6Server) UpdateStaticLease(l *dhcpsvc.Lease) (err error) {
	defer func() {
		if err != nil {
			err = errors.Annotate(err, "dhcpv6: updating static lease: %w")

			return
		}

		s.conf.notify(LeaseChangedDBStore)
		s.conf.notify(LeaseChangedRemovedStatic)
	}()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	found := s.findLease(l.HWAddr)
	if found == nil {
		return fmt.Errorf("can't find lease %s", l.HWAddr)
	}

	err = s.rmLease(found)
	if err != nil {
		return fmt.Errorf("removing previous lease for %s (%s): %w", l.IP, l.HWAddr, err)
	}

	s.addLease(l)

	return nil
}

// RemoveStaticLease removes a static lease.  It is safe for concurrent use.
func (s *v6Server) RemoveStaticLease(l *dhcpsvc.Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv6: %w") }()

	if !l.IP.Is6() {
		return fmt.Errorf("invalid IP")
	}

	err = netutil.ValidateMAC(l.HWAddr)
	if err != nil {
		return fmt.Errorf("validating lease: %w", err)
	}

	s.leasesLock.Lock()
	err = s.rmLease(l)
	if err != nil {
		s.leasesLock.Unlock()
		return err
	}
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()
	s.conf.notify(LeaseChangedRemovedStatic)
	return nil
}

// Add a lease
func (s *v6Server) addLease(l *dhcpsvc.Lease) {
	s.leases = append(s.leases, l)
	ip := l.IP.As16()
	s.ipAddrs[ip[15]] = 1
	log.Debug("dhcpv6: added lease %s <-> %s", l.IP, l.HWAddr)
}

// Remove a lease with the same properties
func (s *v6Server) rmLease(lease *dhcpsvc.Lease) (err error) {
	for i, l := range s.leases {
		if l.IP == lease.IP {
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

// Find lease by MAC.
func (s *v6Server) findLease(mac net.HardwareAddr) (lease *dhcpsvc.Lease) {
	for i := range s.leases {
		if bytes.Equal(mac, s.leases[i].HWAddr) {
			return s.leases[i]
		}
	}

	return nil
}

// Find an expired lease and return its index or -1
func (s *v6Server) findExpiredLease() int {
	now := time.Now().Unix()
	for i, lease := range s.leases {
		if !lease.IsStatic && lease.Expiry.Unix() <= now {
			return i
		}
	}
	return -1
}

// Get next free IP
func (s *v6Server) findFreeIP() net.IP {
	for i := s.conf.ipStart[15]; ; i++ {
		if s.ipAddrs[i] == 0 {
			ip := make([]byte, 16)
			copy(ip, s.conf.ipStart)
			ip[15] = i
			return ip
		}
		if i == 0xff {
			break
		}
	}
	return nil
}

// Reserve lease for MAC
func (s *v6Server) reserveLease(mac net.HardwareAddr) *dhcpsvc.Lease {
	l := dhcpsvc.Lease{
		HWAddr: make([]byte, len(mac)),
	}

	copy(l.HWAddr, mac)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	ip := s.findFreeIP()
	if ip == nil {
		i := s.findExpiredLease()
		if i < 0 {
			return nil
		}

		copy(s.leases[i].HWAddr, mac)

		return s.leases[i]
	}

	netIP, ok := netip.AddrFromSlice(ip)
	if !ok {
		return nil
	}

	l.IP = netIP

	s.addLease(&l)

	return &l
}

func (s *v6Server) commitDynamicLease(l *dhcpsvc.Lease) {
	l.Expiry = time.Now().Add(s.conf.leaseTime)

	s.leasesLock.Lock()
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()
	s.conf.notify(LeaseChangedAdded)
}

// Check Client ID
func (s *v6Server) checkCID(msg *dhcpv6.Message) error {
	if msg.Options.ClientID() == nil {
		return fmt.Errorf("dhcpv6: no ClientID option in request")
	}

	return nil
}

// Check ServerID policy
func (s *v6Server) checkSID(msg *dhcpv6.Message) error {
	sid := msg.Options.ServerID()

	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRebind:

		if sid != nil {
			return fmt.Errorf("dhcpv6: drop packet: ServerID option in message %s", msg.Type().String())
		}
	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRelease,
		dhcpv6.MessageTypeDecline:
		if sid == nil {
			return fmt.Errorf("dhcpv6: drop packet: no ServerID option in message %s", msg.Type().String())
		}

		if !sid.Equal(s.sid) {
			return fmt.Errorf("dhcpv6: drop packet: mismatched ServerID option in message %s: %s",
				msg.Type().String(), sid.String())
		}
	}

	return nil
}

// . IAAddress must be equal to the lease's IP
func (s *v6Server) checkIA(msg *dhcpv6.Message, lease *dhcpsvc.Lease) error {
	switch msg.Type() {
	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:

		oia := msg.Options.OneIANA()
		if oia == nil {
			return fmt.Errorf("no IANA option in %s", msg.Type().String())
		}

		oiaAddr := oia.Options.OneAddress()
		if oiaAddr == nil {
			return fmt.Errorf("no IANA.Addr option in %s", msg.Type().String())
		}

		leaseIP := net.IP(lease.IP.AsSlice())
		if !oiaAddr.IPv6Addr.Equal(leaseIP) {
			return fmt.Errorf("invalid IANA.Addr option in %s", msg.Type().String())
		}
	}
	return nil
}

// Store lease in DB (if necessary) and return lease life time
func (s *v6Server) commitLease(msg *dhcpv6.Message, lease *dhcpsvc.Lease) time.Duration {
	lifetime := s.conf.leaseTime

	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		//

	case dhcpv6.MessageTypeConfirm:
		lifetime = time.Until(lease.Expiry)

	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:

		if !lease.IsStatic {
			s.commitDynamicLease(lease)
		}
	}
	return lifetime
}

// Find a lease associated with MAC and prepare response
func (s *v6Server) process(msg *dhcpv6.Message, req, resp dhcpv6.DHCPv6) bool {
	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit,
		dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:
		// continue

	default:
		return false
	}

	mac, err := dhcpv6.ExtractMAC(req)
	if err != nil {
		log.Debug("dhcpv6: dhcpv6.ExtractMAC: %s", err)

		return false
	}

	var lease *dhcpsvc.Lease
	func() {
		s.leasesLock.Lock()
		defer s.leasesLock.Unlock()

		lease = s.findLease(mac)
	}()

	if lease == nil {
		log.Debug("dhcpv6: no lease for: %s", mac)

		switch msg.Type() {

		case dhcpv6.MessageTypeSolicit:
			lease = s.reserveLease(mac)
			if lease == nil {
				return false
			}

		default:
			return false
		}
	}

	err = s.checkIA(msg, lease)
	if err != nil {
		log.Debug("dhcpv6: %s", err)

		return false
	}

	lifetime := s.commitLease(msg, lease)

	oia := &dhcpv6.OptIANA{
		T1: lifetime / 2,
		T2: time.Duration(float32(lifetime) / 1.5),
	}
	roia := msg.Options.OneIANA()
	if roia != nil {
		copy(oia.IaId[:], roia.IaId[:])
	} else {
		copy(oia.IaId[:], []byte(valueIAID))
	}
	oiaAddr := &dhcpv6.OptIAAddress{
		IPv6Addr:          net.IP(lease.IP.AsSlice()),
		PreferredLifetime: lifetime,
		ValidLifetime:     lifetime,
	}
	oia.Options = dhcpv6.IdentityOptions{
		Options: []dhcpv6.Option{oiaAddr},
	}
	resp.AddOption(oia)

	if msg.IsOptionRequested(dhcpv6.OptionDNSRecursiveNameServer) {
		resp.UpdateOption(dhcpv6.OptDNS(s.conf.dnsIPAddrs...))
	}

	fqdn := msg.GetOneOption(dhcpv6.OptionFQDN)
	if fqdn != nil {
		resp.AddOption(fqdn)
	}

	resp.AddOption(&dhcpv6.OptStatusCode{
		StatusCode:    iana.StatusSuccess,
		StatusMessage: "success",
	})
	return true
}

// 1.
// fe80::* (client) --(Solicit + ClientID+IANA())-> ff02::1:2
// server -(Advertise + ClientID+ServerID+IANA(IAAddress)> fe80::*
// fe80::* --(Request + ClientID+ServerID+IANA(IAAddress))-> ff02::1:2
// server -(Reply + ClientID+ServerID+IANA(IAAddress)+DNS)> fe80::*
//
// 2.
// fe80::* --(Confirm|Renew|Rebind + ClientID+IANA(IAAddress))-> ff02::1:2
// server -(Reply + ClientID+ServerID+IANA(IAAddress)+DNS)> fe80::*
//
// 3.
// fe80::* --(Release + ClientID+ServerID+IANA(IAAddress))-> ff02::1:2
func (s *v6Server) packetHandler(conn net.PacketConn, peer net.Addr, req dhcpv6.DHCPv6) {
	msg, err := req.GetInnerMessage()
	if err != nil {
		log.Error("dhcpv6: %s", err)

		return
	}

	log.Debug("dhcpv6: received: %s", req.Summary())

	err = s.checkCID(msg)
	if err != nil {
		log.Debug("%s", err)
		return
	}

	err = s.checkSID(msg)
	if err != nil {
		log.Debug("%s", err)
		return
	}

	var resp dhcpv6.DHCPv6

	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		if msg.GetOneOption(dhcpv6.OptionRapidCommit) == nil {
			resp, err = dhcpv6.NewAdvertiseFromSolicit(msg)

			break
		}

		resp, err = dhcpv6.NewReplyFromMessage(msg)
	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind,
		dhcpv6.MessageTypeRelease,
		dhcpv6.MessageTypeInformationRequest:
		resp, err = dhcpv6.NewReplyFromMessage(msg)
	default:
		log.Error("dhcpv6: message type %d not supported", msg.Type())

		return
	}
	if err != nil {
		log.Error("dhcpv6: %s", err)

		return
	}

	resp.AddOption(dhcpv6.OptServerID(s.sid))

	_ = s.process(msg, req, resp)

	log.Debug("dhcpv6: sending: %s", resp.Summary())

	_, err = conn.WriteTo(resp.ToBytes(), peer)
	if err != nil {
		log.Error("dhcpv6: conn.Write to %s failed: %s", peer, err)

		return
	}
}

// configureDNSIPAddrs updates v6Server configuration with the slice of DNS IP
// addresses of provided interface iface.  Initializes RA module.
func (s *v6Server) configureDNSIPAddrs(iface *net.Interface) (ok bool, err error) {
	dnsIPAddrs, err := aghnet.IfaceDNSIPAddrs(
		iface,
		aghnet.IPVersion6,
		defaultMaxAttempts,
		defaultBackoff,
	)
	if err != nil {
		return false, fmt.Errorf("interface %s: %w", iface.Name, err)
	}

	if len(dnsIPAddrs) == 0 {
		return false, nil
	}

	s.conf.dnsIPAddrs = dnsIPAddrs

	return true, s.initRA(iface)
}

// initRA initializes RA module.
func (s *v6Server) initRA(iface *net.Interface) (err error) {
	// Choose the source IP address - should be link-local-unicast.
	s.ra.ipAddr = s.conf.dnsIPAddrs[0]
	for _, ip := range s.conf.dnsIPAddrs {
		if ip.IsLinkLocalUnicast() {
			s.ra.ipAddr = ip
			break
		}
	}

	s.ra.raAllowSLAAC = s.conf.RAAllowSLAAC
	s.ra.raSLAACOnly = s.conf.RASLAACOnly
	s.ra.dnsIPAddr = s.ra.ipAddr
	s.ra.prefixIPAddr = s.conf.ipStart
	s.ra.ifaceName = s.conf.InterfaceName
	s.ra.iface = iface
	s.ra.packetSendPeriod = 1 * time.Second

	return s.ra.Init()
}

// Start starts the IPv6 DHCP server.
func (s *v6Server) Start() (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv6: %w") }()

	if !s.conf.Enabled {
		return nil
	}

	ifaceName := s.conf.InterfaceName
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("finding interface %s by name: %w", ifaceName, err)
	}

	log.Debug("dhcpv6: starting...")

	ok, err := s.configureDNSIPAddrs(iface)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if !ok {
		// No available IP addresses which may appear later.
		return nil
	}

	// Don't initialize DHCPv6 server if we must force the clients to use SLAAC.
	if s.conf.RASLAACOnly {
		log.Debug("not starting dhcpv6 server due to ra_slaac_only=true")

		return nil
	}

	err = netutil.ValidateMAC(iface.HardwareAddr)
	if err != nil {
		return fmt.Errorf("validating interface %s: %w", iface.Name, err)
	}

	s.sid = &dhcpv6.DUIDLLT{
		HWType:        iana.HWTypeEthernet,
		LinkLayerAddr: iface.HardwareAddr,
		Time:          dhcpv6.GetTime(),
	}

	s.srv, err = server6.NewServer(iface.Name, nil, s.packetHandler, server6.WithDebugLogger())
	if err != nil {
		return err
	}

	log.Debug("dhcpv6: listening...")

	go func() {
		if sErr := s.srv.Serve(); errors.Is(sErr, net.ErrClosed) {
			log.Info("dhcpv6: server is closed")
		} else if sErr != nil {
			log.Error("dhcpv6: srv.Serve: %s", sErr)
		}
	}()

	return nil
}

// Stop - stop server
func (s *v6Server) Stop() (err error) {
	err = s.ra.Close()
	if err != nil {
		return fmt.Errorf("closing ra ctx: %w", err)
	}

	// DHCPv6 server may not be initialized if ra_slaac_only=true
	if s.srv == nil {
		return
	}

	log.Debug("dhcpv6: stopping")
	err = s.srv.Close()
	if err != nil {
		return fmt.Errorf("closing dhcpv6 srv: %w", err)
	}

	// now server.Serve() will return
	s.srv = nil

	return nil
}

// Create DHCPv6 server
func v6Create(conf V6ServerConf) (DHCPServer, error) {
	s := &v6Server{}
	s.conf = conf

	if !conf.Enabled {
		return s, nil
	}

	s.conf.ipStart = conf.RangeStart
	if s.conf.ipStart == nil || s.conf.ipStart.To16() == nil {
		return s, fmt.Errorf("dhcpv6: invalid range-start IP: %s", conf.RangeStart)
	}

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = timeutil.Day
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s, nil
}
