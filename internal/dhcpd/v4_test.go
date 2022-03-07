//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"net"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/mdlayher/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notify4(flags uint32) {
}

// defaultV4ServerConf returns the default configuration for *v4Server to use in
// tests.
func defaultV4ServerConf() (conf V4ServerConf) {
	return V4ServerConf{
		Enabled:    true,
		RangeStart: net.IP{192, 168, 10, 100},
		RangeEnd:   net.IP{192, 168, 10, 200},
		GatewayIP:  net.IP{192, 168, 10, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		notify:     notify4,
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

func TestV4_AddRemove_static(t *testing.T) {
	s := defaultSrv(t)

	ls := s.GetLeases(LeasesStatic)
	assert.Empty(t, ls)

	// Add static lease.
	l := &Lease{
		Hostname: "static-1.local",
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       net.IP{192, 168, 10, 150},
	}

	err := s.AddStaticLease(l)
	require.NoError(t, err)

	err = s.AddStaticLease(l)
	assert.Error(t, err)

	ls = s.GetLeases(LeasesStatic)
	require.Len(t, ls, 1)

	assert.True(t, l.IP.Equal(ls[0].IP))
	assert.Equal(t, l.HWAddr, ls[0].HWAddr)
	assert.True(t, ls[0].IsStatic())

	// Try to remove static lease.
	err = s.RemoveStaticLease(&Lease{
		IP:     net.IP{192, 168, 10, 110},
		HWAddr: net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
	})
	assert.Error(t, err)

	// Remove static lease.
	err = s.RemoveStaticLease(l)
	require.NoError(t, err)
	ls = s.GetLeases(LeasesStatic)
	assert.Empty(t, ls)
}

func TestV4_AddReplace(t *testing.T) {
	sIface := defaultSrv(t)

	s, ok := sIface.(*v4Server)
	require.True(t, ok)

	dynLeases := []Lease{{
		Hostname: "dynamic-1.local",
		HWAddr:   net.HardwareAddr{0x11, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       net.IP{192, 168, 10, 150},
	}, {
		Hostname: "dynamic-2.local",
		HWAddr:   net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       net.IP{192, 168, 10, 151},
	}}

	for i := range dynLeases {
		err := s.addLease(&dynLeases[i])
		require.NoError(t, err)
	}

	stLeases := []*Lease{{
		Hostname: "static-1.local",
		HWAddr:   net.HardwareAddr{0x33, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       net.IP{192, 168, 10, 150},
	}, {
		Hostname: "static-2.local",
		HWAddr:   net.HardwareAddr{0x22, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       net.IP{192, 168, 10, 152},
	}}

	for _, l := range stLeases {
		err := s.AddStaticLease(l)
		require.NoError(t, err)
	}

	ls := s.GetLeases(LeasesStatic)
	require.Len(t, ls, 2)

	for i, l := range ls {
		assert.True(t, stLeases[i].IP.Equal(l.IP))
		assert.Equal(t, stLeases[i].HWAddr, l.HWAddr)
		assert.True(t, l.IsStatic())
	}
}

func TestV4Server_Process_optionsPriority(t *testing.T) {
	defaultIP := net.IP{192, 168, 1, 1}
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
		}

		ss, err := v4Create(conf)
		require.NoError(t, err)

		var ok bool
		s, ok = ss.(*v4Server)
		require.True(t, ok)

		s.conf.dnsIPAddrs = []net.IP{defaultIP}

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

		res := s.process(req, resp)
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

		checkResp(t, s, []net.IP{defaultIP})
	})

	t.Run("explicitly_configured", func(t *testing.T) {
		s := prepareSrv(t, []net.IP{knownIP, knownIP})

		checkResp(t, s, []net.IP{knownIP, knownIP})
	})
}

func TestV4StaticLease_Get(t *testing.T) {
	sIface := defaultSrv(t)

	s, ok := sIface.(*v4Server)
	require.True(t, ok)

	s.conf.dnsIPAddrs = []net.IP{{192, 168, 10, 1}}

	l := &Lease{
		Hostname: "static-1.local",
		HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
		IP:       net.IP{192, 168, 10, 150},
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

		assert.Equal(t, 1, s.process(req, resp))
	})

	// Don't continue if we got any errors in the previous subtest.
	require.NoError(t, err)

	t.Run("offer", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)
		assert.True(t, l.IP.Equal(resp.YourIPAddr))
		assert.True(t, s.conf.GatewayIP.Equal(resp.Router()[0]))
		assert.True(t, s.conf.GatewayIP.Equal(resp.ServerIdentifier()))
		assert.Equal(t, s.conf.subnet.Mask, resp.SubnetMask())
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	})

	t.Run("request", func(t *testing.T) {
		req, err = dhcpv4.NewRequestFromOffer(resp)
		require.NoError(t, err)

		resp, err = dhcpv4.NewReplyFromRequest(req)
		require.NoError(t, err)

		assert.Equal(t, 1, s.process(req, resp))
	})

	require.NoError(t, err)

	t.Run("ack", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeAck, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)
		assert.True(t, l.IP.Equal(resp.YourIPAddr))
		assert.True(t, s.conf.GatewayIP.Equal(resp.Router()[0]))
		assert.True(t, s.conf.GatewayIP.Equal(resp.ServerIdentifier()))
		assert.Equal(t, s.conf.subnet.Mask, resp.SubnetMask())
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	})

	dnsAddrs := resp.DNS()
	require.Len(t, dnsAddrs, 1)

	assert.True(t, s.conf.GatewayIP.Equal(dnsAddrs[0]))

	t.Run("check_lease", func(t *testing.T) {
		ls := s.GetLeases(LeasesStatic)
		require.Len(t, ls, 1)

		assert.True(t, l.IP.Equal(ls[0].IP))
		assert.Equal(t, mac, ls[0].HWAddr)
	})
}

