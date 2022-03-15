//go:build darwin || freebsd
// +build darwin freebsd

package aghnet

import (
	"net"
)

const arpAOutput = `
hostname.one (192.168.1.2) at ab:cd:ef:ab:cd:ef on en0 ifscope [ethernet]
hostname.two (::ffff:ffff) at ef:cd:ab:ef:cd:ab on em0 expires in 1198 seconds [ethernet]
`

var wantNeighs = []Neighbor{{
	Name: "hostname.one",
	IP:   net.IPv4(192, 168, 1, 2),
	MAC:  net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF},
}, {
	Name: "hostname.two",
	IP:   net.ParseIP("::ffff:ffff"),
	MAC:  net.HardwareAddr{0xEF, 0xCD, 0xAB, 0xEF, 0xCD, 0xAB},
}}
