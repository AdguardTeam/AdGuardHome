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

func TestRecurrentChecker(t *testing.T) {
	c := &recurrentChecker{
		checker:  ifacesStaticConfig,
		initPath: "./testdata/include-subsources",
	}

	has, err := c.check("sample_name")
	require.NoError(t, err)
	assert.True(t, has)

	has, err = c.check("another_name")
	require.NoError(t, err)
	assert.False(t, has)
}

const nl = "\n"

func TestDHCPCDStaticConfig(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{{
		name: "has_not",
		data: []byte(`#comment` + nl +
			`# comment` + nl +
			`interface eth0` + nl +
			`static ip_address=192.168.0.1/24` + nl +
			`# interface wlan0` + nl +
			`static ip_address=192.168.1.1/24` + nl +
			`# comment` + nl,
		),
		want: false,
	}, {
		name: "has",
		data: []byte(`#comment` + nl +
			`# comment` + nl +
			`interface eth0` + nl +
			`static ip_address=192.168.0.1/24` + nl +
			`# interface wlan0` + nl +
			`static ip_address=192.168.1.1/24` + nl +
			`# comment` + nl +
			`interface wlan0` + nl +
			`# comment` + nl +
			`static ip_address=192.168.2.1/24` + nl,
		),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.data)
			_, has, err := dhcpcdStaticConfig(r, "wlan0")
			require.NoError(t, err)

			assert.Equal(t, tc.want, has)
		})
	}
}

func TestIfacesStaticConfig(t *testing.T) {
	testCases := []struct {
		name         string
		data         []byte
		want         bool
		wantPatterns []string
	}{{
		name: "has_not",
		data: []byte(`allow-hotplug enp0s3` + nl +
			`#iface enp0s3 inet static` + nl +
			`#  address 192.168.0.200` + nl +
			`#  netmask 255.255.255.0` + nl +
			`#  gateway 192.168.0.1` + nl +
			`iface enp0s3 inet dhcp` + nl,
		),
		want:         false,
		wantPatterns: []string{},
	}, {
		name: "has",
		data: []byte(`allow-hotplug enp0s3` + nl +
			`iface enp0s3 inet static` + nl +
			`  address 192.168.0.200` + nl +
			`  netmask 255.255.255.0` + nl +
			`  gateway 192.168.0.1` + nl +
			`#iface enp0s3 inet dhcp` + nl,
		),
		want:         true,
		wantPatterns: []string{},
	}, {
		name: "return_patterns",
		data: []byte(`source hello` + nl +
			`source world` + nl +
			`#iface enp0s3 inet static` + nl,
		),
		want:         false,
		wantPatterns: []string{"hello", "world"},
	}, {
		// This one tests if the first found valid interface prevents
		// checking files under the `source` directive.
		name: "ignore_patterns",
		data: []byte(`source hello` + nl +
			`source world` + nl +
			`iface enp0s3 inet static` + nl,
		),
		want:         true,
		wantPatterns: []string{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.data)
			patterns, has, err := ifacesStaticConfig(r, "enp0s3")
			require.NoError(t, err)

			assert.Equal(t, tc.want, has)
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
