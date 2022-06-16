//go:build windows
// +build windows

package aghnet

import (
	"net"
)

const arpAOutput = `

Interface: 192.168.1.1 --- 0x7
  Internet Address      Physical Address      Type
  192.168.1.2           ab-cd-ef-ab-cd-ef     dynamic
  ::ffff:ffff           ef-cd-ab-ef-cd-ab     static`

var wantNeighs = []Neighbor{{
	IP:  net.IPv4(192, 168, 1, 2),
	MAC: net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF},
}, {
	IP:  net.ParseIP("::ffff:ffff"),
	MAC: net.HardwareAddr{0xEF, 0xCD, 0xAB, 0xEF, 0xCD, 0xAB},
}}
