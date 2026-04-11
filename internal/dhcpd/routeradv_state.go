package dhcpd

import (
	"cmp"
	"math"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
)

const (
	defaultRAPrefixLifetimeSeconds = uint32(3600)
	defaultRARDNSSLifetimeSeconds  = uint32(3600)

	raObservedPrefixBits        = 64
	raDeprecatedLifetimeCap     = 2 * time.Hour
	raDeprecatedLifetimeCapSecs = uint32(raDeprecatedLifetimeCap / time.Second)
)

// prefixPIO describes the Prefix Information Option to encode into an RA
// packet.
type prefixPIO struct {
	Prefix       netip.Prefix
	PreferredSec uint32
	ValidSec     uint32
}

// raPrefixSnapshot is the observed state of one advertised IPv6 prefix.
type raPrefixSnapshot struct {
	Prefix       netip.Prefix
	PreferredSec uint32
	ValidSec     uint32
}

// raObservation is the current Router Advertisement input derived from either
// a static configuration or interface state.
type raObservation struct {
	SourceAddr netip.Addr
	RDNSSAddr  netip.Addr
	Active     *raPrefixSnapshot
	Inactive   []raPrefixSnapshot
}

// raPrefixOrigin describes where a tracked prefix came from.
type raPrefixOrigin uint8

// raPrefixOrigin values.
const (
	raPrefixOriginStaticConfigured raPrefixOrigin = iota
	raPrefixOriginObservedActive
	raPrefixOriginObservedInactive
	raPrefixOriginDeprecated
)

// trackedPrefix keeps either a fixed-lifetime configured prefix or a prefix
// whose remaining lifetimes count down from observed values.
type trackedPrefix struct {
	prefix netip.Prefix
	origin raPrefixOrigin

	fixedPreferredSec uint32
	fixedValidSec     uint32

	preferredUntil time.Time
	validUntil     time.Time
}

// raState is the current runtime Router Advertisement state.
type raState struct {
	sourceAddr netip.Addr
	rdnssAddr  netip.Addr
	active     *trackedPrefix
	deprecated map[netip.Prefix]*trackedPrefix
}

// raActiveChange describes an active-prefix transition.
type raActiveChange struct {
	Changed bool
	Active  *raPrefixSnapshot
}

// newObservedRAState returns a new empty RA state for interface-derived
// observations.
func newObservedRAState() (st raState) {
	return raState{
		deprecated: map[netip.Prefix]*trackedPrefix{},
	}
}

// newStaticRAState returns a new state keeping the current fixed-lifetime
// semantics for the configured static prefix.
func newStaticRAState(obs raObservation) (st raState) {
	st = newObservedRAState()
	st.sourceAddr = obs.SourceAddr
	st.rdnssAddr = obs.RDNSSAddr
	if obs.Active != nil {
		st.active = newTrackedPrefix(*obs.Active, raPrefixOriginStaticConfigured, time.Time{})
	}

	return st
}

// merge merges a fresh interface observation into s and reports whether the
// active prefix changed.
func (s *raState) merge(obs raObservation, now time.Time) (change raActiveChange) {
	if s.deprecated == nil {
		s.deprecated = map[netip.Prefix]*trackedPrefix{}
	}

	prev := s.activeSnapshot(now)
	prevActivePrefix := netip.Prefix{}
	activeChanged := false
	if prev != nil {
		prevActivePrefix = prev.Prefix
	}

	s.sourceAddr = obs.SourceAddr
	s.rdnssAddr = obs.RDNSSAddr

	switch {
	case obs.Active != nil && s.active != nil && s.active.prefix == obs.Active.Prefix:
		s.active = newTrackedPrefix(*obs.Active, raPrefixOriginObservedActive, now)
	case obs.Active != nil:
		s.moveActiveToDeprecated(now)
		s.active = newTrackedPrefix(*obs.Active, raPrefixOriginObservedActive, now)
		activeChanged = prev != nil && prev.Prefix != obs.Active.Prefix
	case obs.Active == nil:
		s.moveActiveToDeprecated(now)
		s.active = nil
		activeChanged = prev != nil
	}

	if s.active != nil {
		delete(s.deprecated, s.active.prefix)
	}

	observedInactive := map[netip.Prefix]struct{}{}

	for _, dep := range obs.Inactive {
		if s.active != nil && dep.Prefix == s.active.prefix {
			continue
		}
		if activeChanged && dep.Prefix == prevActivePrefix && dep.PreferredSec == 0 {
			valid := dep.ValidSec
			if valid > raDeprecatedLifetimeCapSecs || valid == math.MaxUint32 {
				valid = raDeprecatedLifetimeCapSecs
			}
			if valid == 0 {
				delete(s.deprecated, dep.Prefix)

				continue
			}

			s.deprecated[dep.Prefix] = newTrackedPrefix(raPrefixSnapshot{
				Prefix:       dep.Prefix,
				PreferredSec: 0,
				ValidSec:     valid,
			}, raPrefixOriginDeprecated, now)

			continue
		}

		if dep.PreferredSec > 0 {
			observedInactive[dep.Prefix] = struct{}{}
			s.deprecated[dep.Prefix] = newTrackedPrefix(dep, raPrefixOriginObservedInactive, now)

			continue
		}

		valid := dep.ValidSec
		if valid > raDeprecatedLifetimeCapSecs || valid == math.MaxUint32 {
			valid = raDeprecatedLifetimeCapSecs
		}
		if valid == 0 {
			delete(s.deprecated, dep.Prefix)

			continue
		}

		s.deprecated[dep.Prefix] = newTrackedPrefix(raPrefixSnapshot{
			Prefix:       dep.Prefix,
			PreferredSec: 0,
			ValidSec:     valid,
		}, raPrefixOriginDeprecated, now)
	}

	for pref, tracked := range s.deprecated {
		if tracked.origin != raPrefixOriginObservedInactive {
			continue
		}

		if _, ok := observedInactive[pref]; !ok {
			s.deprecateTrackedPrefix(pref, tracked, now)
		}
	}

	s.evictExpired(now)

	next := s.activeSnapshot(now)
	change.Changed = !sameActivePrefix(prev, next)
	change.Active = next

	return change
}

