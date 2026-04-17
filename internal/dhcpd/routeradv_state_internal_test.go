package dhcpd

import (
	"math"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInterfaceRAObservation(t *testing.T) {
	obs := buildInterfaceRAObservation([]aghnet.IPv6AddrState{{
		Addr:                 netip.MustParseAddr("2001:db8::10"),
		Prefix:               netip.MustParsePrefix("2001:db8::/64"),
		PreferredLifetimeSec: math.MaxUint32,
		ValidLifetimeSec:     math.MaxUint32,
	}, {
		Addr:                 netip.MustParseAddr("fe80::1"),
		Prefix:               netip.MustParsePrefix("fe80::/64"),
		PreferredLifetimeSec: math.MaxUint32,
		ValidLifetimeSec:     math.MaxUint32,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8:1::20"),
		Prefix:               netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredLifetimeSec: 0,
		ValidLifetimeSec:     900,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8:2::30"),
		Prefix:               netip.MustParsePrefix("2001:db8:2::/64"),
		PreferredLifetimeSec: 1200,
		ValidLifetimeSec:     3600,
		Temporary:            true,
	}})

	require.NotNil(t, obs.Active)
	assert.Equal(t, netip.MustParseAddr("fe80::1"), obs.SourceAddr)
	assert.Equal(t, netip.MustParseAddr("fe80::1"), obs.RDNSSAddr)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:2::/64"), obs.Active.Prefix)
	assert.Equal(t, []raPrefixSnapshot{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: math.MaxUint32,
		ValidSec:     math.MaxUint32,
	}, {
		Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredSec: 0,
		ValidSec:     900,
	}}, obs.Inactive)
}

func TestBuildInterfaceRAObservation_OverlapPrefersLongerPreferredLifetime(t *testing.T) {
	obs := buildInterfaceRAObservation([]aghnet.IPv6AddrState{{
		Addr:                 netip.MustParseAddr("2001:db8::10"),
		Prefix:               netip.MustParsePrefix("2001:db8::/64"),
		PreferredLifetimeSec: 300,
		ValidLifetimeSec:     7200,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8:1::10"),
		Prefix:               netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredLifetimeSec: 1800,
		ValidLifetimeSec:     3600,
	}})

	require.NotNil(t, obs.Active)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), obs.Active.Prefix)
	assert.Equal(t, []raPrefixSnapshot{{
		Prefix:       netip.MustParsePrefix("2001:db8::/64"),
		PreferredSec: 300,
		ValidSec:     7200,
	}}, obs.Inactive)
}

func TestBuildInterfaceRAObservation_TemporaryOnlyPrefix(t *testing.T) {
	obs := buildInterfaceRAObservation([]aghnet.IPv6AddrState{{
		Addr:                 netip.MustParseAddr("fe80::1"),
		Prefix:               netip.MustParsePrefix("fe80::/64"),
		PreferredLifetimeSec: math.MaxUint32,
		ValidLifetimeSec:     math.MaxUint32,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8::20"),
		Prefix:               netip.MustParsePrefix("2001:db8::/64"),
		PreferredLifetimeSec: 1200,
		ValidLifetimeSec:     3600,
		Temporary:            true,
	}})

	require.NotNil(t, obs.Active)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), obs.Active.Prefix)
	assert.Empty(t, obs.Inactive)
}

func TestBuildInterfaceRAObservation_PrefersLongestLifetimeAcrossAddressKinds(t *testing.T) {
	obs := buildInterfaceRAObservation([]aghnet.IPv6AddrState{{
		Addr:                 netip.MustParseAddr("2001:db8::10"),
		Prefix:               netip.MustParsePrefix("2001:db8::/64"),
		PreferredLifetimeSec: 0,
		ValidLifetimeSec:     900,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8::20"),
		Prefix:               netip.MustParsePrefix("2001:db8::/64"),
		PreferredLifetimeSec: 1200,
		ValidLifetimeSec:     3600,
		Temporary:            true,
	}})

	require.NotNil(t, obs.Active)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), obs.Active.Prefix)
	assert.Equal(t, uint32(1200), obs.Active.PreferredSec)
	assert.Equal(t, uint32(3600), obs.Active.ValidSec)
}

func TestRAStateMerge_OverlappingOldActiveStaysPreferredIfObservedPreferred(t *testing.T) {
	now := time.Unix(75, 0)
	st := newObservedRAState()

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 900,
			ValidSec:     3600,
		}},
	}, now.Add(time.Minute))

	pios := st.pios(now.Add(time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(900), pios[1].PreferredSec)
	assert.Equal(t, uint32(3600), pios[1].ValidSec)
}

