package dnssvc_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

func TestService(t *testing.T) {
	const (
		listenAddr    = "127.0.0.1:0"
		bootstrapAddr = "127.0.0.1:0"
		upstreamAddr  = "upstream.example"
	)

	upstreamErrCh := make(chan error, 1)
	upstreamStartedCh := make(chan struct{})
	upstreamSrv := &dns.Server{
		Addr: bootstrapAddr,
		Net:  "udp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			pt := testutil.PanicT{}

			resp := (&dns.Msg{}).SetReply(req)
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{},
				A:   netip.MustParseAddrPort(bootstrapAddr).Addr().AsSlice(),
			})

			writeErr := w.WriteMsg(resp)
			require.NoError(pt, writeErr)
		}),
		NotifyStartedFunc: func() { close(upstreamStartedCh) },
	}

	go func() {
		listenErr := upstreamSrv.ListenAndServe()
		if listenErr != nil {
			// Log these immediately to see what happens.
			t.Logf("upstream listen error: %s", listenErr)
		}

		upstreamErrCh <- listenErr
	}()

	_, _ = testutil.RequireReceive(t, upstreamStartedCh, testTimeout)

	c := &dnssvc.Config{
		Logger:              slogutil.NewDiscardLogger(),
		Addresses:           []netip.AddrPort{netip.MustParseAddrPort(listenAddr)},
		BootstrapServers:    []string{upstreamSrv.PacketConn.LocalAddr().String()},
		UpstreamServers:     []string{upstreamAddr},
		DNS64Prefixes:       nil,
		UpstreamTimeout:     testTimeout,
		BootstrapPreferIPv6: false,
		UseDNS64:            false,
	}

	svc, err := dnssvc.New(c)
	require.NoError(t, err)

	err = svc.Start(testutil.ContextWithTimeout(t, testTimeout))
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

		cli := &dns.Client{}
		ctx := testutil.ContextWithTimeout(t, testTimeout)

		var resp *dns.Msg
		require.Eventually(t, func() (ok bool) {
			var excErr error
			resp, _, excErr = cli.ExchangeContext(ctx, req, addr.String())

			return excErr == nil
		}, testTimeout, testTimeout/10)

		assert.NotNil(t, resp)
	})

	err = svc.Shutdown(testutil.ContextWithTimeout(t, testTimeout))

	require.NoError(t, err)

	err = upstreamSrv.Shutdown()
	require.NoError(t, err)

	err, ok := testutil.RequireReceive(t, upstreamErrCh, testTimeout)
	require.True(t, ok)
	require.NoError(t, err)
}
