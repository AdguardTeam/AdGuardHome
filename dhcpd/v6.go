// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"
)

const valueIAID = "ADGH" // value for IANA.ID

// v6Server - DHCPv6 server
type v6Server struct {
	srv        *server6.Server
	leasesLock sync.Mutex
	leases     []*Lease
	ipAddrs    [256]byte
	sid        dhcpv6.Duid

	ra raCtx // RA module

	conf V6ServerConf
}

// WriteDiskConfig4 - write configuration
func (s *v6Server) WriteDiskConfig4(c *V4ServerConf) {
}

// WriteDiskConfig6 - write configuration
func (s *v6Server) WriteDiskConfig6(c *V6ServerConf) {
	*c = s.conf
}

// Return TRUE if IP address is within range [start..0xff]
// nolint(staticcheck)
func ip6InRange(start net.IP, ip net.IP) bool {
	if len(start) != 16 {
		return false
	}
	if !bytes.Equal(start[:15], ip[:15]) {
		return false
	}
	return start[15] <= ip[15]
}

// ResetLeases - reset leases
func (s *v6Server) ResetLeases(ll []*Lease) {
	s.leases = nil
	for _, l := range ll {

		if l.Expiry.Unix() != leaseExpireStatic &&
			!ip6InRange(s.conf.ipStart, l.IP) {

			log.Debug("DHCPv6: skipping a lease with IP %v: not within current IP range", l.IP)
			continue
		}

		s.addLease(l)
	}
}

// GetLeases - get current leases
func (s *v6Server) GetLeases(flags int) []Lease {
	var result []Lease
	s.leasesLock.Lock()
	for _, lease := range s.leases {

		if lease.Expiry.Unix() == leaseExpireStatic {
			if (flags & LeasesStatic) != 0 {
				result = append(result, *lease)
			}

		} else {
			if (flags & LeasesDynamic) != 0 {
				result = append(result, *lease)
			}
		}
	}
	s.leasesLock.Unlock()
	return result
}

// GetLeasesRef - get leases
func (s *v6Server) GetLeasesRef() []*Lease {
	return s.leases
}

// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
func (s *v6Server) FindMACbyIP(ip net.IP) net.HardwareAddr {
	now := time.Now().Unix()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if l.IP.Equal(ip) {
			unix := l.Expiry.Unix()
			if unix > now || unix == leaseExpireStatic {
				return l.HWAddr
			}
		}
	}
	return nil
}

// Remove (swap) lease by index
func (s *v6Server) leaseRemoveSwapByIndex(i int) {
	s.ipAddrs[s.leases[i].IP[15]] = 0
	log.Debug("DHCPv6: removed lease %s", s.leases[i].HWAddr)

	n := len(s.leases)
	if i != n-1 {
		s.leases[i] = s.leases[n-1] // swap with the last element
	}
	s.leases = s.leases[:n-1]
}

// Remove a dynamic lease with the same properties
// Return error if a static lease is found
func (s *v6Server) rmDynamicLease(lease Lease) error {
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

// AddStaticLease - add a static lease
func (s *v6Server) AddStaticLease(l Lease) error {
	if len(l.IP) != 16 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}

	l.Expiry = time.Unix(leaseExpireStatic, 0)

	s.leasesLock.Lock()
	err := s.rmDynamicLease(l)
	if err != nil {
		s.leasesLock.Unlock()
		return err
	}
	s.addLease(&l)
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()

	s.conf.notify(LeaseChangedAddedStatic)
	return nil
}

