//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net"
	"net/netip"
	"slices"
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

	leases                 []*dhcpsvc.Lease
	leasesLock             sync.Mutex
	ipAddrs                [256]byte
	dnsIPAddrsMu           sync.RWMutex
	advertisedPrefixes     map[netip.Prefix]struct{}
	renewablePrefixes      map[netip.Prefix]struct{}
	preferredUntilByPrefix map[netip.Prefix]time.Time
	validUntilByPrefix     map[netip.Prefix]time.Time
	restoredRenewable      map[netip.Prefix]struct{}
	restoredDeprecated     map[netip.Prefix]time.Time
	persistRestoredMeta    bool
}

// WriteDiskConfig4 - write configuration
func (s *v6Server) WriteDiskConfig4(c *V4ServerConf) {
}

// WriteDiskConfig6 - write configuration
func (s *v6Server) WriteDiskConfig6(c *V6ServerConf) {
	*c = V6ServerConf{
		Logger:             s.conf.Logger,
		CommandConstructor: s.conf.CommandConstructor,
		Enabled:            s.conf.Enabled,
		InterfaceName:      s.conf.InterfaceName,
		RangeStart:         bytes.Clone(s.conf.RangeStart),
		PrefixSource:       s.conf.PrefixSource,
		LeaseDuration:      s.conf.LeaseDuration,
		RASLAACOnly:        s.conf.RASLAACOnly,
		RAAllowSLAAC:       s.conf.RAAllowSLAAC,
		leaseTime:          s.conf.leaseTime,
		notify:             s.conf.notify,
	}

	s.leasesLock.Lock()
	c.ipStart = bytes.Clone(s.conf.ipStart)
	s.leasesLock.Unlock()

	s.dnsIPAddrsMu.RLock()
	c.dnsIPAddrs = slices.Clone(s.conf.dnsIPAddrs)
	s.dnsIPAddrsMu.RUnlock()

	c.PrefixSource = c.NormalizedPrefixSource()
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

	if len(leases) == 0 {
		s.restoredRenewable = nil
		s.restoredDeprecated = nil
	}

	s.leases = nil
	for _, l := range leases {
		ip := net.IP(l.IP.AsSlice())
		if !l.IsStatic && !s.keepInterfaceLeaseOnReset(l.IP, ip) {

			log.Debug("dhcpv6: skipping a lease with IP %v: not within current IP range", l.IP)

			continue
		}

		s.addLease(l)
	}

	return nil
}

// keepInterfaceLeaseOnReset reports whether a dynamic lease should be kept
// while rebuilding in-memory state from disk.
func (s *v6Server) keepInterfaceLeaseOnReset(ip netip.Addr, raw net.IP) (ok bool) {
	if s.conf.NormalizedPrefixSource() != V6PrefixSourceInterface {
		return ip6InRange(s.conf.ipStart, raw)
	}

	if len(s.advertisedPrefixes) == 0 {
		return true
	}

	return leasePrefixAdvertised(s.advertisedPrefixes, ip)
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

// dbSnapshot returns a consistent snapshot of DHCPv6 leases and persisted
// prefix-tracking metadata.
func (s *v6Server) dbSnapshot(now time.Time) (
	leases []*dhcpsvc.Lease,
	renewable map[netip.Prefix]struct{},
	deprecated map[netip.Prefix]time.Time,
) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	leases = make([]*dhcpsvc.Lease, 0, len(s.leases))
	for _, l := range s.leases {
		leases = append(leases, l.Clone())
	}

	renewable, deprecated = s.deprecatedPrefixMetaLocked(now)

	return leases, renewable, deprecated
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
	s.unmarkLeaseOccupied(s.leases[i])
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
	s.leasesLock.Unlock()

	s.conf.notify(LeaseChangedDBStore)
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
	s.leasesLock.Unlock()

	s.conf.notify(LeaseChangedDBStore)
	s.conf.notify(LeaseChangedRemovedStatic)
	return nil
}

// Add a lease
func (s *v6Server) addLease(l *dhcpsvc.Lease) {
	s.leases = append(s.leases, l)
	s.markLeaseOccupied(l)
	log.Debug("dhcpv6: added lease %s <-> %s", l.IP, l.HWAddr)
}

