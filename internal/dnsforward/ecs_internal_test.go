package dnsforward

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
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
			return newTestMsgWithECS(t, 1, 24, "1.2.3.4")
		},
		wantOK: true,
		wantIP: netip.MustParseAddr("1.2.3.4"),
	}, {
		name: "ecs_v6",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 2, 56, "2001:db8::1")
		},
		wantOK: true,
		wantIP: netip.MustParseAddr("2001:db8::1"),
	}, {
		name: "ecs_unknown_family",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 3, 24, "1.2.3.4")
		},
	}, {
		name: "ecs_unspecified",
		msg: func(t *testing.T) (m *dns.Msg) {
			return newTestMsgWithECS(t, 1, 24, "0.0.0.0")
		},
	}, {
		name: "ecs_invalid_v4",
		msg: func(t *testing.T) (m *dns.Msg) {
			// A v6 address advertised as an IPv4 ECS option should be skipped.
			return newTestMsgWithECS(t, 1, 24, "2001:db8::1")
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
