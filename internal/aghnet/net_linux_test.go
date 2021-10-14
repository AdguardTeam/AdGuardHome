//go:build linux
// +build linux

package aghnet

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPCDStaticConfig(t *testing.T) {
	const iface interfaceName = `wlan0`

	testCases := []struct {
		name     string
		data     []byte
		wantCont bool
	}{{
		name: "has_not",
		data: []byte(`#comment` + nl +
			`# comment` + nl +
			`interface eth0` + nl +
			`static ip_address=192.168.0.1/24` + nl +
			`# interface ` + iface + nl +
			`static ip_address=192.168.1.1/24` + nl +
			`# comment` + nl,
		),
		wantCont: true,
	}, {
		name: "has",
		data: []byte(`#comment` + nl +
			`# comment` + nl +
			`interface eth0` + nl +
			`static ip_address=192.168.0.1/24` + nl +
			`# interface ` + iface + nl +
			`static ip_address=192.168.1.1/24` + nl +
			`# comment` + nl +
			`interface ` + iface + nl +
			`# comment` + nl +
			`static ip_address=192.168.2.1/24` + nl,
		),
		wantCont: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.data)
			_, cont, err := iface.dhcpcdStaticConfig(r)
			require.NoError(t, err)

			assert.Equal(t, tc.wantCont, cont)
		})
	}
}

func TestIfacesStaticConfig(t *testing.T) {
	const iface interfaceName = `enp0s3`

	testCases := []struct {
		name         string
		data         []byte
		wantCont     bool
		wantPatterns []string
	}{{
		name: "has_not",
		data: []byte(`allow-hotplug ` + iface + nl +
			`#iface enp0s3 inet static` + nl +
			`#  address 192.168.0.200` + nl +
			`#  netmask 255.255.255.0` + nl +
			`#  gateway 192.168.0.1` + nl +
			`iface ` + iface + ` inet dhcp` + nl,
		),
		wantCont:     true,
		wantPatterns: []string{},
	}, {
		name: "has",
		data: []byte(`allow-hotplug ` + iface + nl +
			`iface ` + iface + ` inet static` + nl +
			`  address 192.168.0.200` + nl +
			`  netmask 255.255.255.0` + nl +
			`  gateway 192.168.0.1` + nl +
			`#iface ` + iface + ` inet dhcp` + nl,
		),
		wantCont:     false,
		wantPatterns: []string{},
	}, {
		name: "return_patterns",
		data: []byte(`source hello` + nl +
			`source world` + nl +
			`#iface ` + iface + ` inet static` + nl,
		),
		wantCont:     true,
		wantPatterns: []string{"hello", "world"},
	}, {
		// This one tests if the first found valid interface prevents
		// checking files under the `source` directive.
		name: "ignore_patterns",
		data: []byte(`source hello` + nl +
			`source world` + nl +
			`iface ` + iface + ` inet static` + nl,
		),
		wantCont:     false,
		wantPatterns: []string{},
	}}

	for _, tc := range testCases {
		r := bytes.NewReader(tc.data)
		t.Run(tc.name, func(t *testing.T) {
			patterns, has, err := iface.ifacesStaticConfig(r)
			require.NoError(t, err)

			assert.Equal(t, tc.wantCont, has)
			assert.ElementsMatch(t, tc.wantPatterns, patterns)
		})
	}
}

func TestSetStaticIPdhcpcdConf(t *testing.T) {
	testCases := []struct {
		name       string
		dhcpcdConf string
		routers    net.IP
	}{{
		name: "with_gateway",
		dhcpcdConf: nl + `# wlan0 added by AdGuard Home.` + nl +
			`interface wlan0` + nl +
			`static ip_address=192.168.0.2/24` + nl +
			`static routers=192.168.0.1` + nl +
			`static domain_name_servers=192.168.0.2` + nl + nl,
		routers: net.IP{192, 168, 0, 1},
	}, {
		name: "without_gateway",
		dhcpcdConf: nl + `# wlan0 added by AdGuard Home.` + nl +
			`interface wlan0` + nl +
			`static ip_address=192.168.0.2/24` + nl +
			`static domain_name_servers=192.168.0.2` + nl + nl,
		routers: nil,
	}}

	ipNet := &net.IPNet{
		IP:   net.IP{192, 168, 0, 2},
		Mask: net.IPMask{255, 255, 255, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := dhcpcdConfIface("wlan0", ipNet, tc.routers, net.IP{192, 168, 0, 2})
			assert.Equal(t, tc.dhcpcdConf, s)
		})
	}
}
