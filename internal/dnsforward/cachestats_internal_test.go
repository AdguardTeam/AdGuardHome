package dnsforward

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyCacheStats(t *testing.T) {
	t.Parallel()

	const host = "cache.example."

	ups := aghtest.NewUpstream()
	ups.OnExchange = func(_ context.Context, req *dns.Msg) (resp *dns.Msg, err error) {
		resp = (&dns.Msg{}).SetReply(req)
		resp.Answer = []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: net.IP{192, 0, 2, 1},
		}}

		return resp, nil
	}

	cliAddr := netip.MustParseAddrPort("192.0.2.2:53")

	testCases := []struct {
		name        string
		cacheOn     bool
		ecsOn       bool
		wantEntries uint64
	}{{
		name:        "disabled",
		cacheOn:     false,
		ecsOn:       false,
		wantEntries: 0,
	}, {
		name:        "enabled",
		cacheOn:     true,
		ecsOn:       false,
		wantEntries: 1,
	}, {
		name:        "enabled_ecs",
		cacheOn:     true,
		ecsOn:       true,
		wantEntries: 1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := proxy.New(&proxy.Config{
				Logger: testLogger,
				UpstreamConfig: &proxy.UpstreamConfig{
					Upstreams: []upstream.Upstream{ups},
				},
				CacheEnabled:           tc.cacheOn,
				CacheSizeBytes:         64 * 1024,
				EnableEDNSClientSubnet: tc.ecsOn,
			})
			require.NoError(t, err)

			cs, err := proxyCacheStats(p)
			require.NoError(t, err)

			assert.Zero(t, cs)

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			for range 2 {
				req := (&dns.Msg{}).SetQuestion(host, dns.TypeA)
				err = p.Resolve(ctx, &proxy.DNSContext{
					Req:  req,
					Addr: cliAddr,
				})
				require.NoError(t, err)
			}

			cs, err = proxyCacheStats(p)
			require.NoError(t, err)

			assert.Equal(t, tc.wantEntries, cs.entries)
			if tc.wantEntries > 0 {
				assert.Positive(t, cs.size)
			} else {
				assert.Zero(t, cs.size)
			}
		})
	}
}
