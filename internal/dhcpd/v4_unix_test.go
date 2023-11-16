//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	DefaultRangeStart = netip.MustParseAddr("192.168.10.100")
	DefaultRangeEnd   = netip.MustParseAddr("192.168.10.200")
	DefaultGatewayIP  = netip.MustParseAddr("192.168.10.1")
	DefaultSelfIP     = netip.MustParseAddr("192.168.10.2")
	DefaultSubnetMask = netip.MustParseAddr("255.255.255.0")
)

// defaultV4ServerConf returns the default configuration for *v4Server to use in
// tests.
func defaultV4ServerConf() (conf *V4ServerConf) {
	return &V4ServerConf{
		Enabled:    true,
		RangeStart: DefaultRangeStart,
		RangeEnd:   DefaultRangeEnd,
		GatewayIP:  DefaultGatewayIP,
		SubnetMask: DefaultSubnetMask,
		notify:     testNotify,
		dnsIPAddrs: []netip.Addr{DefaultSelfIP},
	}
}

// defaultSrv prepares the default DHCPServer to use in tests.  The underlying
// type of s is *v4Server.
func defaultSrv(t *testing.T) (s DHCPServer) {
	t.Helper()

	var err error
	s, err = v4Create(defaultV4ServerConf())
	require.NoError(t, err)

	return s
}

func TestV4Server_leasing(t *testing.T) {
	const (
		staticName  = "static-client"
		anotherName = "another-client"
	)

	staticIP := netip.MustParseAddr("192.168.10.10")
	anotherIP := DefaultRangeStart
	staticMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
	anotherMAC := net.HardwareAddr{0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}

	s := defaultSrv(t)

	t.Run("add_static", func(t *testing.T) {
		err := s.AddStaticLease(&dhcpsvc.Lease{
			Hostname: staticName,
			HWAddr:   staticMAC,
			IP:       staticIP,
			IsStatic: true,
		})
		require.NoError(t, err)

		t.Run("same_name", func(t *testing.T) {
			err = s.AddStaticLease(&dhcpsvc.Lease{
				Hostname: staticName,
				HWAddr:   anotherMAC,
				IP:       anotherIP,
				IsStatic: true,
			})
			assert.ErrorIs(t, err, ErrDupHostname)
		})

		t.Run("same_mac", func(t *testing.T) {
			wantErrMsg := "dhcpv4: adding static lease: removing " +
				"dynamic leases for " + anotherIP.String() +
				" (" + staticMAC.String() + "): static lease already exists"

			err = s.AddStaticLease(&dhcpsvc.Lease{
				Hostname: anotherName,
				HWAddr:   staticMAC,
				IP:       anotherIP,
				IsStatic: true,
			})
			testutil.AssertErrorMsg(t, wantErrMsg, err)
		})

		t.Run("same_ip", func(t *testing.T) {
			wantErrMsg := "dhcpv4: adding static lease: removing " +
				"dynamic leases for " + staticIP.String() +
				" (" + anotherMAC.String() + "): static lease already exists"

			err = s.AddStaticLease(&dhcpsvc.Lease{
				Hostname: anotherName,
				HWAddr:   anotherMAC,
				IP:       staticIP,
				IsStatic: true,
			})
			testutil.AssertErrorMsg(t, wantErrMsg, err)
		})
	})

	t.Run("add_dynamic", func(t *testing.T) {
		s4, ok := s.(*v4Server)
		require.True(t, ok)

		discoverAnOffer := func(
			t *testing.T,
			name string,
			netIP netip.Addr,
			mac net.HardwareAddr,
		) (resp *dhcpv4.DHCPv4) {
			testutil.CleanupAndRequireSuccess(t, func() (err error) {
				return s.ResetLeases(s.GetLeases(LeasesStatic))
			})

			ip := net.IP(netIP.AsSlice())
			req, err := dhcpv4.NewDiscovery(
				mac,
				dhcpv4.WithOption(dhcpv4.OptHostName(name)),
				dhcpv4.WithOption(dhcpv4.OptRequestedIPAddress(ip)),
				dhcpv4.WithOption(dhcpv4.OptClientIdentifier([]byte{1, 2, 3, 4, 5, 6, 8})),
				dhcpv4.WithGatewayIP(DefaultGatewayIP.AsSlice()),
			)
			require.NoError(t, err)

			resp = &dhcpv4.DHCPv4{}
			res := s4.handle(req, resp)
			require.Positive(t, res)
			require.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())

			resp.ClientHWAddr = mac

			return resp
		}

		t.Run("same_name", func(t *testing.T) {
			resp := discoverAnOffer(t, staticName, anotherIP, anotherMAC)

			req, err := dhcpv4.NewRequestFromOffer(resp, dhcpv4.WithOption(
				dhcpv4.OptHostName(staticName),
			))
			require.NoError(t, err)

			res := s4.handle(req, resp)
			require.Positive(t, res)

			var netIP netip.Addr
			netIP, ok = netip.AddrFromSlice(resp.YourIPAddr)
			require.True(t, ok)

			assert.Equal(t, aghnet.GenerateHostname(netIP), resp.HostName())
		})

		t.Run("same_mac", func(t *testing.T) {
			resp := discoverAnOffer(t, anotherName, anotherIP, staticMAC)

			req, err := dhcpv4.NewRequestFromOffer(resp, dhcpv4.WithOption(
				dhcpv4.OptHostName(anotherName),
			))
			require.NoError(t, err)

			res := s4.handle(req, resp)
			require.Positive(t, res)

			fqdnOptData := resp.Options.Get(dhcpv4.OptionFQDN)
			require.Len(t, fqdnOptData, 3+len(staticName))
			assert.Equal(t, []uint8(staticName), fqdnOptData[3:])

			ip := net.IP(staticIP.AsSlice())
			assert.Equal(t, ip, resp.YourIPAddr)
		})

		t.Run("same_ip", func(t *testing.T) {
			resp := discoverAnOffer(t, anotherName, staticIP, anotherMAC)

			req, err := dhcpv4.NewRequestFromOffer(resp, dhcpv4.WithOption(
				dhcpv4.OptHostName(anotherName),
			))
			require.NoError(t, err)

			res := s4.handle(req, resp)
			require.Positive(t, res)

			assert.NotEqual(t, staticIP, resp.YourIPAddr)
		})
	})
}

