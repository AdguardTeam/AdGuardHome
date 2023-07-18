package rdns_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/rdns"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	config := &rdns.Config{
		CacheSize: 100,
		CacheTTL:  time.Hour,
	}

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
			onExchange := func(ip netip.Addr) (host string, err error) {
				hit++

				switch ip {
				case ip1:
					return revAddr1, nil
				case ip2:
					return revAddr2, nil
				case localIP:
					return localRevAddr1, nil
				default:
					return "", nil
				}
			}
			exchanger := &aghtest.Exchanger{
				OnExchange: onExchange,
			}

			config.Exchanger = exchanger
			r := rdns.New(config)

			got, changed := r.Process(tc.addr)
			require.True(t, changed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, 1, hit)

			// From cache.
			got, changed = r.Process(tc.addr)
			require.False(t, changed)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, 1, hit)
		})
	}
}
