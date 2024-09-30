package rdns_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is a common timeout for tests and contexts.
const testTimeout = 1 * time.Second

func TestDefault_Process(t *testing.T) {
	ip1 := netip.MustParseAddr("1.2.3.4")
	revAddr1, err := netutil.IPToReversedAddr(ip1.AsSlice())
	require.NoError(t, err)

	ip2 := netip.MustParseAddr("4.3.2.1")
	revAddr2, err := netutil.IPToReversedAddr(ip2.AsSlice())
	require.NoError(t, err)

	localIP := netip.MustParseAddr("192.168.0.1")
	localRevAddr1, err := netutil.IPToReversedAddr(localIP.AsSlice())
	require.NoError(t, err)

	testCases := []struct {
		name string
		addr netip.Addr
		want string
	}{{
		name: "first",
		addr: ip1,
		want: revAddr1,
	}, {
		name: "second",
		addr: ip2,
		want: revAddr2,
	}, {
		name: "empty",
		addr: netip.MustParseAddr("0.0.0.0"),
		want: "",
	}, {
		name: "private",
		addr: localIP,
		want: localRevAddr1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hit := 0
			onExchange := func(ip netip.Addr) (host string, ttl time.Duration, err error) {
				hit++

				switch ip {
				case ip1:
					return revAddr1, time.Hour, nil
				case ip2:
					return revAddr2, time.Hour, nil
				case localIP:
					return localRevAddr1, time.Hour, nil
				default:
					return "", time.Hour, nil
				}
			}

			r := rdns.New(&rdns.Config{
				CacheSize: 100,
				CacheTTL:  time.Hour,
				Exchanger: &aghtest.Exchanger{OnExchange: onExchange},
			})

			got, changed := r.Process(testutil.ContextWithTimeout(t, testTimeout), tc.addr)
			require.True(t, changed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, 1, hit)

			// From cache.
			got, changed = r.Process(testutil.ContextWithTimeout(t, testTimeout), tc.addr)
			require.False(t, changed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, 1, hit)
		})
	}

	t.Run("zero_ttl", func(t *testing.T) {
		const cacheTTL = time.Second / 2

		zeroTTLExchanger := &aghtest.Exchanger{
			OnExchange: func(ip netip.Addr) (host string, ttl time.Duration, err error) {
				return revAddr1, 0, nil
			},
		}

		r := rdns.New(&rdns.Config{
			CacheSize: 1,
			CacheTTL:  cacheTTL,
			Exchanger: zeroTTLExchanger,
		})

		got, changed := r.Process(testutil.ContextWithTimeout(t, testTimeout), ip1)
		require.True(t, changed)
		assert.Equal(t, revAddr1, got)

		zeroTTLExchanger.OnExchange = func(ip netip.Addr) (host string, ttl time.Duration, err error) {
			return revAddr2, time.Hour, nil
		}

		ctx := testutil.ContextWithTimeout(t, testTimeout)
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			got, changed = r.Process(ctx, ip1)
			assert.True(t, changed)
			assert.Equal(t, revAddr2, got)
		}, 2*cacheTTL, time.Millisecond*100)

		assert.Never(t, func() (changed bool) {
			_, changed = r.Process(testutil.ContextWithTimeout(t, testTimeout), ip1)

			return changed
		}, 2*cacheTTL, time.Millisecond*100)
	})
}
