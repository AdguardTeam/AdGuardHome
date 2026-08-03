package dnsforward

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

func TestServer_Resolve_systemHosts(t *testing.T) {
	const (
		host                 = "filter.example"
		upstreamOnlyHost     = "upstream.example"
		dualStackHost        = "dual-stack.example"
		upstreamIPv4OnlyHost = "upstream-ipv4.example"
		upstreamIPv6OnlyHost = "upstream-ipv6.example"
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	hosts, err := hostsfile.NewDefaultStorage(ctx, &hostsfile.DefaultStorageConfig{
		Logger: testLogger,
	})
	require.NoError(t, err)

	hostsIPv4 := netip.MustParseAddr("192.0.2.1")
	hostsIPv6 := netip.MustParseAddr("2001:db8::1")
	hosts.Add(ctx, &hostsfile.Record{
		Addr:  hostsIPv4,
		Names: []string{host},
	})
	hosts.Add(ctx, &hostsfile.Record{
		Addr:  hostsIPv4,
		Names: []string{dualStackHost, upstreamIPv6OnlyHost},
	})
	hosts.Add(ctx, &hostsfile.Record{
		Addr:  hostsIPv6,
		Names: []string{dualStackHost, upstreamIPv4OnlyHost},
	})

	upstreamIPv4 := netip.MustParseAddr("192.0.2.2")
	upstreamIPv6 := netip.MustParseAddr("2001:db8::2")
	internalProxy, err := proxy.New(&proxy.Config{
		UpstreamConfig: &proxy.UpstreamConfig{
			Upstreams: []upstream.Upstream{&aghtest.Upstream{
				IPv4: map[string][]net.IP{
					host + ".":                 {upstreamIPv4.AsSlice()},
					upstreamOnlyHost + ".":     {upstreamIPv4.AsSlice()},
					upstreamIPv4OnlyHost + ".": {upstreamIPv4.AsSlice()},
				},
				IPv6: map[string][]net.IP{
					upstreamIPv6OnlyHost + ".": {upstreamIPv6.AsSlice()},
				},
			}},
		},
	})
	require.NoError(t, err)

	s := &Server{
		etcHosts:      upstream.NewHostsResolver(hosts),
		internalProxy: internalProxy,
	}

	testCases := []struct {
		name    string
		network string
		host    string
		want    []netip.Addr
	}{
		{
			name:    "system_hosts",
			network: "tcp",
			host:    host,
			want:    []netip.Addr{hostsIPv4},
		},
		{
			name:    "upstream_fallback",
			network: "tcp",
			host:    upstreamOnlyHost,
			want:    []netip.Addr{upstreamIPv4},
		},
		{
			name:    "tcp_dual_stack_system_hosts",
			network: "tcp",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv4, hostsIPv6},
		},
		{
			name:    "tcp4_system_hosts",
			network: "tcp4",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv4},
		},
		{
			name:    "udp4_system_hosts",
			network: "udp4",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv4},
		},
		{
			name:    "ip4_system_hosts",
			network: "ip4",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv4},
		},
		{
			name:    "tcp6_system_hosts",
			network: "tcp6",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv6},
		},
		{
			name:    "udp6_system_hosts",
			network: "udp6",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv6},
		},
		{
			name:    "ip6_system_hosts",
			network: "ip6",
			host:    dualStackHost,
			want:    []netip.Addr{hostsIPv6},
		},
		{
			name:    "tcp4_opposite_family_fallback",
			network: "tcp4",
			host:    upstreamIPv4OnlyHost,
			want:    []netip.Addr{upstreamIPv4},
		},
		{
			name:    "tcp6_opposite_family_fallback",
			network: "tcp6",
			host:    upstreamIPv6OnlyHost,
			want:    []netip.Addr{upstreamIPv6},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addrs, resolveErr := s.Resolve(ctx, tc.network, tc.host)
			require.NoError(t, resolveErr)
			require.Equal(t, tc.want, addrs)
		})
	}
}
