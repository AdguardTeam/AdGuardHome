package aghnet

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHostName(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		testCases := []struct {
			name string
			want string
			ip   netip.Addr
		}{{
			name: "good_ipv4",
			want: "127-0-0-1",
			ip:   netip.MustParseAddr("127.0.0.1"),
		}, {
			name: "good_ipv6",
			want: "fe00-0000-0000-0000-0000-0000-0000-0001",
			ip:   netip.MustParseAddr("fe00::1"),
		}, {
			name: "4to6",
			want: "1-2-3-4",
			ip:   netip.MustParseAddr("::ffff:1.2.3.4"),
		}}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				hostname := GenerateHostname(tc.ip)
				assert.Equal(t, tc.want, hostname)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		assert.Panics(t, func() { GenerateHostname(netip.Addr{}) })
	})
}
