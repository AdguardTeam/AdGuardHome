//go:build freebsd

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
			"ifconfig -L em0 inet6",
			0,
			`
em0: flags=8863<UP,BROADCAST>
	inet6 fe80::1%em0 prefixlen 64 scopeid 0x1 pltime infty vltime infty
	inet6 2001:db8::2 prefixlen 64 autoconf temporary pltime 600 vltime 1200
`,
			nil,
		),
		"em0",
	)
	require.NoError(t, err)
	require.Len(t, states, 2)

	assert.True(t, states[1].Temporary)
	assert.Equal(t, uint32(1200), states[1].ValidLifetimeSec)
}