func TestV4Server_AddRemove_static(t *testing.T) {
	s := defaultSrv(t)

	ls := s.GetLeases(LeasesStatic)
	require.Empty(t, ls)

	testCases := []struct {
		lease      *dhcpsvc.Lease
		name       string
		wantErrMsg string
	}{{
		lease: &dhcpsvc.Lease{
			Hostname: "success.local",
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       netip.MustParseAddr("192.168.10.150"),
		},
		name:       "success",
		wantErrMsg: "",
	}, {
		lease: &dhcpsvc.Lease{
			Hostname: "probably-router.local",
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       DefaultGatewayIP,
		},
		name: "with_gateway_ip",
		wantErrMsg: "dhcpv4: adding static lease: " +
			`can't assign the gateway IP "192.168.10.1" to the lease`,
	}, {
		lease: &dhcpsvc.Lease{
			Hostname: "ip6.local",
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       netip.MustParseAddr("ffff::1"),
		},
		name: "ipv6",
		wantErrMsg: `dhcpv4: adding static lease: ` +
			`invalid IP "ffff::1": only IPv4 is supported`,
	}, {
		lease: &dhcpsvc.Lease{
			Hostname: "bad-mac.local",
			HWAddr:   net.HardwareAddr{0xAA, 0xAA},
			IP:       netip.MustParseAddr("192.168.10.150"),
		},
		name: "bad_mac",
		wantErrMsg: `dhcpv4: adding static lease: bad mac address "aa:aa": ` +
			`bad mac address length 2, allowed: [6 8 20]`,
	}, {
		lease: &dhcpsvc.Lease{
			Hostname: "bad-lbl-.local",
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       netip.MustParseAddr("192.168.10.150"),
		},
		name: "bad_hostname",
		wantErrMsg: `dhcpv4: adding static lease: validating hostname: ` +
			`bad hostname "bad-lbl-.local": ` +
			`bad hostname label "bad-lbl-": bad hostname label rune '-'`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.AddStaticLease(tc.lease)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			if tc.wantErrMsg != "" {
				return
			}

			err = s.RemoveStaticLease(&dhcpsvc.Lease{
				IP:     tc.lease.IP,
				HWAddr: tc.lease.HWAddr,
			})
			diffErrMsg := fmt.Sprintf("dhcpv4: lease for ip %s is different: %+v", tc.lease.IP, tc.lease)
			testutil.AssertErrorMsg(t, diffErrMsg, err)

			// Remove static lease.
			err = s.RemoveStaticLease(tc.lease)
			require.NoError(t, err)
		})

		ls = s.GetLeases(LeasesStatic)
		require.Emptyf(t, ls, "after %s", tc.name)
	}
}