// ipInCurrentPool reports whether ip belongs to the currently active pool.
func (s *v6Server) ipInCurrentPool(ip netip.Addr) (ok bool) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	return s.ipInCurrentPoolLocked(ip)
}

// ipInCurrentPoolLocked reports whether ip belongs to the currently active
// pool.  s.leasesLock must be held.
func (s *v6Server) ipInCurrentPoolLocked(ip netip.Addr) (ok bool) {
	return ip.Is6() && ip6InRange(s.conf.ipStart, net.IP(ip.AsSlice()))
}

// markLeaseOccupied updates the dynamic-pool occupancy bitmap for l.
func (s *v6Server) markLeaseOccupied(l *dhcpsvc.Lease) {
	if !s.ipInCurrentPoolLocked(l.IP) {
		return
	}

	ip := l.IP.As16()
	s.ipAddrs[ip[15]] = 1
}

// unmarkLeaseOccupied updates the dynamic-pool occupancy bitmap after l is
// removed.
func (s *v6Server) unmarkLeaseOccupied(l *dhcpsvc.Lease) {
	if !s.ipInCurrentPoolLocked(l.IP) {
		return
	}

	ip := l.IP.As16()
	s.ipAddrs[ip[15]] = 0
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
		if !lease.IsStatic && s.ipInCurrentPoolLocked(lease.IP) && lease.Expiry.Unix() <= now {
			return i
		}
	}
	return -1
}

