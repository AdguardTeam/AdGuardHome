package aghnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHostName(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		testCases := []struct {
			name string
			want string
			ip   net.IP
		}{{
			name: "good_ipv4",
			want: "127-0-0-1",
			ip:   net.IP{127, 0, 0, 1},
		}, {
			name: "good_ipv6",
			want: "fe00-0000-0000-0000-0000-0000-0000-0001",
			ip:   net.ParseIP("fe00::1"),
		}, {
			name: "4to6",
			want: "1-2-3-4",
			ip:   net.ParseIP("::ffff:1.2.3.4"),
		}}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				hostname := GenerateHostname(tc.ip)
				assert.Equal(t, tc.want, hostname)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		testCases := []struct {
			name string
			ip   net.IP
		}{{
			name: "bad_ipv4",
			ip:   net.IP{127, 0, 0, 1, 0},
		}, {
			name: "bad_ipv6",
			ip: net.IP{
				0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff,
			},
		}, {
			name: "nil",
			ip:   nil,
		}}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Panics(t, func() { GenerateHostname(tc.ip) })
			})
		}
	})
}