func TestV4_AddReplace(t *testing.T) {
	sIface := defaultSrv(t)

	s, ok := sIface.(*v4Server)
	require.True(t, ok)

	dynLeases := []dhcpsvc.Lease{{
		Hostname: "dynamic-1.local",
		HWAddr:   net.HardwareAddr{0x11, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       netip.MustParseAddr("192.168.10.150"),
	}, {
		Hostname: "dynamic-2.local",
		HWAddr:   net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       netip.MustParseAddr("192.168.10.151"),
	}}

	for i := range dynLeases {
		err := s.addLease(&dynLeases[i])
		require.NoError(t, err)
	}

	stLeases := []*dhcpsvc.Lease{{
		Hostname: "static-1.local",
		HWAddr:   net.HardwareAddr{0x33, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       netip.MustParseAddr("192.168.10.150"),
	}, {
		Hostname: "static-2.local",
		HWAddr:   net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       netip.MustParseAddr("192.168.10.152"),
	}}

	for _, l := range stLeases {
		err := s.AddStaticLease(l)
		require.NoError(t, err)
	}

	ls := s.GetLeases(LeasesStatic)
	require.Len(t, ls, 2)

	for i, l := range ls {
		assert.Equal(t, stLeases[i].IP, l.IP)
		assert.Equal(t, stLeases[i].HWAddr, l.HWAddr)
		assert.True(t, l.IsStatic)
	}
}

func TestV4Server_handle_optionsPriority(t *testing.T) {
	defaultIP := netip.MustParseAddr("192.168.1.1")
	knownIP := net.IP{1, 2, 3, 4}

	// prepareSrv creates a *v4Server and sets the opt6IPs in the initial
	// configuration of the server as the value for DHCP option 6.
	prepareSrv := func(t *testing.T, opt6IPs []net.IP) (s *v4Server) {
		t.Helper()

		conf := defaultV4ServerConf()
		if len(opt6IPs) > 0 {
			b := &strings.Builder{}
			stringutil.WriteToBuilder(b, "6 ips ", opt6IPs[0].String())
			for _, ip := range opt6IPs[1:] {
				stringutil.WriteToBuilder(b, ",", ip.String())
			}
			conf.Options = []string{b.String()}
		} else {
			defer func() { s.implicitOpts.Update(dhcpv4.OptDNS(defaultIP.AsSlice())) }()
		}

		var err error
		s, err = v4Create(conf)
		require.NoError(t, err)

		s.conf.dnsIPAddrs = []netip.Addr{defaultIP}

		return s
	}

	// checkResp creates a discovery message with DHCP option 6 requested amd
	// asserts the response to contain wantIPs in this option.
	checkResp := func(t *testing.T, s *v4Server, wantIPs []net.IP) {
		t.Helper()

		mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}
		req, err := dhcpv4.NewDiscovery(mac, dhcpv4.WithRequestedOptions(
			dhcpv4.OptionDomainNameServer,
		))
		require.NoError(t, err)

		var resp *dhcpv4.DHCPv4
		resp, err = dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		res := s.handle(req, resp)
		require.Equal(t, 1, res)

		o := resp.GetOneOption(dhcpv4.OptionDomainNameServer)
		require.NotEmpty(t, o)

		wantData := []byte{}
		for _, ip := range wantIPs {
			wantData = append(wantData, ip...)
		}
		assert.Equal(t, o, wantData)
	}

	t.Run("default", func(t *testing.T) {
		s := prepareSrv(t, nil)

		checkResp(t, s, []net.IP{defaultIP.AsSlice()})
	})

	t.Run("explicitly_configured", func(t *testing.T) {
		s := prepareSrv(t, []net.IP{knownIP, knownIP})

		checkResp(t, s, []net.IP{knownIP, knownIP})
	})
}

func TestV4Server_updateOptions(t *testing.T) {
	testIP := net.IP{1, 2, 3, 4}

	dontWant := func(c dhcpv4.OptionCode) (opt dhcpv4.Option) {
		return dhcpv4.OptGeneric(c, nil)
	}

	testCases := []struct {
		name     string
		wantOpts dhcpv4.Options
		reqMods  []dhcpv4.Modifier
		confOpts []string
	}{{
		name: "requested_default",
		wantOpts: dhcpv4.OptionsFromList(
			dhcpv4.OptBroadcastAddress(netutil.IPv4bcast()),
		),
		reqMods: []dhcpv4.Modifier{
			dhcpv4.WithRequestedOptions(dhcpv4.OptionBroadcastAddress),
		},
		confOpts: nil,
	}, {
		name: "requested_non-default",
		wantOpts: dhcpv4.OptionsFromList(
			dhcpv4.OptBroadcastAddress(testIP),
		),
		reqMods: []dhcpv4.Modifier{
			dhcpv4.WithRequestedOptions(dhcpv4.OptionBroadcastAddress),
		},
		confOpts: []string{
			fmt.Sprintf("%d ip %s", dhcpv4.OptionBroadcastAddress, testIP),
		},
	}, {
		name: "non-requested_default",
		wantOpts: dhcpv4.OptionsFromList(
			dontWant(dhcpv4.OptionBroadcastAddress),
		),
		reqMods:  nil,
		confOpts: nil,
	}, {
		name: "non-requested_non-default",
		wantOpts: dhcpv4.OptionsFromList(
			dhcpv4.OptBroadcastAddress(testIP),
		),
		reqMods: nil,
		confOpts: []string{
			fmt.Sprintf("%d ip %s", dhcpv4.OptionBroadcastAddress, testIP),
		},
	}, {
		name: "requested_deleted",
		wantOpts: dhcpv4.OptionsFromList(
			dontWant(dhcpv4.OptionBroadcastAddress),
		),
		reqMods: []dhcpv4.Modifier{
			dhcpv4.WithRequestedOptions(dhcpv4.OptionBroadcastAddress),
		},
		confOpts: []string{
			fmt.Sprintf("%d del", dhcpv4.OptionBroadcastAddress),
		},
	}, {
		name: "requested_non-default_deleted",
		wantOpts: dhcpv4.OptionsFromList(
			dontWant(dhcpv4.OptionBroadcastAddress),
		),
		reqMods: []dhcpv4.Modifier{
			dhcpv4.WithRequestedOptions(dhcpv4.OptionBroadcastAddress),
		},
		confOpts: []string{
			fmt.Sprintf("%d ip %s", dhcpv4.OptionBroadcastAddress, testIP),
			fmt.Sprintf("%d del", dhcpv4.OptionBroadcastAddress),
		},
	}}

	for _, tc := range testCases {
		req, err := dhcpv4.New(tc.reqMods...)
		require.NoError(t, err)

		resp, err := dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		conf := defaultV4ServerConf()
		conf.Options = tc.confOpts

		s, err := v4Create(conf)
		require.NoError(t, err)
		require.IsType(t, (*v4Server)(nil), s)

		t.Run(tc.name, func(t *testing.T) {
			s.updateOptions(req, resp)

			for c, v := range tc.wantOpts {
				if v == nil {
					assert.NotContains(t, resp.Options, c)

					continue
				}

				assert.Equal(t, v, resp.Options.Get(dhcpv4.GenericOptionCode(c)))
			}
		})
	}
}

func TestV4StaticLease_Get(t *testing.T) {
	sIface := defaultSrv(t)

	s, ok := sIface.(*v4Server)
	require.True(t, ok)

	dnsAddr := netip.MustParseAddr("192.168.10.1")
	s.conf.dnsIPAddrs = []netip.Addr{dnsAddr}
	s.implicitOpts.Update(dhcpv4.OptDNS(dnsAddr.AsSlice()))

	l := &dhcpsvc.Lease{
		Hostname: "static-1.local",
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       netip.MustParseAddr("192.168.10.150"),
	}
	err := s.AddStaticLease(l)
	require.NoError(t, err)

	var req, resp *dhcpv4.DHCPv4
	mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	t.Run("discover", func(t *testing.T) {
		req, err = dhcpv4.NewDiscovery(mac, dhcpv4.WithRequestedOptions(
			dhcpv4.OptionDomainNameServer,
		))
		require.NoError(t, err)

		resp, err = dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		assert.Equal(t, 1, s.handle(req, resp))
	})

	// Don't continue if we got any errors in the previous subtest.
	require.NoError(t, err)

	t.Run("offer", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)

		ip := net.IP(l.IP.AsSlice())
		assert.True(t, ip.Equal(resp.YourIPAddr))

		assert.True(t, resp.Router()[0].Equal(s.conf.GatewayIP.AsSlice()))
		assert.True(t, resp.ServerIdentifier().Equal(s.conf.GatewayIP.AsSlice()))

		ones, _ := resp.SubnetMask().Size()
		assert.Equal(t, s.conf.subnet.Bits(), ones)
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	})

	t.Run("request", func(t *testing.T) {
		req, err = dhcpv4.NewRequestFromOffer(resp)
		require.NoError(t, err)

		resp, err = dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		assert.Equal(t, 1, s.handle(req, resp))
	})

	require.NoError(t, err)

	t.Run("ack", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeAck, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)

		ip := net.IP(l.IP.AsSlice())
		assert.True(t, ip.Equal(resp.YourIPAddr))

		assert.True(t, resp.Router()[0].Equal(s.conf.GatewayIP.AsSlice()))
		assert.True(t, resp.ServerIdentifier().Equal(s.conf.GatewayIP.AsSlice()))

		ones, _ := resp.SubnetMask().Size()
		assert.Equal(t, s.conf.subnet.Bits(), ones)
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	})

	dnsAddrs := resp.DNS()
	require.Len(t, dnsAddrs, 1)

	assert.True(t, dnsAddrs[0].Equal(s.conf.GatewayIP.AsSlice()))

	t.Run("check_lease", func(t *testing.T) {
		ls := s.GetLeases(LeasesStatic)
		require.Len(t, ls, 1)

		assert.Equal(t, l.IP, ls[0].IP)
		assert.Equal(t, mac, ls[0].HWAddr)
	})
}