// Get next free IP
func (s *v6Server) findFreeIP() net.IP {
	if len(s.conf.ipStart) != net.IPv6len {
		return nil
	}

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

	if len(s.conf.ipStart) != net.IPv6len {
		return nil
	}

	for i := 0; i < len(s.leases); i++ {
		if s.leases[i].IsStatic ||
			!bytes.Equal(s.leases[i].HWAddr, mac) ||
			!s.ipInCurrentPoolLocked(s.leases[i].IP) {
			continue
		}

		s.leaseRemoveSwapByIndex(i)
		i--
	}

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

func (s *v6Server) commitDynamicLease(l *dhcpsvc.Lease, lifetime time.Duration) {
	l.Expiry = time.Now().Add(lifetime)

	s.conf.notify(LeaseChangedDBStore)
	s.conf.notify(LeaseChangedAdded)
}

// dnsIPAddrs returns the current DHCPv6 DNS server addresses.
func (s *v6Server) dnsIPAddrs() (addrs []net.IP) {
	s.dnsIPAddrsMu.RLock()
	defer s.dnsIPAddrsMu.RUnlock()

	if len(s.conf.dnsIPAddrs) == 0 {
		return nil
	}

	return slices.Clone(s.conf.dnsIPAddrs)
}

// setDNSIPAddrs updates the current DHCPv6 DNS server addresses.
func (s *v6Server) setDNSIPAddrs(addrs []net.IP) {
	s.dnsIPAddrsMu.Lock()
	defer s.dnsIPAddrsMu.Unlock()

	s.conf.dnsIPAddrs = slices.Clone(addrs)
}

// observedDNSIPAddrs converts observed IPv6 interface state into the DNS
// addresses returned by DHCPv6 replies.
func observedDNSIPAddrs(states []aghnet.IPv6AddrState) (addrs []net.IP) {
	for _, st := range states {
		if !st.Addr.IsValid() || !st.Addr.Is6() || st.Tentative {
			continue
		}
		if !st.Addr.IsLinkLocalUnicast() && st.PreferredLifetimeSec == 0 {
			continue
		}

		addrs = append(addrs, net.IP(st.Addr.AsSlice()))
	}

	switch len(addrs) {
	case 0:
		return nil
	case 1:
		return append(addrs, slices.Clone(addrs[0]))
	default:
		return addrs
	}
}

// advertisedLeasePrefixes returns the set of /64 prefixes currently
// advertised in Router Advertisements.
func advertisedLeasePrefixes(advertised []prefixPIO) (prefixes map[netip.Prefix]struct{}) {
	prefixes = make(map[netip.Prefix]struct{}, len(advertised))
	for _, p := range advertised {
		prefixes[p.Prefix.Masked()] = struct{}{}
	}

	return prefixes
}

// leasePrefixAdvertised reports whether ip belongs to one of the advertised
// /64 prefixes.
func leasePrefixAdvertised(prefixes map[netip.Prefix]struct{}, ip netip.Addr) (ok bool) {
	if !ip.Is6() {
		return false
	}

	_, ok = prefixes[netip.PrefixFrom(ip, raObservedPrefixBits).Masked()]

	return ok
}

// samePrefixSet reports whether a and b contain the same prefixes.
func samePrefixSet(a, b map[netip.Prefix]struct{}) (ok bool) {
	if len(a) != len(b) {
		return false
	}

	for pref := range a {
		if _, ok = b[pref]; !ok {
			return false
		}
	}

	return true
}

// prefixSetContainsAll reports whether haystack contains every prefix from
// needle.
func prefixSetContainsAll(haystack, needle map[netip.Prefix]struct{}) (ok bool) {
	for pref := range needle {
		if _, ok = haystack[pref]; !ok {
			return false
		}
	}

	return true
}

// renewableLeasePrefixes returns the set of /64 prefixes currently advertised
// with a non-zero preferred lifetime.
func renewableLeasePrefixes(advertised []prefixPIO) (prefixes map[netip.Prefix]struct{}) {
	prefixes = make(map[netip.Prefix]struct{}, len(advertised))
	for _, p := range advertised {
		if p.PreferredSec == 0 {
			continue
		}

		prefixes[p.Prefix.Masked()] = struct{}{}
	}

	return prefixes
}

// validUntilByPrefix returns the valid-lifetime deadlines for advertised /64
// prefixes.
func validUntilByPrefix(advertised []prefixPIO, now time.Time) (deadlines map[netip.Prefix]time.Time) {
	deadlines = make(map[netip.Prefix]time.Time, len(advertised))
	for _, p := range advertised {
		deadlines[p.Prefix.Masked()] = deadlineFromRemaining(now, p.ValidSec)
	}

	return deadlines
}

// refreshDeadlineMap updates absolute prefix deadlines while preserving
// existing deadlines when the remaining lifetime has not changed.
func refreshDeadlineMap(
	existing map[netip.Prefix]time.Time,
	advertised []prefixPIO,
	now time.Time,
	lifetime func(prefixPIO) uint32,
) (deadlines map[netip.Prefix]time.Time) {
	deadlines = make(map[netip.Prefix]time.Time, len(advertised))
	for _, p := range advertised {
		pref := p.Prefix.Masked()
		target := lifetime(p)
		if until, ok := existing[pref]; ok && remainingUntil(now, until) == target {
			deadlines[pref] = until

			continue
		}

		deadlines[pref] = deadlineFromRemaining(now, target)
	}

	return deadlines
}

// preferredUntilByPrefix returns the preferred-lifetime deadlines for
// advertised /64 prefixes.
func preferredUntilByPrefix(advertised []prefixPIO, now time.Time) (deadlines map[netip.Prefix]time.Time) {
	deadlines = make(map[netip.Prefix]time.Time, len(advertised))
	for _, p := range advertised {
		deadlines[p.Prefix.Masked()] = deadlineFromRemaining(now, p.PreferredSec)
	}

	return deadlines
}

// leasePrefixRenewable reports whether ip belongs to an advertised prefix with
// a non-zero preferred lifetime.
func leasePrefixRenewable(prefixes map[netip.Prefix]struct{}, ip netip.Addr) (ok bool) {
	return leasePrefixAdvertised(prefixes, ip)
}

// deprecatedMetaFrom returns the persisted deprecated-prefix metadata derived
// from the current tracked state.
func deprecatedMetaFrom(
	now time.Time,
	renewable map[netip.Prefix]struct{},
	advertised map[netip.Prefix]struct{},
	validUntil map[netip.Prefix]time.Time,
) (deprecated map[netip.Prefix]time.Time) {
	deprecated = map[netip.Prefix]time.Time{}
	for pref := range advertised {
		if _, ok := renewable[pref]; ok {
			continue
		}

		until, ok := validUntil[pref]
		if !ok || !until.After(now) {
			continue
		}

		deprecated[pref] = until
	}

	return deprecated
}

// sameDeadlineMap reports whether a and b contain the same deadlines.
func sameDeadlineMap(a, b map[netip.Prefix]time.Time) (ok bool) {
	if len(a) != len(b) {
		return false
	}

	for pref, until := range a {
		if other, ok := b[pref]; !ok || !other.Equal(until) {
			return false
		}
	}

	return true
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
		switch {
		case lease.IsStatic:
			lifetime = s.conf.leaseTime
		case s.leaseIsRenewable(lease.IP):
			lifetime = min(max(time.Until(lease.Expiry), 0), s.renewableLeaseLifetime(lease.IP))
		default:
			lifetime = s.deprecatedLeaseLifetime(lease.IP, lease.Expiry)
		}

	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:

		if !lease.IsStatic {
			if s.leaseIsRenewable(lease.IP) {
				lifetime = s.renewableLeaseLifetime(lease.IP)
				s.commitDynamicLease(lease, lifetime)
			} else {
				lifetime = s.deprecatedLeaseLifetime(lease.IP, lease.Expiry)
			}
		}
	}
	return lifetime
}

