// Package aghtest contains utilities for testing.
package aghtest

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

const (
	// ReqHost is the common request host for filtering tests.
	ReqHost = "www.host.example"

	// ReqFQDN is the common request FQDN for filtering tests.
	ReqFQDN = ReqHost + "."
)

// HostToIPs is a helper that generates one IPv4 and one IPv6 address from host.
func HostToIPs(host string) (ipv4, ipv6 netip.Addr) {
	hash := sha256.Sum256([]byte(host))

	return netip.AddrFrom4([4]byte(hash[:4])), netip.AddrFrom16([16]byte(hash[4:20]))
}

// StartHTTPServer is a helper that starts the HTTP server, which is configured
// to return data on every request, and returns the client and server URL.
func StartHTTPServer(tb testing.TB, data []byte) (c *http.Client, u *url.URL) {
	tb.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	tb.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(tb, err)

	return srv.Client(), u
}

// testTimeout is a timeout for tests.
//
// TODO(e.burkov):  Move into agdctest.
const testTimeout = 1 * time.Second

// StartLocalhostUpstream is a test helper that starts a DNS server on
// localhost.
func StartLocalhostUpstream(tb *testing.T, h dns.Handler) (addr *url.URL) {
	tb.Helper()

	startCh := make(chan netip.AddrPort)
	defer close(startCh)
	errCh := make(chan error)

	srv := &dns.Server{
		Addr:         "127.0.0.1:0",
		Net:          string(proxy.ProtoTCP),
		Handler:      h,
		ReadTimeout:  testTimeout,
		WriteTimeout: testTimeout,
	}
	srv.NotifyStartedFunc = func() {
		addrPort := srv.Listener.Addr()
		startCh <- netutil.NetAddrToAddrPort(addrPort)
	}

	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case addrPort := <-startCh:
		addr = &url.URL{
			Scheme: string(proxy.ProtoTCP),
			Host:   addrPort.String(),
		}

		testutil.CleanupAndRequireSuccess(tb, func() (err error) { return <-errCh })
		testutil.CleanupAndRequireSuccess(tb, srv.Shutdown)
	case err := <-errCh:
		require.NoError(tb, err)
	case <-time.After(testTimeout):
		require.FailNow(tb, "timeout exceeded")
	}

	return addr
}