// deprecateTrackedPrefix converts a tracked prefix into a deprecated one using
// its remaining valid lifetime bounded by the standard two-hour cap.
func (s *raState) deprecateTrackedPrefix(pref netip.Prefix, tracked *trackedPrefix, now time.Time) {
	_, valid, expired := tracked.remaining(now)
	if expired || valid == 0 {
		delete(s.deprecated, pref)

		return
	}
	if valid > raDeprecatedLifetimeCapSecs || valid == math.MaxUint32 {
		valid = raDeprecatedLifetimeCapSecs
	}

	s.deprecated[pref] = newTrackedPrefix(raPrefixSnapshot{
		Prefix:       pref,
		PreferredSec: 0,
		ValidSec:     valid,
	}, raPrefixOriginDeprecated, now)
}

// sourceAndRDNSS returns the addresses currently used for RA and RDNSS.
func (s *raState) sourceAndRDNSS() (source, rdnss netip.Addr) {
	return s.sourceAddr, s.rdnssAddr
}

// activeSnapshot returns the active prefix snapshot at now.
func (s *raState) activeSnapshot(now time.Time) (snap *raPrefixSnapshot) {
	if s.active == nil {
		return nil
	}

	return s.active.snapshot(now)
}

// pios returns the currently advertised prefix options in the correct order.
func (s *raState) pios(now time.Time) (pios []prefixPIO) {
	s.evictExpired(now)

	if active := s.active.snapshot(now); active != nil {
		pios = append(pios, prefixPIO{
			Prefix:       active.Prefix,
			PreferredSec: active.PreferredSec,
			ValidSec:     active.ValidSec,
		})
	}

	deprecated := make([]prefixPIO, 0, len(s.deprecated))
	for _, p := range s.deprecated {
		snap := p.snapshot(now)
		if snap == nil {
			continue
		}

		deprecated = append(deprecated, prefixPIO{
			Prefix:       snap.Prefix,
			PreferredSec: snap.PreferredSec,
			ValidSec:     snap.ValidSec,
		})
	}

	slices.SortFunc(deprecated, func(a, b prefixPIO) int {
		return prefixCompare(a.Prefix, b.Prefix)
	})

	return append(pios, deprecated...)
}