// requestedIP returns the IPv6 address requested by msg, if any.
func requestedIP(msg *dhcpv6.Message) (ip netip.Addr) {
	oia := msg.Options.OneIANA()
	if oia == nil {
		return netip.Addr{}
	}

	oiaAddr := oia.Options.OneAddress()
	if oiaAddr == nil {
		return netip.Addr{}
	}

	addr, ok := netip.AddrFromSlice(oiaAddr.IPv6Addr)
	if !ok {
		return netip.Addr{}
	}

	return addr
}

// findUsableLease returns a lease that should be served to mac for msg.
func (s *v6Server) findUsableLease(msg *dhcpv6.Message, mac net.HardwareAddr) (lease *dhcpsvc.Lease) {
	reqIP := requestedIP(msg)
	msgType := msg.Type()

	for _, l := range s.leases {
		if !bytes.Equal(mac, l.HWAddr) {
			continue
		}

		if l.IsStatic {
			if reqIP.IsValid() && l.IP == reqIP {
				return l
			} else if lease == nil {
				lease = l
			}

			continue
		}

		if reqIP.IsValid() && l.IP == reqIP {
			if (s.ipInCurrentPoolLocked(l.IP) && leaseNotExpired(l)) ||
				(canServeDeprecatedLease(msgType, l.IP, s.advertisedPrefixes) && leaseNotExpired(l)) {
				return l
			}
		} else if lease == nil && s.ipInCurrentPoolLocked(l.IP) && leaseNotExpired(l) {
			lease = l
		}
	}

	return lease
}

// canServeDeprecatedLease reports whether a deprecated dynamic lease for ip may
// still be served for msgType while its prefix remains advertised.
func canServeDeprecatedLease(
	msgType dhcpv6.MessageType,
	ip netip.Addr,
	advertisedPrefixes map[netip.Prefix]struct{},
) (ok bool) {
	switch msgType {
	case dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:
		return leasePrefixAdvertised(advertisedPrefixes, ip)
	default:
		return false
	}
}

// leaseIsRenewable reports whether a dynamic lease on ip should be refreshed
// with the full DHCP lifetime.
func (s *v6Server) leaseIsRenewable(ip netip.Addr) (ok bool) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	return leasePrefixRenewable(s.renewablePrefixes, ip)
}

// renewableLeaseLifetime returns the lifetime to grant to a renewable lease on
// ip.
func (s *v6Server) renewableLeaseLifetime(ip netip.Addr) (lifetime time.Duration) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	pref := netip.PrefixFrom(ip, raObservedPrefixBits).Masked()
	until, ok := s.validUntilByPrefix[pref]
	if !ok {
		return s.conf.leaseTime
	}

	validForPrefix := time.Duration(remainingUntil(time.Now(), until)) * time.Second

	return min(s.conf.leaseTime, validForPrefix)
}

// deprecatedLeaseLifetime returns the remaining valid lifetime for a
// non-renewable lease on ip, capped by the currently advertised prefix
// lifetime.
func (s *v6Server) deprecatedLeaseLifetime(ip netip.Addr, leaseExpiry time.Time) (lifetime time.Duration) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	pref := netip.PrefixFrom(ip, raObservedPrefixBits).Masked()
	until, ok := s.validUntilByPrefix[pref]
	if !ok {
		return 0
	}

	validForPrefix := time.Duration(remainingUntil(time.Now(), until)) * time.Second
	validForLease := max(time.Until(leaseExpiry), 0)

	return min(validForLease, validForPrefix)
}

