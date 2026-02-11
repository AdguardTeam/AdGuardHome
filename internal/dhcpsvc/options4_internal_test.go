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

const (
	// testIPv4Str is the string representation of the test IPv4 address.
	testIPv4Str = "192.0.2.1"

	// testAnotherIPv4Str is the string representation of the test another IPv4
	// address.
	testAnotherIPv4Str = "198.51.100.1"

	// broadcastIPv4Str is the string representation of the broadcast IPv4
	// address.
	broadcastIPv4Str = "255.255.255.255"
)

// broadcastAddr is the broadcast IPv4 address.
var broadcastAddr = netip.MustParseAddr(broadcastIPv4Str)

func TestIPv4Config_Options(t *testing.T) {
	var (
		ipv4        = netip.MustParseAddr(testIPv4Str)
		anotherIPv4 = netip.MustParseAddr(testAnotherIPv4Str)
		subnetMask  = netip.PrefixFrom(broadcastAddr, broadcastAddr.BitLen()/2).Masked().Addr()

		optSubnetMask = layers.NewDHCPOption(layers.DHCPOptSubnetMask, subnetMask.AsSlice())
		optDNS        = layers.NewDHCPOption(
			layers.DHCPOptDNS,
			append(ipv4.AsSlice(), anotherIPv4.AsSlice()...),
		)
		optBroadcast   = layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, ipv4.AsSlice())
		optStaticRoute = layers.NewDHCPOption(layers.DHCPOptClasslessStaticRoute, []byte("cba"))
	)
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
			Options: layers.DHCPOptions{optBroadcast},
		},
		wantExplicit: layers.DHCPOptions{optBroadcast},
	}, {
		name: "configured_ips",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{optDNS},
		},
		wantExplicit: layers.DHCPOptions{optDNS},
	}, {
		name: "configured_del",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{
				layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, nil),
			},
		},
		wantExplicit: layers.DHCPOptions{
			layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, nil),
		},
	}, {
		name: "rewritten_del",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{
				layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, nil),
				optBroadcast,
			},
		},
		wantExplicit: layers.DHCPOptions{optBroadcast},
	}, {
		name: "configured_and_del",
		conf: &IPv4Config{
			Options: layers.DHCPOptions{
				layers.NewDHCPOption(layers.DHCPOptClasslessStaticRoute, []byte("a")),
				layers.NewDHCPOption(layers.DHCPOptClasslessStaticRoute, nil),
				optStaticRoute,
			},
		},
		wantExplicit: layers.DHCPOptions{optStaticRoute},
	}, {
		name: "replace_config_value",
		conf: &IPv4Config{
			SubnetMask: netip.PrefixFrom(broadcastAddr, 3*broadcastAddr.BitLen()/4).Masked().Addr(),
			Options:    layers.DHCPOptions{optSubnetMask},
		},
		wantExplicit: layers.DHCPOptions{optSubnetMask},
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
