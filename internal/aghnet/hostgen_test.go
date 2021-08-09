package aghnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHostName(t *testing.T) {
	testCases := []struct {
		name string
		want string
		ip   net.IP
	}{{
		name: "good_ipv4",
		want: "127-0-0-1",
		ip:   net.IP{127, 0, 0, 1},
	}, {
		name: "bad_ipv4",
		want: "",
		ip:   net.IP{127, 0, 0, 1, 0},
	}, {
		name: "good_ipv6",
		want: "fe00-0000-0000-0000-0000-0000-0000-0001",
		ip:   net.ParseIP("fe00::1"),
	}, {
		name: "bad_ipv6",
		want: "",
		ip: net.IP{
			0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff,
		},
	}, {
		name: "nil",
		want: "",
		ip:   nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hostname := GenerateHostname(tc.ip)
			assert.Equal(t, tc.want, hostname)
		})
	}
}