func TestV4DynamicLease_Get(t *testing.T) {
	conf := defaultV4ServerConf()
	conf.Options = []string{
		"81 hex 303132",
		"82 ip 1.2.3.4",
	}

	var err error
	sIface, err := v4Create(conf)
	require.NoError(t, err)

	s, ok := sIface.(*v4Server)
	require.True(t, ok)

	s.conf.dnsIPAddrs = []net.IP{{192, 168, 10, 1}}

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

		assert.Equal(t, 1, s.process(req, resp))
	})

	// Don't continue if we got any errors in the previous subtest.
	require.NoError(t, err)

	t.Run("offer", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeOffer, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)

		assert.Equal(t, s.conf.RangeStart, resp.YourIPAddr)
		assert.Equal(t, s.conf.GatewayIP, resp.ServerIdentifier())

		router := resp.Router()
		require.Len(t, router, 1)

		assert.Equal(t, s.conf.GatewayIP, router[0])

		assert.Equal(t, s.conf.subnet.Mask, resp.SubnetMask())
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

		assert.Equal(t, 1, s.process(req, resp))
	})

	require.NoError(t, err)

	t.Run("ack", func(t *testing.T) {
		assert.Equal(t, dhcpv4.MessageTypeAck, resp.MessageType())
		assert.Equal(t, mac, resp.ClientHWAddr)
		assert.True(t, s.conf.RangeStart.Equal(resp.YourIPAddr))

		router := resp.Router()
		require.Len(t, router, 1)

		assert.Equal(t, s.conf.GatewayIP, router[0])

		assert.True(t, s.conf.GatewayIP.Equal(resp.ServerIdentifier()))
		assert.Equal(t, s.conf.subnet.Mask, resp.SubnetMask())
		assert.Equal(t, s.conf.leaseTime.Seconds(), resp.IPAddressLeaseTime(-1).Seconds())
	})

	dnsAddrs := resp.DNS()
	require.Len(t, dnsAddrs, 1)

	assert.True(t, net.IP{192, 168, 10, 1}.Equal(dnsAddrs[0]))

	// check lease
	t.Run("check_lease", func(t *testing.T) {
		ls := s.GetLeases(LeasesDynamic)
		require.Len(t, ls, 1)

		assert.True(t, net.IP{192, 168, 10, 100}.Equal(ls[0].IP))
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

func TestV4Server_Send(t *testing.T) {
	s := &v4Server{}

	var (
		defaultIP = net.IP{99, 99, 99, 99}
		knownIP   = net.IP{4, 2, 4, 2}
		knownMAC  = net.HardwareAddr{6, 5, 4, 3, 2, 1}
	)

	defaultPeer := &net.UDPAddr{
		IP: defaultIP,
		// Use neither client nor server port to check it actually
		// changed.
		Port: dhcpv4.ClientPort + dhcpv4.ServerPort,
	}
	defaultResp := &dhcpv4.DHCPv4{}

	testCases := []struct {
		want net.Addr
		req  *dhcpv4.DHCPv4
		resp *dhcpv4.DHCPv4
		name string
	}{{
		name: "giaddr",
		req:  &dhcpv4.DHCPv4{GatewayIPAddr: knownIP},
		resp: defaultResp,
		want: &net.UDPAddr{
			IP:   knownIP,
			Port: dhcpv4.ServerPort,
		},
	}, {
		name: "nak",
		req:  &dhcpv4.DHCPv4{},
		resp: &dhcpv4.DHCPv4{
			Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeNak),
			),
		},
		want: defaultPeer,
	}, {
		name: "ciaddr",
		req:  &dhcpv4.DHCPv4{ClientIPAddr: knownIP},
		resp: &dhcpv4.DHCPv4{},
		want: &net.UDPAddr{
			IP:   knownIP,
			Port: dhcpv4.ClientPort,
		},
	}, {
		name: "chaddr",
		req:  &dhcpv4.DHCPv4{ClientHWAddr: knownMAC},
		resp: &dhcpv4.DHCPv4{YourIPAddr: knownIP},
		want: &dhcpUnicastAddr{
			Addr:   raw.Addr{HardwareAddr: knownMAC},
			yiaddr: knownIP,
		},
	}, {
		name: "who_are_you",
		req:  &dhcpv4.DHCPv4{},
		resp: &dhcpv4.DHCPv4{},
		want: defaultPeer,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &fakePacketConn{
				writeTo: func(_ []byte, addr net.Addr) (_ int, _ error) {
					assert.Equal(t, tc.want, addr)

					return 0, nil
				},
			}

			s.send(cloneUDPAddr(defaultPeer), conn, tc.req, tc.resp)
		})
	}

	t.Run("giaddr_nak", func(t *testing.T) {
		req := &dhcpv4.DHCPv4{
			GatewayIPAddr: knownIP,
		}
		// Ensure the request is for unicast.
		req.SetUnicast()
		resp := &dhcpv4.DHCPv4{
			Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeNak),
			),
		}
		want := &net.UDPAddr{
			IP:   req.GatewayIPAddr,
			Port: dhcpv4.ServerPort,
		}

		conn := &fakePacketConn{
			writeTo: func(_ []byte, addr net.Addr) (n int, err error) {
				assert.Equal(t, want, addr)

				return 0, nil
			},
		}

		s.send(cloneUDPAddr(defaultPeer), conn, req, resp)
		assert.True(t, resp.IsBroadcast())
	})
}
