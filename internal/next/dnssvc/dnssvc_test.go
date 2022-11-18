package dnssvc_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// testTimeout is the common timeout for tests.
const testTimeout = 100 * time.Millisecond

func TestService(t *testing.T) {
	const (
		bootstrapAddr = "bootstrap.example"
		upstreamAddr  = "upstream.example"

		closeErr errors.Error = "closing failed"
	)

	ups := &aghtest.UpstreamMock{
		OnAddress: func() (addr string) {
			return upstreamAddr
		},
		OnExchange: func(req *dns.Msg) (resp *dns.Msg, err error) {
			resp = (&dns.Msg{}).SetReply(req)

			return resp, nil
		},
		OnClose: func() (err error) {
			return closeErr
		},
	}

	c := &dnssvc.Config{
		Addresses:        []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:0")},
		Upstreams:        []upstream.Upstream{ups},
		BootstrapServers: []string{bootstrapAddr},
		UpstreamServers:  []string{upstreamAddr},
		UpstreamTimeout:  testTimeout,
	}

	svc, err := dnssvc.New(c)
	require.NoError(t, err)

	err = svc.Start()
	require.NoError(t, err)

	gotConf := svc.Config()
	require.NotNil(t, gotConf)
	require.Len(t, gotConf.Addresses, 1)

	addr := gotConf.Addresses[0]

	t.Run("dns", func(t *testing.T) {
		req := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id:               dns.Id(),
				RecursionDesired: true,
			},
			Question: []dns.Question{{
				Name:   "example.com.",
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			}},
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		cli := &dns.Client{}
		resp, _, excErr := cli.ExchangeContext(ctx, req, addr.String())
		require.NoError(t, excErr)

		assert.NotNil(t, resp)
	})

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err = svc.Shutdown(ctx)
	require.ErrorIs(t, err, closeErr)
}
