package aghnet

import (
	"math"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIfconfigIPv6Addrs(t *testing.T) {
	states, err := parseIfconfigIPv6Addrs([]byte(`
en0: flags=8863<UP,BROADCAST>
	inet6 fe80::1%en0 prefixlen 64 scopeid 0x8 pltime infty vltime infty
	inet6 2001:db8::100 prefixlen 64 autoconf temporary pltime 600 vltime 1200
	inet6 2001:db8:1::200 prefixlen 64 tentative pltime 30 vltime 60
`))
	require.NoError(t, err)

	assert.Equal(t, []IPv6AddrState{{
		Addr:                 netip.MustParseAddr("fe80::1"),
		Prefix:               netip.MustParsePrefix("fe80::/64"),
		PreferredLifetimeSec: math.MaxUint32,
		ValidLifetimeSec:     math.MaxUint32,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8::100"),
		Prefix:               netip.MustParsePrefix("2001:db8::/64"),
		PreferredLifetimeSec: 600,
		ValidLifetimeSec:     1200,
		Temporary:            true,
	}, {
		Addr:                 netip.MustParseAddr("2001:db8:1::200"),
		Prefix:               netip.MustParsePrefix("2001:db8:1::/64"),
		PreferredLifetimeSec: 30,
		ValidLifetimeSec:     60,
		Tentative:            true,
	}}, states)
}
