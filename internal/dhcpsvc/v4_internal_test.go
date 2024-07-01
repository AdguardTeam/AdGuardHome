package dhcpsvc

import (
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

func TestIPv4Config_Options(t *testing.T) {
	oneIP, otherIP := netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("5.6.7.8")
	subnetMask := netip.MustParseAddr("255.255.0.0")

	opt1 := layers.NewDHCPOption(layers.DHCPOptSubnetMask, subnetMask.AsSlice())
	opt6 := layers.NewDHCPOption(layers.DHCPOptDNS, append(oneIP.AsSlice(), otherIP.AsSlice()...))
	opt28 := layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, oneIP.AsSlice())
	opt121 := layers.NewDHCPOption(layers.DHCPOptClasslessStaticRoute, []byte("cba"))

	testCases := []struct {
		name         string
		conf         *IPv4Config
		wantExplicit layers.DHCPOptions
	}{{
		name: "all_default",
		conf: &IPv4Config{
			Options: nil,
		},
		wantExplicit: nil,
	}, {
		name: "configured_ip",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{opt28},
		},
		wantExplicit: layers.DHCPOptions{opt28},
	}, {
		name: "configured_ips",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{opt6},
		},
		wantExplicit: layers.DHCPOptions{opt6},
	}, {
		name: "configured_del",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{
				layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, nil),
			},
		},
		wantExplicit: nil,
	}, {
		name: "rewritten_del",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{
				layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, nil),
				opt28,
			},
		},
		wantExplicit: layers.DHCPOptions{opt28},
	}, {
		name: "configured_and_del",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{
				layers.NewDHCPOption(layers.DHCPOptClasslessStaticRoute, []byte("a")),
				layers.NewDHCPOption(layers.DHCPOptClasslessStaticRoute, nil),
				opt121,
			},
		},
		wantExplicit: layers.DHCPOptions{opt121},
	}, {
		name: "replace_config_value",
		conf: &IPv4Config{
			SubnetMask: netip.MustParseAddr("255.255.255.0"),
			Options:    layers.DHCPOptions{opt1},
		},
		wantExplicit: layers.DHCPOptions{opt1},
	}}

	ctx := testutil.ContextWithTimeout(t, time.Second)
	l := slogutil.NewDiscardLogger()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imp, exp := tc.conf.options(ctx, l)
			assert.Equal(t, tc.wantExplicit, exp)

			for c := range exp {
				assert.NotContains(t, imp, c)
			}
		})
	}
}
