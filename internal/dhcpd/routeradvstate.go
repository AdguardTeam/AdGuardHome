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

// raStateDigest is a lifetime-agnostic fingerprint of raState used to detect
// changes that are not just the ordinary countdown of elapsed time.  Two
// observations that produce equal digests describe the same kernel state and
// should not trigger downstream reconciliation.
type raStateDigest struct {
	sourceAddr netip.Addr
	rdnssAddr  netip.Addr
	active     *trackedPrefixDigest
	deprecated map[netip.Prefix]trackedPrefixDigest
}

// trackedPrefixDigest is the deadline-based fingerprint of one trackedPrefix.
//
// preferredExpired is a time-derived boolean that flips from false to true
// exactly once during a natural countdown, the moment the prefix's preferred
// lifetime reaches zero.  Without it, a prefix whose preferred lifetime
// counts down while its absolute preferredUntil deadline stays put would
// produce identical digests before and after it becomes non-renewable, so
// downstream reconciliation, which rebuilds the v6Server renewablePrefixes
// set, would never fire for that transition.
type trackedPrefixDigest struct {
	prefix            netip.Prefix
	origin            raPrefixOrigin
	fixedPreferredSec uint32
	fixedValidSec     uint32
	preferredUntil    time.Time
	validUntil        time.Time
	preferredExpired  bool
}

// digestTrackedPrefix returns the digest for tp at now.
func digestTrackedPrefix(tp *trackedPrefix, now time.Time) trackedPrefixDigest {
	preferred, _, _ := tp.remaining(now)

	return trackedPrefixDigest{
		prefix:            tp.prefix,
		origin:            tp.origin,
		fixedPreferredSec: tp.fixedPreferredSec,
		fixedValidSec:     tp.fixedValidSec,
		preferredUntil:    tp.preferredUntil,
		validUntil:        tp.validUntil,
		preferredExpired:  preferred == 0,
	}
}

// digest returns the current state digest after evicting prefixes whose
// lifetimes have already expired at now.
func (s *raState) digest(now time.Time) (d raStateDigest) {
	s.evictExpired(now)

	d.sourceAddr = s.sourceAddr
	d.rdnssAddr = s.rdnssAddr
	if s.active != nil {
		tmp := digestTrackedPrefix(s.active, now)
		d.active = &tmp
	}

	d.deprecated = make(map[netip.Prefix]trackedPrefixDigest, len(s.deprecated))
	for pref, tp := range s.deprecated {
		d.deprecated[pref] = digestTrackedPrefix(tp, now)
	}

	return d
}

