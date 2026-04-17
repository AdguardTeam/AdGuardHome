//go:build openbsd

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
			"ifconfig -A em0 inet6",
			0,
			`
em0: flags=8843<UP,BROADCAST>
	inet6 fe80::1%em0 prefixlen 64 scopeid 0x1 pltime infty vltime infty
	inet6 2001:db8::2 prefixlen 64 autoconf tentative pltime 30 vltime 60
`,
			nil,
		),
		"em0",
	)
	require.NoError(t, err)
	require.Len(t, states, 2)

	assert.True(t, states[1].Tentative)
	assert.Equal(t, uint32(30), states[1].PreferredLifetimeSec)
}