func TestV4DynamicLease_Get(t *testing.T) {
	conf := defaultV4ServerConf()
	conf.Options = []string{
		"81 hex 303132",
		"82 ip 1.2.3.4",
	}

	s, err := v4Create(conf)
	require.NoError(t, err)

	s.conf.dnsIPAddrs = []netip.Addr{netip.MustParseAddr("192.168.10.1")}
	s.implicitOpts.Update(dhcpv4.OptDNS(s.conf.dnsIPAddrs[0].AsSlice()))

	var req, resp *dhcpv4.DHCPv4
	mac := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	t.Run("discover", func(t *testing.T) {
		req, err = dhcpv4.NewDiscovery(mac, dhcpv4.WithRequestedOptions(
			dhcpv4.OptionFQDN,
			dhcpv4.OptionRelayAgentInformation,
		))
		require.NoError(t, err)

		resp, err = dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		assert.Equal(t, 1, s.handle(req, resp))
	})

	// Don't continue if we got any errors in the previous subtest.
	require.NoError(t, err)

	t.Run("offer", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)

		assert.True(t, resp.YourIPAddr.Equal(s.conf.RangeStart.AsSlice()))
		assert.True(t, resp.ServerIdentifier().Equal(s.conf.GatewayIP.AsSlice()))

		router := resp.Router()
		require.Len(t, router, 1)

		assert.True(t, router[0].Equal(s.conf.GatewayIP.AsSlice()))

		ones, _ := resp.SubnetMask().Size()
		assert.Equal(t, s.conf.subnet.Bits(), ones)
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
		assert.Equal(t, []byte("012"), resp.Options.Get(dhcpv4.OptionFQDN))

		rai := resp.RelayAgentInfo()
		require.NotNil(t, rai)
		assert.Equal(t, net.IP{1, 2, 3, 4}, net.IP(rai.ToBytes()))
	})

	t.Run("request", func(t *testing.T) {
		req, err = dhcpv4.NewRequestFromOffer(resp)
		require.NoError(t, err)

		resp, err = dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		assert.Equal(t, 1, s.handle(req, resp))
	})

	require.NoError(t, err)

	t.Run("ack", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeAck, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)
		assert.True(t, resp.YourIPAddr.Equal(s.conf.RangeStart.AsSlice()))

		router := resp.Router()
		require.Len(t, router, 1)

		assert.True(t, router[0].Equal(s.conf.GatewayIP.AsSlice()))

		assert.True(t, resp.ServerIdentifier().Equal(s.conf.GatewayIP.AsSlice()))

		ones, _ := resp.SubnetMask().Size()
		assert.Equal(t, s.conf.subnet.Bits(), ones)
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	})

	dnsAddrs := resp.DNS()
	require.Len(t, dnsAddrs, 1)

	assert.True(t, net.IP{192, 168, 10, 1}.Equal(dnsAddrs[0]))

	// check lease
	t.Run("check_lease", func(t *testing.T) {
		ls := s.GetLeases(LeasesDynamic)
		require.Len(t, ls, 1)

		ip := netip.MustParseAddr("192.168.10.100")
		assert.Equal(t, ip, ls[0].IP)
		assert.Equal(t, mac, ls[0].HWAddr)
	})
}