// sameRAStateDigest reports whether a and b describe the same state.
func sameRAStateDigest(a, b raStateDigest) (ok bool) {
	if a.sourceAddr != b.sourceAddr || a.rdnssAddr != b.rdnssAddr {
		return false
	}

	if (a.active == nil) != (b.active == nil) {
		return false
	}

	if a.active != nil && *a.active != *b.active {
		return false
	}

	if len(a.deprecated) != len(b.deprecated) {
		return false
	}

	for pref, digest := range a.deprecated {
		other, found := b.deprecated[pref]
		if !found || other != digest {
			return false
		}
	}

	return true
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

// capDeprecatedLifetime bounds a remaining valid lifetime by the standard
// two-hour cap used for deprecated prefixes.
func capDeprecatedLifetime(valid uint32) (capped uint32) {
	if valid > raDeprecatedLifetimeCapSecs || valid == math.MaxUint32 {
		return raDeprecatedLifetimeCapSecs
	}

	return valid
}

// merge merges a fresh interface observation into s and reports whether the
// active prefix changed.
func (s *raState) merge(obs raObservation, now time.Time) raActiveChange {
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

	activeChanged = s.mergeActiveObservation(obs.Active, prev, now)

	if s.active != nil {
		delete(s.deprecated, s.active.prefix)
	}

	observedInactive := s.mergeInactiveObservations(obs.Inactive, prevActivePrefix, activeChanged, now)

	s.deprecateMissingObservedInactivePrefixes(observedInactive, now)

	s.evictExpired(now)

	next := s.activeSnapshot(now)
	return raActiveChange{
		Changed: !sameActivePrefix(prev, next),
		Active:  next,
	}
}

// mergeActiveObservation updates the active prefix from obs and reports
// whether the active prefix changed.
func (s *raState) mergeActiveObservation(
	obs, prev *raPrefixSnapshot,
	now time.Time,
) (activeChanged bool) {
	switch {
	case obs != nil && s.active != nil && s.active.prefix == obs.Prefix:
		s.active = reconcileTrackedPrefix(s.active, *obs, raPrefixOriginObservedActive, now)
	case obs != nil:
		s.moveActiveToDeprecated(now)
		s.active = newTrackedPrefix(*obs, raPrefixOriginObservedActive, now)
		activeChanged = prev != nil && prev.Prefix != obs.Prefix
	default:
		s.moveActiveToDeprecated(now)
		s.active = nil
		activeChanged = prev != nil
	}

	return activeChanged
}

// mergeInactiveObservations reconciles inactive prefix observations with the
// tracked deprecated prefixes and returns the set of prefixes that still
// appear as observed inactive.
func (s *raState) mergeInactiveObservations(
	inactive []raPrefixSnapshot,
	prevActivePrefix netip.Prefix,
	activeChanged bool,
	now time.Time,
) (observedInactive map[netip.Prefix]struct{}) {
	observedInactive = map[netip.Prefix]struct{}{}

	for _, dep := range inactive {
		s.mergeInactiveObservation(dep, prevActivePrefix, activeChanged, now, observedInactive)
	}

	return observedInactive
}

// mergeInactiveObservation reconciles one inactive prefix observation.
func (s *raState) mergeInactiveObservation(
	dep raPrefixSnapshot,
	prevActivePrefix netip.Prefix,
	activeChanged bool,
	now time.Time,
	observedInactive map[netip.Prefix]struct{},
) {
	if s.active != nil && dep.Prefix == s.active.prefix {
		return
	}

	if activeChanged && dep.Prefix == prevActivePrefix && dep.PreferredSec == 0 {
		valid := capDeprecatedLifetime(dep.ValidSec)
		if valid == 0 {
			delete(s.deprecated, dep.Prefix)

			return
		}

		s.deprecated[dep.Prefix] = reconcileTrackedPrefix(
			s.deprecated[dep.Prefix],
			raPrefixSnapshot{
				Prefix:       dep.Prefix,
				PreferredSec: 0,
				ValidSec:     valid,
			},
			raPrefixOriginDeprecated,
			now,
		)

		return
	}

	if dep.PreferredSec > 0 {
		observedInactive[dep.Prefix] = struct{}{}
		s.deprecated[dep.Prefix] = reconcileTrackedPrefix(
			s.deprecated[dep.Prefix],
			dep,
			raPrefixOriginObservedInactive,
			now,
		)

		return
	}

	valid := capDeprecatedLifetime(dep.ValidSec)
	if valid == 0 {
		delete(s.deprecated, dep.Prefix)

		return
	}

	s.deprecated[dep.Prefix] = reconcileTrackedPrefix(
		s.deprecated[dep.Prefix],
		raPrefixSnapshot{
			Prefix:       dep.Prefix,
			PreferredSec: 0,
			ValidSec:     valid,
		},
		raPrefixOriginDeprecated,
		now,
	)
}

// deprecateMissingObservedInactivePrefixes converts any observed-inactive
// prefixes that are no longer present into deprecated prefixes.
func (s *raState) deprecateMissingObservedInactivePrefixes(
	observedInactive map[netip.Prefix]struct{},
	now time.Time,
) {
	for pref, tracked := range s.deprecated {
		if tracked.origin != raPrefixOriginObservedInactive {
			continue
		}

		if _, ok := observedInactive[pref]; !ok {
			s.deprecateTrackedPrefix(pref, tracked, now)
		}
	}
}

// deprecateTrackedPrefix converts a tracked prefix into a deprecated one using
// its remaining valid lifetime bounded by the standard two-hour cap.
func (s *raState) deprecateTrackedPrefix(pref netip.Prefix, tracked *trackedPrefix, now time.Time) {
	_, valid, expired := tracked.remaining(now)
	if expired || valid == 0 {
		delete(s.deprecated, pref)

		return
	}
	valid = capDeprecatedLifetime(valid)

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

	prefixes := buildInterfaceRAPrefixSnapshots(states)
	obs.Active, obs.Inactive = splitInterfaceRAPrefixSnapshots(prefixes)

	return obs
}

// buildInterfaceRAPrefixSnapshots groups eligible interface states by prefix
// and collapses each group into a single snapshot.
func buildInterfaceRAPrefixSnapshots(states []aghnet.IPv6AddrState) (prefixes []raPrefixSnapshot) {
	grouped := map[netip.Prefix][]aghnet.IPv6AddrState{}
	for _, st := range states {
		if !isEligibleRAPrefixState(st) {
			continue
		}

		pref := st.Prefix.Masked()
		grouped[pref] = append(grouped[pref], st)
	}

	prefixes = make([]raPrefixSnapshot, 0, len(grouped))
	for pref, group := range grouped {
		snap, ok := collapsePrefixGroup(pref, group)
		if !ok {
			continue
		}

		prefixes = append(prefixes, snap)
	}

	return prefixes
}

// splitInterfaceRAPrefixSnapshots selects the active snapshot and sorts the
// inactive ones.
func splitInterfaceRAPrefixSnapshots(
	prefixes []raPrefixSnapshot,
) (active *raPrefixSnapshot, inactive []raPrefixSnapshot) {
	activeIdx := selectInterfaceRAActivePrefixIndex(prefixes)
	for i, pref := range prefixes {
		if i == activeIdx {
			active = &raPrefixSnapshot{
				Prefix:       pref.Prefix,
				PreferredSec: pref.PreferredSec,
				ValidSec:     pref.ValidSec,
			}

			continue
		}

		inactive = append(inactive, pref)
	}

	slices.SortFunc(inactive, func(a, b raPrefixSnapshot) int {
		return prefixCompare(a.Prefix, b.Prefix)
	})

	return active, inactive
}

// selectInterfaceRAActivePrefixIndex reports the best active prefix index in
// prefixes, or -1 when none is suitable.
func selectInterfaceRAActivePrefixIndex(prefixes []raPrefixSnapshot) (activeIdx int) {
	activeIdx = -1
	for i, pref := range prefixes {
		if pref.PreferredSec == 0 {
			continue
		}

		if activeIdx == -1 || betterActivePrefix(pref, prefixes[activeIdx]) {
			activeIdx = i
		}
	}

	return activeIdx
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

// reconcileTrackedPrefix returns existing when its current countdown is
// consistent with the freshly observed snap; otherwise it builds a new
// tracked prefix from snap.  The consistency check tolerates a one-second
// slack in either direction so that sub-second drift between the wall clock
// of successive polls and the kernel's integer-second countdown does not
// perturb the absolute deadlines and trip the state digest.
func reconcileTrackedPrefix(
	existing *trackedPrefix,
	snap raPrefixSnapshot,
	origin raPrefixOrigin,
	now time.Time,
) (tracked *trackedPrefix) {
	if existing != nil && existing.prefix == snap.Prefix && existing.origin == origin {
		preferred, valid, _ := existing.remaining(now)
		if lifetimesConsistent(preferred, snap.PreferredSec) &&
			lifetimesConsistent(valid, snap.ValidSec) {
			return existing
		}
	}

	return newTrackedPrefix(snap, origin, now)
}

// lifetimesConsistent reports whether an existing tracked-prefix remaining
// lifetime and a freshly observed one describe the same kernel state.  Both
// "infinity" values must match exactly, because an infinite lifetime is a
// qualitatively different state from any finite one; finite values may differ
// by at most one second to absorb sub-second polling jitter.
func lifetimesConsistent(existing, observed uint32) (ok bool) {
	if existing == observed {
		return true
	}
	if existing == math.MaxUint32 || observed == math.MaxUint32 {
		return false
	}
	if existing+1 == observed || observed+1 == existing {
		return true
	}

	return false
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