// preferredLeaseLifetime returns the preferred lifetime to encode in DHCPv6
// replies for lease.
func (s *v6Server) preferredLeaseLifetime(lease *dhcpsvc.Lease, validLifetime time.Duration) time.Duration {
	if !lease.IsStatic && !s.leaseIsRenewable(lease.IP) {
		return 0
	}

	if lease.IsStatic {
		return validLifetime
	}

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	pref := netip.PrefixFrom(lease.IP, raObservedPrefixBits).Masked()
	until, ok := s.preferredUntilByPrefix[pref]
	if !ok {
		return validLifetime
	}

	preferredForPrefix := time.Duration(remainingUntil(time.Now(), until)) * time.Second

	return min(validLifetime, preferredForPrefix)
}

// leaseNotExpired reports whether lease is still valid at the time of the
// request.  Zero expiries are treated as not yet committed.
func leaseNotExpired(lease *dhcpsvc.Lease) (ok bool) {
	return lease.Expiry.IsZero() || lease.Expiry.After(time.Now())
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

		lease = s.findUsableLease(msg, mac)
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
	preferredLifetime := s.preferredLeaseLifetime(lease, lifetime)

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
		PreferredLifetime: preferredLifetime,
		ValidLifetime:     lifetime,
	}
	oia.Options = dhcpv6.IdentityOptions{
		Options: []dhcpv6.Option{oiaAddr},
	}
	resp.AddOption(oia)

	if msg.IsOptionRequested(dhcpv6.OptionDNSRecursiveNameServer) {
		resp.UpdateOption(dhcpv6.OptDNS(s.dnsIPAddrs()...))
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

	if !s.process(msg, req, resp) {
		if code, text, ok := replyStatusForProcessFailure(msg.Type()); ok {
			resp.AddOption(&dhcpv6.OptStatusCode{
				StatusCode:    code,
				StatusMessage: text,
			})
		} else if requiresProcessSuccess(msg.Type()) {
			return
		}
	}

	log.Debug("dhcpv6: sending: %s", resp.Summary())

	_, err = conn.WriteTo(resp.ToBytes(), peer)
	if err != nil {
		log.Error("dhcpv6: conn.Write to %s failed: %s", peer, err)

		return
	}
}

// requiresProcessSuccess reports whether msgType requires a usable lease
// before it is safe to send a reply.
func requiresProcessSuccess(msgType dhcpv6.MessageType) (ok bool) {
	switch msgType {
	case dhcpv6.MessageTypeSolicit,
		dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:
		return true
	default:
		return false
	}
}

// replyStatusForProcessFailure maps a failed lease-processing path to the
// DHCPv6 status code a client can use to recover.
func replyStatusForProcessFailure(msgType dhcpv6.MessageType) (code iana.StatusCode, text string, ok bool) {
	switch msgType {
	case dhcpv6.MessageTypeSolicit:
		return iana.StatusNoAddrsAvail, iana.StatusNoAddrsAvail.String(), true
	case dhcpv6.MessageTypeConfirm:
		return iana.StatusNotOnLink, iana.StatusNotOnLink.String(), true
	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind:
		return iana.StatusNoBinding, iana.StatusNoBinding.String(), true
	default:
		return 0, "", false
	}
}

