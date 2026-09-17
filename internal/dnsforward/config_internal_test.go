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

func TestMiddleware_Wrap(t *testing.T) {
	t.Parallel()

	handler := proxy.HandlerFunc(
		func(ctx context.Context, p *proxy.Proxy, dctx *proxy.DNSContext) (err error) {
			dctx.Res = newResp(dns.RcodeSuccess, dctx.Req, nil)

			return nil
		},
	)

	testCases := []struct {
		name          string
		conf          ServerConfig
		req           netip.Addr
		wantErrMsg    string
		wantFirstDrop int
		attemptNum    int
	}{{
		name: "disabled_ratelimit",
		conf: ServerConfig{
			Config: Config{
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
			},
		},
		req:           netip.MustParseAddr("192.0.2.1"),
		attemptNum:    3,
		wantFirstDrop: 3,
		wantErrMsg:    "",
	}, {
		name: "ratelimit_without_whitelist",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
			},
		},
		req:           netip.MustParseAddr("192.0.2.1"),
		attemptNum:    3,
		wantFirstDrop: 1,
		wantErrMsg:    "",
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
		req:           netip.MustParseAddr("198.51.100.7"),
		attemptNum:    4,
		wantFirstDrop: 4,
		wantErrMsg:    "",
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
		req:           netip.MustParseAddr("2001:db8::7"),
		attemptNum:    4,
		wantFirstDrop: 4,
		wantErrMsg:    "",
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
		req:           netip.MustParseAddr("192.0.2.1"),
		attemptNum:    2,
		wantFirstDrop: 1,
		wantErrMsg:    "",
	}, {
		name: "ratelimit_whitelisted_mapped_v4",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{netip.MustParseAddr("::ffff:198.51.100.7")},
			},
		},
		req:           netip.MustParseAddr("198.51.100.7"),
		attemptNum:    4,
		wantFirstDrop: 4,
		wantErrMsg:    "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mw, err := newRatelimitMw(testLogger, tc.conf)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			if tc.wantErrMsg != "" {
				return
			}

			wrappedHdlr := mw.Wrap(handler)
			handleDrops(t, wrappedHdlr, tc.req, tc.attemptNum, tc.wantFirstDrop)
		})
	}
}

func TestMiddleware_Wrap_errors(t *testing.T) {
	t.Parallel()

	const invalidIPErrorMsg = "ratelimit: whitelist: at index 0: invalid ip"

	testCases := []struct {
		name       string
		conf       ServerConfig
		req        netip.Addr
		wantDrops  []bool
		wantErrMsg string
	}{{
		name: "invalid_whitelist_ip",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{{}},
			},
		},
		wantErrMsg: invalidIPErrorMsg,
	}, {
		name: "empty_whitelist_ip",
		conf: ServerConfig{
			Config: Config{
				Ratelimit:              1,
				RatelimitSubnetLenIPv4: netutil.IPv4BitLen,
				RatelimitSubnetLenIPv6: netutil.IPv6BitLen,
				RatelimitWhitelist:     []netip.Addr{{}},
			},
		},
		wantErrMsg: invalidIPErrorMsg,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := newRatelimitMw(testLogger, tc.conf)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

// handleDrops handles a series of DNS requests and checks if they are dropped
// according to the wantDrops slice.  handler must not be nil.
func handleDrops(
	tb testing.TB,
	handler proxy.Handler,
	addr netip.Addr,
	attemptNum int,
	wantFirstDrop int,
) {
	const testPort = 1

	for i := 0; i < attemptNum; i++ {
		dctx := &proxy.DNSContext{
			Proto: proxy.ProtoUDP,
			Addr:  netip.AddrPortFrom(addr, testPort),
			Req:   createTestMessage(testQuestionTarget),
		}

		err := handler.ServeDNS(testutil.ContextWithTimeout(tb, testTimeout), nil, dctx)
		if i >= wantFirstDrop {
			assert.ErrorIsf(tb, err, proxy.ErrDrop, "request %d", i)

			continue
		}
		require.NoError(tb, err)

		assert.NotNil(tb, dctx.Res)
	}
}