func TestRAStateMerge_OverlappingOldActiveBecomesDeprecatedWhenObservedDeprecated(t *testing.T) {
	now := time.Unix(75, 0)
	st := newObservedRAState()

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 0,
			ValidSec:     3600,
		}},
	}, now.Add(time.Minute))

	pios := st.pios(now.Add(time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(0), pios[1].PreferredSec)
	assert.Equal(t, uint32(3600), pios[1].ValidSec)
}

func TestRAStateMergeTracksPrefixTransitions(t *testing.T) {
	now := time.Unix(100, 0)
	st := newObservedRAState()

	change := st.merge(raObservation{
		SourceAddr: netip.MustParseAddr("fe80::1"),
		RDNSSAddr:  netip.MustParseAddr("fe80::1"),
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)
	require.True(t, change.Changed)
	require.NotNil(t, change.Active)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), change.Active.Prefix)

	change = st.merge(raObservation{
		SourceAddr: netip.MustParseAddr("fe80::1"),
		RDNSSAddr:  netip.MustParseAddr("fe80::1"),
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1200,
			ValidSec:     7000,
		},
	}, now.Add(time.Minute))
	assert.False(t, change.Changed)

	change = st.merge(raObservation{
		SourceAddr: netip.MustParseAddr("fe80::1"),
		RDNSSAddr:  netip.MustParseAddr("fe80::1"),
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 900,
			ValidSec:     3600,
		},
	}, now.Add(2*time.Minute))
	require.True(t, change.Changed)
	require.NotNil(t, change.Active)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), change.Active.Prefix)

	pios := st.pios(now.Add(2 * time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[0].Prefix)
	assert.Equal(t, uint32(900), pios[0].PreferredSec)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(0), pios[1].PreferredSec)
	assert.Equal(t, uint32(6940), pios[1].ValidSec)

	change = st.merge(raObservation{
		SourceAddr: netip.MustParseAddr("fe80::1"),
		RDNSSAddr:  netip.MustParseAddr("fe80::1"),
	}, now.Add(3*time.Minute))
	require.True(t, change.Changed)
	assert.Nil(t, change.Active)

	pios = st.pios(now.Add(3 * time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[0].Prefix)
	assert.Equal(t, uint32(6880), pios[0].ValidSec)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(3540), pios[1].ValidSec)
}

func TestRAStateMergeReappearingPrefix(t *testing.T) {
	now := time.Unix(50, 0)
	st := newObservedRAState()

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
	}, now)
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 900,
			ValidSec:     3600,
		},
	}, now.Add(time.Minute))

	change := st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 800,
			ValidSec:     3500,
		},
	}, now.Add(2*time.Minute))
	require.True(t, change.Changed)
	require.NotNil(t, change.Active)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), change.Active.Prefix)

	pios := st.pios(now.Add(2 * time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[1].Prefix)
}

func TestRAStateMerge_PreservesObservedInactivePrefix(t *testing.T) {
	now := time.Unix(200, 0)
	st := newObservedRAState()

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("fd00::/64"),
			PreferredSec: 900,
			ValidSec:     3600,
		}},
	}, now)

	pios := st.pios(now)
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("2001:db8:1::/64"), pios[0].Prefix)
	assert.Equal(t, netip.MustParsePrefix("fd00::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(900), pios[1].PreferredSec)
}

func TestRAStateMerge_DisappearingInactiveBecomesDeprecated(t *testing.T) {
	now := time.Unix(300, 0)
	st := newObservedRAState()

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1800,
			ValidSec:     7200,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("fd00::/64"),
			PreferredSec: 900,
			ValidSec:     3600,
		}},
	}, now)

	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8:1::/64"),
			PreferredSec: 1700,
			ValidSec:     7100,
		},
	}, now.Add(time.Minute))

	pios := st.pios(now.Add(time.Minute))
	require.Len(t, pios, 2)
	assert.Equal(t, netip.MustParsePrefix("fd00::/64"), pios[1].Prefix)
	assert.Equal(t, uint32(0), pios[1].PreferredSec)
	assert.Equal(t, uint32(3540), pios[1].ValidSec)
}

