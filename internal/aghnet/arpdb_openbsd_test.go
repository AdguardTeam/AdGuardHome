//go:build openbsd
// +build openbsd

package aghnet

import (
	"net"
)

const arpAOutput = `
Host        Ethernet Address  Netif Expire    Flags
192.168.1.2 ab:cd:ef:ab:cd:ef   em0 19m56s
::ffff:ffff ef:cd:ab:ef:cd:ab   em0 permanent l
`

var wantNeighs = []Neighbor{{
	IP:  net.IPv4(192, 168, 1, 2),
	MAC: net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF},
}, {
	IP:  net.ParseIP("::ffff:ffff"),
	MAC: net.HardwareAddr{0xEF, 0xCD, 0xAB, 0xEF, 0xCD, 0xAB},
}}
