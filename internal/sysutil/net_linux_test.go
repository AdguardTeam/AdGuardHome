// +build linux

package sysutil

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			has, err := dhcpcdStaticConfig(r, "wlan0")
			assert.Nil(t, err)
			assert.Equal(t, tc.want, has)
		})
	}
}

func TestIfacesStaticConfig(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{{
		name: "has_not",
		data: []byte(`allow-hotplug enp0s3` + nl +
			`#iface enp0s3 inet static` + nl +
			`#  address 192.168.0.200` + nl +
			`#  netmask 255.255.255.0` + nl +
			`#  gateway 192.168.0.1` + nl +
			`iface enp0s3 inet dhcp` + nl,
		),
		want: false,
	}, {
		name: "has",
		data: []byte(`allow-hotplug enp0s3` + nl +
			`iface enp0s3 inet static` + nl +
			`  address 192.168.0.200` + nl +
			`  netmask 255.255.255.0` + nl +
			`  gateway 192.168.0.1` + nl +
			`#iface enp0s3 inet dhcp` + nl,
		),
		want: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.data)
			has, err := ifacesStaticConfig(r, "enp0s3")
			assert.Nil(t, err)
			assert.Equal(t, tc.want, has)
		})
	}
}

func TestSetStaticIPdhcpcdConf(t *testing.T) {
	dhcpcdConf := nl + `interface wlan0` + nl +
		`static ip_address=192.168.0.2/24` + nl +
		`static routers=192.168.0.1` + nl +
		`static domain_name_servers=192.168.0.2` + nl + nl

	s := updateStaticIPdhcpcdConf("wlan0", "192.168.0.2/24", "192.168.0.1", "192.168.0.2")
	assert.Equal(t, dhcpcdConf, s)

	// without gateway
	dhcpcdConf = nl + `interface wlan0` + nl +
		`static ip_address=192.168.0.2/24` + nl +
		`static domain_name_servers=192.168.0.2` + nl + nl

	s = updateStaticIPdhcpcdConf("wlan0", "192.168.0.2/24", "", "192.168.0.2")
	assert.Equal(t, dhcpcdConf, s)
}
