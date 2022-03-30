//go:build linux
// +build linux

package aghnet

import (
	"io/fs"
	"net"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHasStaticIP(t *testing.T) {
	const ifaceName = "wlan0"

	const (
		dhcpcd    = "etc/dhcpcd.conf"
		netifaces = "etc/network/interfaces"
	)

	testCases := []struct {
		rootFsys   fs.FS
		name       string
		wantHas    assert.BoolAssertionFunc
		wantErrMsg string
	}{{
		rootFsys: fstest.MapFS{
			dhcpcd: &fstest.MapFile{
				Data: []byte(`#comment` + nl +
					`# comment` + nl +
					`interface eth0` + nl +
					`static ip_address=192.168.0.1/24` + nl +
					`# interface ` + ifaceName + nl +
					`static ip_address=192.168.1.1/24` + nl +
					`# comment` + nl,
				),
			},
		},
		name:       "dhcpcd_has_not",
		wantHas:    assert.False,
		wantErrMsg: `no information about static ip`,
	}, {
		rootFsys: fstest.MapFS{
			dhcpcd: &fstest.MapFile{
				Data: []byte(`#comment` + nl +
					`# comment` + nl +
					`interface ` + ifaceName + nl +
					`static ip_address=192.168.0.1/24` + nl +
					`# interface ` + ifaceName + nl +
					`static ip_address=192.168.1.1/24` + nl +
					`# comment` + nl,
				),
			},
		},
		name:       "dhcpcd_has",
		wantHas:    assert.True,
		wantErrMsg: ``,
	}, {
		rootFsys: fstest.MapFS{
			netifaces: &fstest.MapFile{
				Data: []byte(`allow-hotplug ` + ifaceName + nl +
					`#iface enp0s3 inet static` + nl +
					`#  address 192.168.0.200` + nl +
					`#  netmask 255.255.255.0` + nl +
					`#  gateway 192.168.0.1` + nl +
					`iface ` + ifaceName + ` inet dhcp` + nl,
				),
			},
		},
		name:       "netifaces_has_not",
		wantHas:    assert.False,
		wantErrMsg: `no information about static ip`,
	}, {
		rootFsys: fstest.MapFS{
			netifaces: &fstest.MapFile{
				Data: []byte(`allow-hotplug ` + ifaceName + nl +
					`iface ` + ifaceName + ` inet static` + nl +
					`  address 192.168.0.200` + nl +
					`  netmask 255.255.255.0` + nl +
					`  gateway 192.168.0.1` + nl +
					`#iface ` + ifaceName + ` inet dhcp` + nl,
				),
			},
		},
		name:       "netifaces_has",
		wantHas:    assert.True,
		wantErrMsg: ``,
	}, {
		rootFsys: fstest.MapFS{
			netifaces: &fstest.MapFile{
				Data: []byte(`source hello` + nl +
					`#iface ` + ifaceName + ` inet static` + nl,
				),
			},
			"hello": &fstest.MapFile{
				Data: []byte(`iface ` + ifaceName + ` inet static` + nl),
			},
		},
		name:       "netifaces_another_file",
		wantHas:    assert.True,
		wantErrMsg: ``,
	}, {
		rootFsys: fstest.MapFS{
			netifaces: &fstest.MapFile{
				Data: []byte(`source hello` + nl +
					`iface ` + ifaceName + ` inet static` + nl,
				),
			},
		},
		name:       "netifaces_ignore_another",
		wantHas:    assert.True,
		wantErrMsg: ``,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substRootDirFS(t, tc.rootFsys)

			has, err := IfaceHasStaticIP(ifaceName)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			tc.wantHas(t, has)
		})
	}
}

func TestSetStaticIP_dhcpcdConfIface(t *testing.T) {
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
