// Package aghtest contains utilities for testing.
package aghtest

import (
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/log"
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

// ReplaceLogWriter moves logger output to w and uses Cleanup method of t to
// revert changes.
func ReplaceLogWriter(t testing.TB, w io.Writer) {
	t.Helper()

	prev := log.Writer()
	t.Cleanup(func() { log.SetOutput(prev) })
	log.SetOutput(w)
}

// ReplaceLogLevel sets logging level to l and uses Cleanup method of t to
// revert changes.
func ReplaceLogLevel(t testing.TB, l log.Level) {
	t.Helper()

	switch l {
	case log.INFO, log.DEBUG, log.ERROR:
		// Go on.
	default:
		t.Fatalf("wrong l value (must be one of %v, %v, %v)", log.INFO, log.DEBUG, log.ERROR)
	}

	prev := log.GetLevel()
	t.Cleanup(func() { log.SetLevel(prev) })
	log.SetLevel(l)
}

// HostToIPs is a helper that generates one IPv4 and one IPv6 address from host.
func HostToIPs(host string) (ipv4, ipv6 netip.Addr) {
	hash := sha256.Sum256([]byte(host))

	return netip.AddrFrom4([4]byte(hash[:4])), netip.AddrFrom16([16]byte(hash[4:20]))
}

// StartHTTPServer is a helper that starts the HTTP server, which is configured
// to return data on every request, and returns the client and server URL.
func StartHTTPServer(t testing.TB, data []byte) (c *http.Client, u *url.URL) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	return srv.Client(), u
}

// testTimeout is a timeout for tests.
//
// TODO(e.burkov):  Move into agdctest.
const testTimeout = 1 * time.Second

// StartLocalhostUpstream is a test helper that starts a DNS server on
// localhost.
func StartLocalhostUpstream(t *testing.T, h dns.Handler) (addr *url.URL) {
	t.Helper()

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

		testutil.CleanupAndRequireSuccess(t, func() (err error) { return <-errCh })
		testutil.CleanupAndRequireSuccess(t, srv.Shutdown)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(testTimeout):
		require.FailNow(t, "timeout exceeded")
	}

	return addr
}
