package dnsforward

import (
	"context"
	"net/netip"
	"slices"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyNameMatches(t *testing.T) {
	dnsNames := []string{"host1", "*.host2", "1.2.3.4"}
	slices.Sort(dnsNames)

	testCases := []struct {
		name    string
		dnsName string
		want    bool
	}{{
		name:    "match",
		dnsName: "host1",
		want:    true,
	}, {
		name:    "match",
		dnsName: "a.host2",
		want:    true,
	}, {
		name:    "match",
		dnsName: "b.a.host2",
		want:    true,
	}, {
		name:    "match",
		dnsName: "1.2.3.4",
		want:    true,
	}, {
		name:    "mismatch_bad_ip",
		dnsName: "1.2.3.256",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "host2",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "",
		want:    false,
	}, {
		name:    "mismatch",
		dnsName: "*.host2",
		want:    false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, anyNameMatches(dnsNames, tc.dnsName))
		})
	}
}

func TestNewRatelimitMw_ratelimitWhitelist(t *testing.T) {
	t.Parallel()

	const limit = 1

	addr := netip.MustParseAddr("192.0.2.1")
	mw, err := newRatelimitMw(testLogger, ServerConfig{
		Config: Config{
			Ratelimit:              limit,
			RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
			RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
			RatelimitWhitelist:     []netip.Addr{addr},
		},
	})
	require.NoError(t, err)

	called := 0
	handler := mw.Wrap(proxy.HandlerFunc(func(
		_ context.Context,
		_ *proxy.Proxy,
		_ *proxy.DNSContext,
	) (err error) {
		called++

		return nil
	}))

	dctx := &proxy.DNSContext{
		Addr:  netip.AddrPortFrom(addr, 53),
		Proto: proxy.ProtoUDP,
	}
	for i := 0; i < limit+1; i++ {
		err = handler.ServeDNS(context.Background(), nil, dctx)
		require.NoError(t, err)
	}

	assert.Equal(t, limit+1, called)
}
