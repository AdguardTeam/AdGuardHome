package dnsforward

import (
	"context"
	"net/netip"
	"slices"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
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

func TestNewRatelimitMw_Whitelist(t *testing.T) {
	t.Parallel()

	handler := proxy.HandlerFunc(
		func(ctx context.Context, p *proxy.Proxy, dctx *proxy.DNSContext) (err error) {
			dctx.Res = newResp(dns.RcodeSuccess, dctx.Req, nil)

			return nil
		},
	)

	testCases := []struct {
		name       string
		conf       ServerConfig
		req        netip.Addr
		wantDrops  []bool
		wantErrMsg string
	}{{
		name: "disabled_ratelimit",
		conf: ServerConfig{
			Config: Config{
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
			},
		},
		req:       netip.MustParseAddr("192.0.2.1"),
		wantDrops: []bool{false, false, false},
	}, {
		name: "ratelimit_without_whitelist",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
			},
		},
		req:       netip.MustParseAddr("192.0.2.1"),
		wantDrops: []bool{false, true, true},
	}, {
		name: "ratelimit_whitelisted_v4",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{netip.MustParseAddr("198.51.100.7")},
			},
		},
		req:       netip.MustParseAddr("198.51.100.7"),
		wantDrops: []bool{false, false, false, false},
	}, {
		name: "ratelimit_whitelisted_v6",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{netip.MustParseAddr("2001:db8::7")},
			},
		},
		req:       netip.MustParseAddr("2001:db8::7"),
		wantDrops: []bool{false, false, false, false},
	}, {
		name: "ratelimit_whitelist_other_ip",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{netip.MustParseAddr("198.51.100.7")},
			},
		},
		req:       netip.MustParseAddr("192.0.2.1"),
		wantDrops: []bool{false, true},
	}, {
		name: "invalid_whitelist_ip",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{{}},
			},
		},
		wantErrMsg: `ratelimit whitelist ip at index 0: ParseAddr("invalid IP"): ` +
			`unable to parse IP`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mw, err := newRatelimitMw(testLogger, tc.conf)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			if tc.wantErrMsg != "" {
				return
			}

			wrapped := mw.Wrap(handler)
			handleDrops(t, wrapped, tc.req, tc.wantDrops)
		})
	}
}

// handleDrops handles a series of DNS requests and checks if they are dropped
// according to the wantDrops slice.  wrapped must not be nil.
func handleDrops(tb testing.TB, wrapped proxy.Handler, addr netip.Addr, wantDrops []bool) {
	const testPort = 1

	for i, wantDrop := range wantDrops {
		dctx := &proxy.DNSContext{
			Proto: proxy.ProtoUDP,
			Addr:  netip.AddrPortFrom(addr, testPort),
			Req:   createTestMessage(testQuestionTarget),
		}

		err := wrapped.ServeDNS(testutil.ContextWithTimeout(tb, testTimeout), nil, dctx)
		if !assert.Equalf(tb, wantDrop, err == proxy.ErrDrop, "request #%d", i) {
			continue
		}
		if wantDrop {
			continue
		}

		require.NoError(tb, err)
		assert.NotNil(tb, dctx.Res)
	}
}