// buildInterfaceRAObservation selects the current RA source address and active
// and deprecated prefixes from raw interface address state.
func buildInterfaceRAObservation(states []aghnet.IPv6AddrState) (obs raObservation) {
	obs.SourceAddr = pickRASourceAddr(states)
	obs.RDNSSAddr = obs.SourceAddr

	grouped := map[netip.Prefix][]aghnet.IPv6AddrState{}
	for _, st := range states {
		if !isEligibleRAPrefixState(st) {
			continue
		}

		pref := st.Prefix.Masked()
		grouped[pref] = append(grouped[pref], st)
	}

	prefixes := make([]raPrefixSnapshot, 0, len(grouped))
	for pref, group := range grouped {
		snap, ok := collapsePrefixGroup(pref, group)
		if !ok {
			continue
		}

		prefixes = append(prefixes, snap)
	}

	activeIdx := -1
	for i, pref := range prefixes {
		if pref.PreferredSec == 0 {
			continue
		}

		if activeIdx == -1 || betterActivePrefix(pref, prefixes[activeIdx]) {
			activeIdx = i
		}
	}

	for i, pref := range prefixes {
		switch {
		case i == activeIdx:
			obs.Active = &raPrefixSnapshot{
				Prefix:       pref.Prefix,
				PreferredSec: pref.PreferredSec,
				ValidSec:     pref.ValidSec,
			}
		default:
			obs.Inactive = append(obs.Inactive, pref)
		}
	}

	slices.SortFunc(obs.Inactive, func(a, b raPrefixSnapshot) int {
		return prefixCompare(a.Prefix, b.Prefix)
	})

	return obs
}

// buildStaticRAObservation returns the configured static prefix observation.
func buildStaticRAObservation(dnsIPAddrs []net.IP, rangeStart net.IP) (obs raObservation) {
	addr := pickStaticRASourceAddr(dnsIPAddrs)
	obs.SourceAddr = addr
	obs.RDNSSAddr = addr

	prefixAddr, ok := netip.AddrFromSlice(rangeStart)
	if !ok {
		return obs
	}

	obs.Active = &raPrefixSnapshot{
		Prefix:       netip.PrefixFrom(prefixAddr, raObservedPrefixBits).Masked(),
		PreferredSec: defaultRAPrefixLifetimeSeconds,
		ValidSec:     defaultRAPrefixLifetimeSeconds,
	}

	return obs
}

// pickStaticRASourceAddr selects the source/RDNSS address from the interface
// address list returned by IfaceDNSIPAddrs.
func pickStaticRASourceAddr(addrs []net.IP) (addr netip.Addr) {
	var fallback netip.Addr
	for _, ip := range addrs {
		a, ok := netip.AddrFromSlice(ip)
		if !ok || !a.Is6() {
			continue
		}

		a = a.WithZone("")
		if a.IsLinkLocalUnicast() {
			return a
		} else if !fallback.IsValid() {
			fallback = a
		}
	}

	return fallback
}

// pickRASourceAddr chooses the preferred RA source address from interface
// observations.
func pickRASourceAddr(states []aghnet.IPv6AddrState) (addr netip.Addr) {
	var linkLocal []netip.Addr
	var fallback []netip.Addr

	for _, st := range states {
		if !st.Addr.IsValid() || !st.Addr.Is6() || st.Tentative {
			continue
		}

		addr = st.Addr.WithZone("")
		if addr.IsLinkLocalUnicast() {
			linkLocal = append(linkLocal, addr)
		} else if !st.Temporary {
			fallback = append(fallback, addr)
		}
	}

	slices.SortFunc(linkLocal, func(a, b netip.Addr) int { return a.Compare(b) })
	slices.SortFunc(fallback, func(a, b netip.Addr) int { return a.Compare(b) })

	switch {
	case len(linkLocal) > 0:
		return linkLocal[0]
	case len(fallback) > 0:
		return fallback[0]
	default:
		return netip.Addr{}
	}
}

// isEligibleRAPrefixState reports whether st may be used to derive a Prefix
// Information Option.
func isEligibleRAPrefixState(st aghnet.IPv6AddrState) (ok bool) {
	switch {
	case !st.Addr.IsValid(),
		!st.Addr.Is6(),
		!st.Addr.IsGlobalUnicast(),
		st.Tentative,
		st.ValidLifetimeSec == 0,
		!st.Prefix.IsValid(),
		st.Prefix.Bits() != raObservedPrefixBits:
		return false
	default:
		return true
	}
}

// collapsePrefixGroup collapses multiple addresses belonging to the same
// prefix into one snapshot by taking the longest remaining preferred and valid
// lifetimes observed for that /64.
func collapsePrefixGroup(
	prefix netip.Prefix,
	group []aghnet.IPv6AddrState,
) (snap raPrefixSnapshot, ok bool) {
	if len(group) == 0 {
		return raPrefixSnapshot{}, false
	}

	preferredSec := group[0].PreferredLifetimeSec
	validSec := group[0].ValidLifetimeSec
	for _, st := range group[1:] {
		preferredSec = max(preferredSec, st.PreferredLifetimeSec)
		validSec = max(validSec, st.ValidLifetimeSec)
	}

	return raPrefixSnapshot{
		Prefix:       prefix,
		PreferredSec: preferredSec,
		ValidSec:     validSec,
	}, true
}