// configureDNSIPAddrs updates v6Server configuration with the slice of DNS IP
// addresses of provided interface iface.
func (s *v6Server) configureDNSIPAddrs(
	ctx context.Context,
	iface *net.Interface,
) (ok bool, err error) {
	dnsIPAddrs, err := aghnet.IfaceDNSIPAddrs(
		ctx,
		s.conf.Logger,
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

	s.setDNSIPAddrs(dnsIPAddrs)

	return true, nil
}

// initRA initializes the Router Advertisement state loop.
func (s *v6Server) initRA(
	iface *net.Interface,
	initial raState,
	observe raObserver,
) (err error) {
	s.ra.raAllowSLAAC = s.conf.RAAllowSLAAC
	s.ra.raSLAACOnly = s.conf.RASLAACOnly
	s.ra.observe = observe
	s.ra.ifaceName = s.conf.InterfaceName
	s.ra.iface = iface
	s.ra.packetSendPeriod = 1 * time.Second
	s.ra.observePeriod = defaultRAObservePeriod

	return s.ra.Init(initial)
}

// observeRAState refreshes the current interface-derived Router Advertisement
// observation.
func (s *v6Server) observeRAState(ctx context.Context) (obs raObservation, err error) {
	states, err := aghnet.ObserveIPv6Addrs(
		ctx,
		s.conf.Logger,
		s.conf.CommandConstructor,
		s.conf.InterfaceName,
	)
	if err != nil {
		return raObservation{}, err
	}

	s.setDNSIPAddrs(observedDNSIPAddrs(states))

	obs = buildInterfaceRAObservation(states)
	if !obs.SourceAddr.IsValid() {
		obs.SourceAddr = pickStaticRASourceAddr(s.dnsIPAddrs())
		obs.RDNSSAddr = obs.SourceAddr
	}

	return obs, nil
}

// trackedPrefixChanged updates the effective DHCPv6 pool start from active.
func (s *v6Server) trackedPrefixChanged(active *raPrefixSnapshot, advertised []prefixPIO) (err error) {
	if !s.conf.NeedsDHCPv6Pool() {
		s.setTrackedRangeStart(nil, advertised)

		return nil
	}

	if active == nil {
		s.setTrackedRangeStart(nil, advertised)

		return nil
	}

	ipStart, err := deriveTrackedRangeStart(s.conf.RangeStart, active.Prefix)
	if err != nil {
		return err
	}

	s.setTrackedRangeStart(ipStart, advertised)

	return nil
}

// setTrackedRangeStart updates the effective DHCPv6 pool start and removes
// dynamic leases whose prefixes are no longer advertised.
func (s *v6Server) setTrackedRangeStart(ipStart net.IP, advertised []prefixPIO) {
	s.leasesLock.Lock()

	now := time.Now()
	oldDeprecated := deprecatedMetaFrom(
		now,
		s.renewablePrefixes,
		s.advertisedPrefixes,
		s.validUntilByPrefix,
	)
	oldRenewable := maps.Clone(s.renewablePrefixes)
	keepPrefixes := advertisedLeasePrefixes(advertised)
	renewable := renewableLeasePrefixes(advertised)
	preferredUntil := refreshDeadlineMap(s.preferredUntilByPrefix, advertised, now, func(p prefixPIO) uint32 {
		return p.PreferredSec
	})
	validUntil := refreshDeadlineMap(s.validUntilByPrefix, advertised, now, func(p prefixPIO) uint32 {
		return p.ValidSec
	})
	s.conf.ipStart = bytes.Clone(ipStart)
	s.advertisedPrefixes = keepPrefixes
	s.renewablePrefixes = renewable
	s.preferredUntilByPrefix = preferredUntil
	s.validUntilByPrefix = validUntil
	s.ipAddrs = [256]byte{}

	activePrefix := netip.Prefix{}
	if len(ipStart) == net.IPv6len {
		if addr, ok := netip.AddrFromSlice(ipStart); ok {
			activePrefix = netip.PrefixFrom(addr, raObservedPrefixBits).Masked()
		}
	}

	removed := 0
	updated := false
	leases := s.leases[:0]
	for _, l := range s.leases {
		if !l.IsStatic {
			pref := netip.PrefixFrom(l.IP, raObservedPrefixBits).Masked()
			if !leasePrefixAdvertised(keepPrefixes, l.IP) ||
				(activePrefix.IsValid() && pref == activePrefix && !ip6InRange(ipStart, net.IP(l.IP.AsSlice()))) {
				removed++

				continue
			}

			if until, ok := validUntil[pref]; ok && (l.Expiry.IsZero() || l.Expiry.After(until)) {
				l.Expiry = until
				updated = true
			}
		}

		leases = append(leases, l)
		s.markLeaseOccupied(l)
	}

	s.leases = leases
	newDeprecated := deprecatedMetaFrom(now, renewable, keepPrefixes, validUntil)
	metadataChanged := (len(oldDeprecated) > 0 || len(newDeprecated) > 0) &&
		(!samePrefixSet(oldRenewable, renewable) || !sameDeadlineMap(oldDeprecated, newDeprecated))
	s.leasesLock.Unlock()

	if (removed > 0 || updated || metadataChanged) && s.conf.notify != nil {
		s.conf.notify(LeaseChangedDBStore)
	}
}

// deriveTrackedRangeStart returns the effective DHCPv6 pool start for the
// current observed /64 prefix while preserving the configured host bits from
// template.
func deriveTrackedRangeStart(template net.IP, observedPrefix netip.Prefix) (ipStart net.IP, err error) {
	if template == nil || template.To16() == nil {
		return nil, fmt.Errorf("invalid range-start IP: %s", template)
	}
	if !observedPrefix.IsValid() || observedPrefix.Bits() != raObservedPrefixBits {
		return nil, fmt.Errorf("invalid observed prefix: %s", observedPrefix)
	}

	addr := observedPrefix.Masked().Addr().As16()
	ipStart = bytes.Clone(template.To16())
	copy(ipStart[:8], addr[:8])

	return ipStart, nil
}

// hasStaticV6Leases reports whether the current lease set contains IPv6 static
// leases.
func (s *v6Server) hasStaticV6Leases() (ok bool) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if l.IsStatic && l.IP.Is6() {
			return true
		}
	}

	return false
}