// TestRAStateMerge_PreservesDeadlinesUnderSubSecondJitter is the regression
// test for a bug where raState.merge would rebuild trackedPrefix deadlines
// from "now + observed_seconds" on every observation.  When "now" advances by
// a fractional second between polls, the absolute deadlines drifted even
// though the kernel state was identical, which made the state digest look
// changed and fired the v6Server reconciliation callback on every tick.
//
// The fix reconciles the existing tracked prefix against the fresh
// observation: when the remaining lifetimes are consistent (allowing a
// one-second slack), the existing pointer is kept so deadlines stay stable.
func TestRAStateMerge_PreservesDeadlinesUnderSubSecondJitter(t *testing.T) {
	st := newObservedRAState()

	// First poll at a non-whole-second offset.
	t1 := time.Unix(100, 500_000_000)
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1800,
			ValidSec:     3600,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("fd00::/64"),
			PreferredSec: 600,
			ValidSec:     2400,
		}},
	}, t1)

	firstActive := st.active
	firstInactive := st.deprecated[netip.MustParsePrefix("fd00::/64")]
	require.NotNil(t, firstActive)
	require.NotNil(t, firstInactive)

	firstActivePreferredUntil := firstActive.preferredUntil
	firstActiveValidUntil := firstActive.validUntil
	firstInactivePreferredUntil := firstInactive.preferredUntil
	firstInactiveValidUntil := firstInactive.validUntil

	// Second poll 5.3 wall-clock seconds later.  The kernel has decremented
	// its integer counters by 5 whole seconds so the remaining lifetimes
	// are still consistent with the stored deadlines modulo the sub-second
	// offset.
	t2 := time.Unix(105, 800_000_000)
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1795,
			ValidSec:     3595,
		},
		Inactive: []raPrefixSnapshot{{
			Prefix:       netip.MustParsePrefix("fd00::/64"),
			PreferredSec: 595,
			ValidSec:     2395,
		}},
	}, t2)

	// Reconciliation must keep the same trackedPrefix instance and leave
	// its deadlines untouched.
	require.Same(t, firstActive, st.active)
	require.Same(t, firstInactive, st.deprecated[netip.MustParsePrefix("fd00::/64")])
	assert.Equal(t, firstActivePreferredUntil, st.active.preferredUntil)
	assert.Equal(t, firstActiveValidUntil, st.active.validUntil)
	assert.Equal(t, firstInactivePreferredUntil, st.deprecated[netip.MustParsePrefix("fd00::/64")].preferredUntil)
	assert.Equal(t, firstInactiveValidUntil, st.deprecated[netip.MustParsePrefix("fd00::/64")].validUntil)

	// And the full state digests must compare equal.
	d1 := raStateDigest{
		sourceAddr: netip.Addr{},
		rdnssAddr:  netip.Addr{},
		active: &trackedPrefixDigest{
			prefix:         netip.MustParsePrefix("2001:db8::/64"),
			origin:         raPrefixOriginObservedActive,
			preferredUntil: firstActivePreferredUntil,
			validUntil:     firstActiveValidUntil,
		},
		deprecated: map[netip.Prefix]trackedPrefixDigest{
			netip.MustParsePrefix("fd00::/64"): {
				prefix:         netip.MustParsePrefix("fd00::/64"),
				origin:         raPrefixOriginObservedInactive,
				preferredUntil: firstInactivePreferredUntil,
				validUntil:     firstInactiveValidUntil,
			},
		},
	}
	d2 := st.digest(t2)
	assert.True(t, sameRAStateDigest(d1, d2))
}

// TestRAStateMerge_RealLifetimeChangeStillRebuildsDeadlines ensures that the
// slack in reconcileTrackedPrefix does not hide a real kernel state change
// that happens to also be a steady countdown — the previous poll's deadline
// must be replaced when the kernel report diverges by more than one second.
func TestRAStateMerge_RealLifetimeChangeStillRebuildsDeadlines(t *testing.T) {
	st := newObservedRAState()
	t1 := time.Unix(100, 0)
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 1800,
			ValidSec:     3600,
		},
	}, t1)

	firstActive := st.active
	require.NotNil(t, firstActive)

	// 5 wall-clock seconds pass but the kernel reports the lifetime jumped
	// *forward* — some upstream sent a fresh RA that extended the prefix.
	// The trackedPrefix must be rebuilt so the new deadline takes effect.
	t2 := time.Unix(105, 0)
	st.merge(raObservation{
		Active: &raPrefixSnapshot{
			Prefix:       netip.MustParsePrefix("2001:db8::/64"),
			PreferredSec: 3000,
			ValidSec:     7000,
		},
	}, t2)

	assert.NotSame(t, firstActive, st.active)
	assert.Equal(t, t2.Add(3000*time.Second), st.active.preferredUntil)
	assert.Equal(t, t2.Add(7000*time.Second), st.active.validUntil)
}

// TestLifetimesConsistent spot-checks the slack predicate that backs
// reconcileTrackedPrefix.
func TestLifetimesConsistent(t *testing.T) {
	assert.True(t, lifetimesConsistent(1800, 1800))
	assert.True(t, lifetimesConsistent(1800, 1799))
	assert.True(t, lifetimesConsistent(1799, 1800))
	assert.False(t, lifetimesConsistent(1800, 1798))
	assert.False(t, lifetimesConsistent(1798, 1800))
	assert.False(t, lifetimesConsistent(math.MaxUint32, 1800))
	assert.False(t, lifetimesConsistent(1800, math.MaxUint32))
	assert.True(t, lifetimesConsistent(math.MaxUint32, math.MaxUint32))
	assert.True(t, lifetimesConsistent(0, 0))
}