// RemoveStaticLease - remove a static lease
func (s *v6Server) RemoveStaticLease(l Lease) error {
	if len(l.IP) != 16 {
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

// Add a lease
func (s *v6Server) addLease(l *Lease) {
	s.leases = append(s.leases, l)
	s.ipAddrs[l.IP[15]] = 1
	log.Debug("DHCPv6: added lease %s <-> %s", l.IP, l.HWAddr)
}

// Remove a lease with the same properties
func (s *v6Server) rmLease(lease Lease) error {
	for i, l := range s.leases {
		if net.IP.Equal(l.IP, lease.IP) {

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

// Find lease by MAC
func (s *v6Server) findLease(mac net.HardwareAddr) *Lease {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

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
		if lease.Expiry.Unix() != leaseExpireStatic &&
			lease.Expiry.Unix() <= now {
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
func (s *v6Server) reserveLease(mac net.HardwareAddr) *Lease {
	l := Lease{}
	l.HWAddr = make([]byte, 6)
	copy(l.HWAddr, mac)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	copy(l.IP, s.conf.ipStart)
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

func (s *v6Server) commitDynamicLease(l *Lease) {
	l.Expiry = time.Now().Add(s.conf.leaseTime)

	s.leasesLock.Lock()
	s.conf.notify(LeaseChangedDBStore)
	s.leasesLock.Unlock()
	s.conf.notify(LeaseChangedAdded)
}

// Check Client ID
func (s *v6Server) checkCID(msg *dhcpv6.Message) error {
	if msg.Options.ClientID() == nil {
		return fmt.Errorf("DHCPv6: no ClientID option in request")
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
			return fmt.Errorf("DHCPv6: drop packet: ServerID option in message %s", msg.Type().String())
		}

	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRelease,
		dhcpv6.MessageTypeDecline:

		if sid == nil {
			return fmt.Errorf("DHCPv6: drop packet: no ServerID option in message %s", msg.Type().String())
		}
		if !sid.Equal(s.sid) {
			return fmt.Errorf("DHCPv6: drop packet: mismatched ServerID option in message %s: %s",
				msg.Type().String(), sid.String())
		}
	}

	return nil
}

// . IAAddress must be equal to the lease's IP
func (s *v6Server) checkIA(msg *dhcpv6.Message, lease *Lease) error {
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

		if !oiaAddr.IPv6Addr.Equal(lease.IP) {
			return fmt.Errorf("invalid IANA.Addr option in %s", msg.Type().String())
		}
	}
	return nil
}

// Store lease in DB (if necessary) and return lease life time
func (s *v6Server) commitLease(msg *dhcpv6.Message, lease *Lease) time.Duration {
	lifetime := s.conf.leaseTime

	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		//

	case dhcpv6.MessageTypeConfirm:
		lifetime = lease.Expiry.Sub(time.Now())

	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:

		if lease.Expiry.Unix() != leaseExpireStatic {
			s.commitDynamicLease(lease)
		}
	}
	return lifetime
}

// Find a lease associated with MAC and prepare response
func (s *v6Server) process(msg *dhcpv6.Message, req dhcpv6.DHCPv6, resp dhcpv6.DHCPv6) bool {
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
		log.Debug("DHCPv6: dhcpv6.ExtractMAC: %s", err)
		return false
	}

	lease := s.findLease(mac)
	if lease == nil {
		log.Debug("DHCPv6: no lease for: %s", mac)

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
		log.Debug("DHCPv6: %s", err)
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
		IPv6Addr:          lease.IP,
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
		log.Error("DHCPv6: %s", err)
		return
	}

	log.Debug("DHCPv6: received: %s", req.Summary())

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
		fallthrough

	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind,
		dhcpv6.MessageTypeRelease,
		dhcpv6.MessageTypeInformationRequest:
		resp, err = dhcpv6.NewReplyFromMessage(msg)

	default:
		log.Error("DHCPv6: message type %d not supported", msg.Type())
		return
	}

	if err != nil {
		log.Error("DHCPv6: %s", err)
		return
	}

	resp.AddOption(dhcpv6.OptServerID(s.sid))

	_ = s.process(msg, req, resp)

	log.Debug("DHCPv6: sending: %s", resp.Summary())

	_, err = conn.WriteTo(resp.ToBytes(), peer)
	if err != nil {
		log.Error("DHCPv6: conn.Write to %s failed: %s", peer, err)
		return
	}
}

// Get IPv6 address list
func getIfaceIPv6(iface net.Interface) []net.IP {
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
		if ipnet.IP.To4() == nil {
			res = append(res, ipnet.IP)
		}
	}
	return res
}

// initialize RA module
func (s *v6Server) initRA(iface *net.Interface) error {
	// choose the source IP address - should be link-local-unicast
	s.ra.ipAddr = s.conf.dnsIPAddrs[0]
	for _, ip := range s.conf.dnsIPAddrs {
		if ip.IsLinkLocalUnicast() {
			s.ra.ipAddr = ip
			break
		}
	}

	s.ra.raAllowSlaac = s.conf.RaAllowSlaac
	s.ra.raSlaacOnly = s.conf.RaSlaacOnly
	s.ra.dnsIPAddr = s.ra.ipAddr
	s.ra.prefixIPAddr = s.conf.ipStart
	s.ra.ifaceName = s.conf.InterfaceName
	s.ra.iface = iface
	s.ra.packetSendPeriod = 1 * time.Second
	return s.ra.Init()
}

// Start - start server
func (s *v6Server) Start() error {
	if !s.conf.Enabled {
		return nil
	}

	iface, err := net.InterfaceByName(s.conf.InterfaceName)
	if err != nil {
		return wrapErrPrint(err, "Couldn't find interface by name %s", s.conf.InterfaceName)
	}

	s.conf.dnsIPAddrs = getIfaceIPv6(*iface)
	if len(s.conf.dnsIPAddrs) == 0 {
		log.Debug("DHCPv6: no IPv6 address for interface %s", iface.Name)
		return nil
	}

	err = s.initRA(iface)
	if err != nil {
		return err
	}

	// don't initialize DHCPv6 server if we must force the clients to use SLAAC
	if s.conf.RaSlaacOnly {
		log.Debug("DHCPv6: not starting DHCPv6 server due to ra_slaac_only=true")
		return nil
	}

	log.Debug("DHCPv6: starting...")

	if len(iface.HardwareAddr) != 6 {
		return fmt.Errorf("DHCPv6: invalid MAC %s", iface.HardwareAddr)
	}
	s.sid = dhcpv6.Duid{
		Type:          dhcpv6.DUID_LLT,
		HwType:        iana.HWTypeEthernet,
		LinkLayerAddr: iface.HardwareAddr,
		Time:          dhcpv6.GetTime(),
	}

	laddr := &net.UDPAddr{
		IP:   net.ParseIP("::"),
		Port: dhcpv6.DefaultServerPort,
	}
	s.srv, err = server6.NewServer(iface.Name, laddr, s.packetHandler, server6.WithDebugLogger())
	if err != nil {
		return err
	}

	go func() {
		err = s.srv.Serve()
		log.Debug("DHCPv6: srv.Serve: %s", err)
	}()

	return nil
}

// Stop - stop server
func (s *v6Server) Stop() {
	s.ra.Close()

	// DHCPv6 server may not be initialized if ra_slaac_only=true
	if s.srv == nil {
		return
	}

	log.Debug("DHCPv6: stopping")
	err := s.srv.Close()
	if err != nil {
		log.Error("DHCPv6: srv.Close: %s", err)
	}
	// now server.Serve() will return
	s.srv = nil
}

// Create DHCPv6 server
func v6Create(conf V6ServerConf) (DHCPServer, error) {
	s := &v6Server{}
	s.conf = conf

	if !conf.Enabled {
		return s, nil
	}

	s.conf.ipStart = net.ParseIP(conf.RangeStart)
	if s.conf.ipStart == nil || s.conf.ipStart.To16() == nil {
		return s, fmt.Errorf("DHCPv6: invalid range-start IP: %s", conf.RangeStart)
	}

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 24
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s, nil
}