// restoreDeprecatedPrefixes seeds initial deprecated prefixes from persisted
// metadata whose renewable prefixes still match the currently observed
// interface state.
func (s *v6Server) restoreDeprecatedPrefixes(now time.Time, st *raState) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	s.persistRestoredMeta = false

	if len(s.restoredDeprecated) == 0 {
		return
	}

	observedRenewable := renewableLeasePrefixes(st.pios(now))
	if len(s.restoredRenewable) > 0 {
		if !prefixSetContainsAll(observedRenewable, s.restoredRenewable) {
			return
		}
	} else if len(observedRenewable) > 0 {
		return
	} else if len(st.pios(now)) == 0 {
		return
	}

	advertised := advertisedLeasePrefixes(st.pios(now))
	for pref, until := range s.restoredDeprecated {
		if _, ok := advertised[pref]; ok {
			continue
		}

		if !until.After(now) {
			continue
		}

		valid := uint32(until.Sub(now) / time.Second)
		if valid > raDeprecatedLifetimeCapSecs {
			valid = raDeprecatedLifetimeCapSecs
		}

		st.deprecated[pref] = newTrackedPrefix(raPrefixSnapshot{
			Prefix:       pref,
			PreferredSec: 0,
			ValidSec:     valid,
		}, raPrefixOriginDeprecated, now)
	}
}

// setRestoredPrefixMeta stores deprecated-prefix metadata loaded from disk.
func (s *v6Server) setRestoredPrefixMeta(
	renewable map[netip.Prefix]struct{},
	deprecated map[netip.Prefix]time.Time,
) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	s.restoredRenewable = maps.Clone(renewable)
	s.restoredDeprecated = maps.Clone(deprecated)
	s.persistRestoredMeta = len(s.restoredRenewable) > 0 || len(s.restoredDeprecated) > 0
}

// deprecatedPrefixMeta returns persisted metadata for currently tracked
// interface-derived prefixes.
func (s *v6Server) deprecatedPrefixMeta(now time.Time) (renewable map[netip.Prefix]struct{}, deprecated map[netip.Prefix]time.Time) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	return s.deprecatedPrefixMetaLocked(now)
}

// deprecatedPrefixMetaLocked returns persisted metadata for currently tracked
// interface-derived prefixes.  s.leasesLock must be held.
func (s *v6Server) deprecatedPrefixMetaLocked(now time.Time) (renewable map[netip.Prefix]struct{}, deprecated map[netip.Prefix]time.Time) {
	if s.conf.NormalizedPrefixSource() != V6PrefixSourceInterface {
		return nil, nil
	}

	renewable = maps.Clone(s.renewablePrefixes)
	deprecated = map[netip.Prefix]time.Time{}
	for pref := range s.advertisedPrefixes {
		if _, ok := s.renewablePrefixes[pref]; ok {
			continue
		}

		until, ok := s.validUntilByPrefix[pref]
		if !ok || !until.After(now) {
			continue
		}

		deprecated[pref] = until
	}

	if s.persistRestoredMeta && len(renewable) == 0 && len(deprecated) == 0 {
		return maps.Clone(s.restoredRenewable), maps.Clone(s.restoredDeprecated)
	}

	return renewable, deprecated
}

