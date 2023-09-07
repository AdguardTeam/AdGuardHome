//go:build darwin || freebsd

package arpdb

import (
	"net"
	"net/netip"
)

const arpAOutput = `
invalid.mac (1.2.3.4) at 12:34:56:78:910 on el0 ifscope [ethernet]
invalid.ip  (1.2.3.4.5) at ab:cd:ef:ab:cd:12 on ek0 ifscope [ethernet]
invalid.fmt 1 at 12:cd:ef:ab:cd:ef on er0 ifscope [ethernet]
hostname.one (192.168.1.2) at ab:cd:ef:ab:cd:ef on en0 ifscope [ethernet]
hostname.two (::ffff:ffff) at ef:cd:ab:ef:cd:ab on em0 expires in 1198 seconds [ethernet]
? (::1234) at aa:bb:cc:dd:ee:ff on ej0 expires in 1918 seconds [ethernet]
`

var wantNeighs = []Neighbor{{
	Name: "hostname.one",
	IP:   netip.MustParseAddr("192.168.1.2"),
	MAC:  net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF},
}, {
	Name: "hostname.two",
	IP:   netip.MustParseAddr("::ffff:ffff"),
	MAC:  net.HardwareAddr{0xEF, 0xCD, 0xAB, 0xEF, 0xCD, 0xAB},
}, {
	Name: "",
	IP:   netip.MustParseAddr("::1234"),
	MAC:  net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
}}
