//go:build darwin

package aghnet

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserveIPv6Addrs(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	states, err := ObserveIPv6Addrs(
		ctx,
		testLogger,
		agh.NewCommandConstructor(
			"ifconfig -L en0 inet6",
			0,
			`
en0: flags=8863<UP,BROADCAST>
	inet6 fe80::1%en0 prefixlen 64 scopeid 0x8 pltime infty vltime infty
	inet6 2001:db8::2 prefixlen 64 autoconf pltime 600 vltime 1200
`,
			nil,
		),
		"en0",
	)
	require.NoError(t, err)
	require.Len(t, states, 2)

	assert.Equal(t, "fe80::1", states[0].Addr.String())
	assert.Equal(t, "2001:db8::/64", states[1].Prefix.String())
	assert.Equal(t, uint32(600), states[1].PreferredLifetimeSec)
}
