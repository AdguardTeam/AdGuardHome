//go:build openbsd

package arpdb

import (
	"net"
	"net/netip"
)

const arpAOutput = `
Host        Ethernet Address  Netif Expire    Flags
1.2.3.4.5   aa:bb:cc:dd:ee:ff   em0 permanent
1.2.3.4     12:34:56:78:910     em0 permanent
192.168.1.2 ab:cd:ef:ab:cd:ef   em0 19m56s
::ffff:ffff ef:cd:ab:ef:cd:ab   em0 permanent l
`

var wantNeighs = []Neighbor{{
	IP:  netip.MustParseAddr("192.168.1.2"),
	MAC: net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF},
}, {
	IP:  netip.MustParseAddr("::ffff:ffff"),
	MAC: net.HardwareAddr{0xEF, 0xCD, 0xAB, 0xEF, 0xCD, 0xAB},
}}