func TestNormalizeHostname(t *testing.T) {
	testCases := []struct {
		name       string
		hostname   string
		wantErrMsg string
		want       string
	}{{
		name:       "success",
		hostname:   "example.com",
		wantErrMsg: "",
		want:       "example.com",
	}, {
		name:       "success_empty",
		hostname:   "",
		wantErrMsg: "",
		want:       "",
	}, {
		name:       "success_spaces",
		hostname:   "my device 01",
		wantErrMsg: "",
		want:       "my-device-01",
	}, {
		name:       "success_underscores",
		hostname:   "my_device_01",
		wantErrMsg: "",
		want:       "my-device-01",
	}, {
		name:       "error_part",
		hostname:   "device !!!",
		wantErrMsg: "",
		want:       "device",
	}, {
		name:       "error_part_spaces",
		hostname:   "device ! ! !",
		wantErrMsg: "",
		want:       "device",
	}, {
		name:       "error",
		hostname:   "!!!",
		wantErrMsg: `normalizing "!!!": no valid parts`,
		want:       "",
	}, {
		name:       "error_spaces",
		hostname:   "! ! !",
		wantErrMsg: `normalizing "! ! !": no valid parts`,
		want:       "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeHostname(tc.hostname)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// fakePacketConn is a mock implementation of net.PacketConn to simplify
// testing.
type fakePacketConn struct {
	// writeTo is used to substitute net.PacketConn's WriteTo method.
	writeTo func(p []byte, addr net.Addr) (n int, err error)
	// net.PacketConn is embedded here simply to make *fakePacketConn a
	// net.PacketConn without actually implementing all methods.
	net.PacketConn
}

// WriteTo implements net.PacketConn interface for *fakePacketConn.
func (fc *fakePacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return fc.writeTo(p, addr)
}

func TestV4Server_FindMACbyIP(t *testing.T) {
	const (
		staticName  = "static-client"
		anotherName = "another-client"
	)

	staticIP := netip.MustParseAddr("192.168.10.10")
	staticMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	anotherIP := netip.MustParseAddr("192.168.100.100")
	anotherMAC := net.HardwareAddr{0xBB, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}

	s := &v4Server{
		leases: []*dhcpsvc.Lease{{
			Hostname: staticName,
			HWAddr:   staticMAC,
			IP:       staticIP,
			IsStatic: true,
		}, {
			Expiry:   time.Unix(10, 0),
			Hostname: anotherName,
			HWAddr:   anotherMAC,
			IP:       anotherIP,
		}},
	}
	s.ipIndex = map[netip.Addr]*dhcpsvc.Lease{
		staticIP:  s.leases[0],
		anotherIP: s.leases[1],
	}
	s.hostsIndex = map[string]*dhcpsvc.Lease{
		staticName:  s.leases[0],
		anotherName: s.leases[1],
	}

	testCases := []struct {
		want net.HardwareAddr
		ip   netip.Addr
		name string
	}{{
		name: "basic",
		ip:   staticIP,
		want: staticMAC,
	}, {
		name: "not_found",
		ip:   netip.MustParseAddr("1.2.3.4"),
		want: nil,
	}, {
		name: "expired",
		ip:   anotherIP,
		want: nil,
	}, {
		name: "v6",
		ip:   netip.MustParseAddr("ffff::1"),
		want: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mac := s.FindMACbyIP(tc.ip)

			require.Equal(t, tc.want, mac)
		})
	}
}

func TestV4Server_handleDecline(t *testing.T) {
	const (
		dynamicName = "dynamic-client"
		anotherName = "another-client"
	)

	dynamicIP := netip.MustParseAddr("192.168.10.200")
	dynamicMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	s := defaultSrv(t)

	s4, ok := s.(*v4Server)
	require.True(t, ok)

	s4.leases = []*dhcpsvc.Lease{{
		Hostname: dynamicName,
		HWAddr:   dynamicMAC,
		IP:       dynamicIP,
	}}

	req, err := dhcpv4.New(
		dhcpv4.WithOption(dhcpv4.OptRequestedIPAddress(net.IP(dynamicIP.AsSlice()))),
	)
	require.NoError(t, err)

	req.ClientIPAddr = net.IP(dynamicIP.AsSlice())
	req.ClientHWAddr = dynamicMAC

	resp := &dhcpv4.DHCPv4{}
	err = s4.handleDecline(req, resp)
	require.NoError(t, err)

	wantResp := &dhcpv4.DHCPv4{
		YourIPAddr: net.IP(s4.conf.RangeStart.AsSlice()),
		Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeAck),
		),
	}

	require.Equal(t, wantResp, resp)
}

func TestV4Server_handleRelease(t *testing.T) {
	const (
		dynamicName = "dynamic-client"
		anotherName = "another-client"
	)

	dynamicIP := netip.MustParseAddr("192.168.10.200")
	dynamicMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

	s := defaultSrv(t)

	s4, ok := s.(*v4Server)
	require.True(t, ok)

	s4.leases = []*dhcpsvc.Lease{{
		Hostname: dynamicName,
		HWAddr:   dynamicMAC,
		IP:       dynamicIP,
	}}

	req, err := dhcpv4.New(
		dhcpv4.WithOption(dhcpv4.OptRequestedIPAddress(net.IP(dynamicIP.AsSlice()))),
	)
	require.NoError(t, err)

	req.ClientIPAddr = net.IP(dynamicIP.AsSlice())
	req.ClientHWAddr = dynamicMAC

	resp := &dhcpv4.DHCPv4{}
	err = s4.handleRelease(req, resp)
	require.NoError(t, err)

	wantResp := &dhcpv4.DHCPv4{
		Options: dhcpv4.OptionsFromList(
			dhcpv4.OptMessageType(dhcpv4.MessageTypeAck),
		),
	}

	require.Equal(t, wantResp, resp)
}
