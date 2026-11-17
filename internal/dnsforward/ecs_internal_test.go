package dnsforward

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestECSClientAddr is a test for the ecsClientAddr function.
func TestECSClientAddr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		msg    func(t *testing.T) *dns.Msg
		wantOK bool
		wantIP netip.Addr
	}{{
		name: "no_opt",
		msg: func(t *testing.T) (m *dns.Msg) {
			return new(dns.Msg)
		},
	}, {
		name: "opt_without_ecs",
		msg: func(t *testing.T) (m *dns.Msg) {
			o := new(dns.OPT)
			o.Hdr.Name = "."
			o.Hdr.Rrtype = dns.TypeOPT
			m = new(dns.Msg)
			m.Extra = append(m.Extra, o)
			return m
		},
	}, {
		name: "ecs_v4",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 1, 32, "1.2.3.4")
		},
		wantOK: true,
		wantIP: netip.MustParseAddr("1.2.3.4"),
	}, {
		name: "ecs_v6",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 2, 128, "2001:db8::1")
		},
		wantOK: true,
		wantIP: netip.MustParseAddr("2001:db8::1"),
	}, {
		name: "ecs_v4_subnet",
		msg: func(t *testing.T) (m *dns.Msg) {
			// A subnet doesn't identify a single client.
			return newTestMsgWithECS(t, 1, 24, "1.2.3.0")
		},
	}, {
		name: "ecs_v6_subnet",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 2, 56, "2001:db8::")
		},
	}, {
		name: "ecs_unknown_family",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 3, 32, "1.2.3.4")
		},
	}, {
		name: "ecs_unspecified",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 1, 32, "0.0.0.0")
		},
	}, {
		name: "ecs_invalid_v4",
		msg: func(t *testing.T) (m *dns.Msg) {
			// A v6 address advertised as an IPv4 ECS option should be skipped.
			return newTestMsgWithECS(t, 1, 32, "2001:db8::1")
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			addr, ok := ecsClientAddr(tc.msg(t))
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantIP, addr)
			} else {
				assert.False(t, addr.IsValid())
			}
		})
	}
}

// newTestMsgWithECS returns a new DNS message with an EDNS Client Subnet
// option using the given family, netmask and address.
func newTestMsgWithECS(t *testing.T, family uint16, netmask uint8, addr string) (m *dns.Msg) {
	t.Helper()

	ip := net.ParseIP(addr)
	require.NotNil(t, ip)

	o := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
		},
	}
	o.SetUDPSize(4096)

	e := &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        family,
		SourceNetmask: netmask,
		SourceScope:   0,
		Address:       ip,
	}
	o.Option = append(o.Option, e)

	m = new(dns.Msg)
	m.Extra = append(m.Extra, o)

	return m
}

// TestDNSContext_clientAddr is a test for the clientAddr method.
func TestDNSContext_clientAddr(t *testing.T) {
	t.Parallel()

	const (
		connAddr = "192.168.1.1"
		ecsAddr  = "192.168.1.100"
	)

	dctx := &dnsContext{
		proxyCtx: &proxy.DNSContext{
			Addr: netip.AddrPortFrom(netip.MustParseAddr(connAddr), 53),
		},
	}

	// Without ECS, the connection address is used.
	assert.Equal(t, netip.MustParseAddr(connAddr), dctx.clientAddr())

	// With ECS set, it takes precedence.
	dctx.ecsClientAddr = netip.MustParseAddr(ecsAddr)
	assert.Equal(t, netip.MustParseAddr(ecsAddr), dctx.clientAddr())

	// Clearing it falls back again.
	dctx.ecsClientAddr = netip.Addr{}
	assert.Equal(t, netip.MustParseAddr(connAddr), dctx.clientAddr())
}

// TestServer_withECSClientAddr is a test for the withECSClientAddr method.
func TestServer_withECSClientAddr(t *testing.T) {
	t.Parallel()

	var (
		proxyAddr     = netip.MustParseAddr("192.0.2.1")
		otherAddr     = netip.MustParseAddr("192.0.2.2")
		ecsAddr       = netip.MustParseAddr("192.0.2.100")
		trustedPrefix = netip.MustParsePrefix("192.0.2.1/32")
	)

	testCases := []struct {
		remote  netip.Addr
		req     *dns.Msg
		want    netip.Addr
		name    string
		enabled bool
	}{{
		remote:  proxyAddr,
		req:     newTestMsgWithECS(t, 1, 32, ecsAddr.String()),
		want:    ecsAddr,
		name:    "trusted",
		enabled: true,
	}, {
		remote:  netip.AddrFrom16(proxyAddr.As16()),
		req:     newTestMsgWithECS(t, 1, 32, ecsAddr.String()),
		want:    ecsAddr,
		name:    "trusted_mapped",
		enabled: true,
	}, {
		remote:  proxyAddr,
		req:     newTestMsgWithECS(t, 1, 32, ecsAddr.String()),
		want:    proxyAddr,
		name:    "disabled",
		enabled: false,
	}, {
		remote:  otherAddr,
		req:     newTestMsgWithECS(t, 1, 32, ecsAddr.String()),
		want:    otherAddr,
		name:    "untrusted",
		enabled: true,
	}, {
		remote:  proxyAddr,
		req:     newTestMsgWithECS(t, 1, 24, "192.0.2.0"),
		want:    proxyAddr,
		name:    "trusted_subnet",
		enabled: true,
	}, {
		remote:  proxyAddr,
		req:     new(dns.Msg),
		want:    proxyAddr,
		name:    "trusted_no_ecs",
		enabled: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Server{
				conf: ServerConfig{
					Config: Config{
						EDNSClientSubnet: &EDNSClientSubnet{
							UseClientAddrFromECS: tc.enabled,
						},
						TrustedProxies: []netutil.Prefix{{Prefix: trustedPrefix}},
					},
				},
			}

			pctx := &proxy.DNSContext{
				Req:  tc.req,
				Addr: netip.AddrPortFrom(tc.remote, 53),
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			ctx, got := s.withECSClientAddr(ctx, pctx)
			assert.Equal(t, tc.want, got)

			ctxAddr, ok := ecsClientAddrFromContext(ctx)
			if tc.want == tc.remote {
				assert.False(t, ok)
			} else {
				require.True(t, ok)
				assert.Equal(t, tc.want, ctxAddr)
			}
		})
	}
}

// TestServer_setCustomUpstream_ecs checks that the custom upstream
// configuration is looked up by the client address taken from ECS.
func TestServer_setCustomUpstream_ecs(t *testing.T) {
	t.Parallel()

	var (
		proxyAddr = netip.MustParseAddr("192.0.2.1")
		ecsAddr   = netip.MustParseAddr("192.0.2.100")
	)

	wantConf := &proxy.CustomUpstreamConfig{}

	var gotAddr netip.Addr
	s := &Server{
		conf: ServerConfig{
			Config: Config{
				ClientsContainer: &clientsContainer{
					OnCustomUpstreamConfig: func(
						_ string,
						cliAddr netip.Addr,
					) (conf *proxy.CustomUpstreamConfig) {
						gotAddr = cliAddr

						return wantConf
					},
				},
			},
		},
	}

	dctx := &dnsContext{
		proxyCtx: &proxy.DNSContext{
			Addr: netip.AddrPortFrom(proxyAddr, 53),
		},
		ecsClientAddr: ecsAddr,
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	s.setCustomUpstream(ctx, testLogger, dctx.proxyCtx, "", dctx.clientAddr())

	assert.Equal(t, ecsAddr, gotAddr)
	assert.Same(t, wantConf, dctx.proxyCtx.CustomUpstreamConfig)
}