// betterActivePrefix reports whether a should be preferred over b as the
// active interface-derived prefix.
func betterActivePrefix(a, b raPrefixSnapshot) (ok bool) {
	aFinite := a.PreferredSec != math.MaxUint32
	bFinite := b.PreferredSec != math.MaxUint32
	switch {
	case aFinite != bFinite:
		return aFinite
	case a.PreferredSec != b.PreferredSec:
		return a.PreferredSec > b.PreferredSec
	case a.ValidSec != b.ValidSec:
		return a.ValidSec > b.ValidSec
	default:
		return prefixCompare(a.Prefix, b.Prefix) < 0
	}
}

// newTrackedPrefix returns a tracked prefix built from snap.
func newTrackedPrefix(
	snap raPrefixSnapshot,
	origin raPrefixOrigin,
	now time.Time,
) (tracked *trackedPrefix) {
	tracked = &trackedPrefix{
		prefix: snap.Prefix,
		origin: origin,
	}

	switch origin {
	case raPrefixOriginStaticConfigured:
		tracked.fixedPreferredSec = snap.PreferredSec
		tracked.fixedValidSec = snap.ValidSec
	default:
		tracked.preferredUntil = deadlineFromRemaining(now, snap.PreferredSec)
		tracked.validUntil = deadlineFromRemaining(now, snap.ValidSec)
	}

	return tracked
}

// snapshot returns the current remaining lifetimes for p.
func (p *trackedPrefix) snapshot(now time.Time) (snap *raPrefixSnapshot) {
	if p == nil {
		return nil
	}

	preferred, valid, expired := p.remaining(now)
	if expired {
		return nil
	}

	return &raPrefixSnapshot{
		Prefix:       p.prefix,
		PreferredSec: preferred,
		ValidSec:     valid,
	}
}

// remaining returns the current remaining lifetimes for p.
func (p *trackedPrefix) remaining(now time.Time) (preferred, valid uint32, expired bool) {
	if p.origin == raPrefixOriginStaticConfigured {
		if p.fixedValidSec == 0 {
			return 0, 0, true
		}

		return p.fixedPreferredSec, p.fixedValidSec, false
	}

	preferred = remainingUntil(now, p.preferredUntil)
	valid = remainingUntil(now, p.validUntil)

	return preferred, valid, valid == 0
}

// moveActiveToDeprecated converts the current active prefix into a deprecated
// one bounded by RFC 4862's two-hour valid lifetime rule.
func (s *raState) moveActiveToDeprecated(now time.Time) {
	if s.active == nil {
		return
	}

	_, valid, expired := s.active.remaining(now)
	if expired {
		return
	}

	if valid > raDeprecatedLifetimeCapSecs || valid == math.MaxUint32 {
		valid = raDeprecatedLifetimeCapSecs
	}
	if valid == 0 {
		return
	}

	s.deprecated[s.active.prefix] = newTrackedPrefix(raPrefixSnapshot{
		Prefix:       s.active.prefix,
		PreferredSec: 0,
		ValidSec:     valid,
	}, raPrefixOriginDeprecated, now)
}

// evictExpired removes prefixes whose valid lifetimes have expired.
func (s *raState) evictExpired(now time.Time) {
	if s.active != nil {
		if _, _, expired := s.active.remaining(now); expired {
			s.active = nil
		}
	}

	for pref, dep := range s.deprecated {
		if _, _, expired := dep.remaining(now); expired {
			delete(s.deprecated, pref)
		}
	}
}

// deadlineFromRemaining returns the deadline corresponding to the remaining
// lifetime.
func deadlineFromRemaining(now time.Time, sec uint32) (deadline time.Time) {
	switch sec {
	case 0:
		return now
	case math.MaxUint32:
		return time.Time{}
	default:
		return now.Add(time.Duration(sec) * time.Second)
	}
}

// remainingUntil returns the remaining lifetime until deadline.
func remainingUntil(now, deadline time.Time) (sec uint32) {
	switch {
	case deadline.IsZero():
		return math.MaxUint32
	case !deadline.After(now):
		return 0
	default:
		return uint32(deadline.Sub(now) / time.Second)
	}
}

// prefixCompare lexically compares IPv6 prefixes.
func prefixCompare(a, b netip.Prefix) (res int) {
	if res = a.Addr().Compare(b.Addr()); res != 0 {
		return res
	}

	return cmp.Compare(a.Bits(), b.Bits())
}

// sameActivePrefix reports whether a and b describe the same currently active
// prefix.
func sameActivePrefix(a, b *raPrefixSnapshot) (ok bool) {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Prefix == b.Prefix
	}
}