// Start starts the IPv6 DHCP server.
func (s *v6Server) Start(ctx context.Context) (err error) {
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

	ok, err := s.configureDNSIPAddrs(ctx, iface)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if !ok {
		if s.conf.NormalizedPrefixSource() != V6PrefixSourceInterface {
			// No available IP addresses which may appear later.
			return nil
		}
	}

	var (
		initial raState
		observe raObserver
	)

	switch s.conf.NormalizedPrefixSource() {
	case V6PrefixSourceStatic:
		initial = newStaticRAState(buildStaticRAObservation(s.dnsIPAddrs(), s.conf.ipStart))
		s.ra.onStateRefresh = nil
		s.ra.onActivePrefixChange = nil
	case V6PrefixSourceInterface:
		if s.hasStaticV6Leases() {
			s.conf.Logger.WarnContext(
				ctx,
				"dhcpv6: interface-derived prefix tracking does not rewrite literal static IPv6 leases",
			)
		}

		initial = newObservedRAState()

		obs, obsErr := s.observeRAState(ctx)
		if obsErr != nil {
			return fmt.Errorf("observing initial ipv6 prefix state: %w", obsErr)
		} else {
			now := time.Now()
			initial.merge(obs, now)
			s.restoreDeprecatedPrefixes(now, &initial)
			if pios := initial.pios(now); len(pios) > 0 {
				if err = s.trackedPrefixChanged(initial.activeSnapshot(now), pios); err != nil {
					return fmt.Errorf("updating tracked range start: %w", err)
				}
			}
		}

		observe = s.observeRAState
		s.ra.onStateRefresh = func(now time.Time, st *raState) {
			s.restoreDeprecatedPrefixes(now, st)
		}
		s.ra.onActivePrefixChange = func(active *raPrefixSnapshot, advertised []prefixPIO) {
			if activeErr := s.trackedPrefixChanged(active, advertised); activeErr != nil {
				log.Error("dhcpv6: updating tracked pool: %s", activeErr)
			}
		}
	default:
		return fmt.Errorf("unsupported prefix source %q", s.conf.PrefixSource)
	}

	err = s.initRA(iface, initial, observe)
	if err != nil {
		return err
	}

	// Don't initialize DHCPv6 server if we must force the clients to use SLAAC.
	if !s.conf.NeedsDHCPv6Pool() {
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
		return nil
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
	conf.PrefixSource = conf.NormalizedPrefixSource()
	s.conf = conf

	err := conf.ValidatePrefixSource()
	if err != nil {
		return s, fmt.Errorf("dhcpv6: %w", err)
	}

	if !conf.Enabled {
		return s, nil
	}

	needsConfiguredRange := conf.NormalizedPrefixSource() == V6PrefixSourceStatic || conf.NeedsDHCPv6Pool()
	if needsConfiguredRange && (conf.RangeStart == nil || conf.RangeStart.To16() == nil) {
		return s, fmt.Errorf("dhcpv6: invalid range-start IP: %s", conf.RangeStart)
	}
	if len(conf.RangeStart) != 0 {
		if conf.RangeStart.To16() == nil {
			return s, fmt.Errorf("dhcpv6: invalid range-start IP: %s", conf.RangeStart)
		}

		s.conf.RangeStart = bytes.Clone(conf.RangeStart.To16())
	}

	if conf.NormalizedPrefixSource() == V6PrefixSourceStatic {
		s.conf.ipStart = bytes.Clone(s.conf.RangeStart)
		if addr, ok := netip.AddrFromSlice(s.conf.ipStart); ok {
			prefix := netip.PrefixFrom(addr, raObservedPrefixBits).Masked()
			s.advertisedPrefixes = map[netip.Prefix]struct{}{prefix: {}}
			s.renewablePrefixes = map[netip.Prefix]struct{}{prefix: {}}
		}
	}

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = timeutil.Day
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s, nil
}
