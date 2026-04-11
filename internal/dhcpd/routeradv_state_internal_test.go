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
