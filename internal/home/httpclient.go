package home

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/httphdr"
)

// customUserAgentTransport sets the User-Agent on requests when it is missing
// to prevent Go from adding its default User-Agent.
type customUserAgentTransport struct {
	// transport is the underlying HTTP transport being wrapped.  It must not be
	// nil.
	transport http.RoundTripper

	// userAgent is the custom User-Agent string for requests.  It must not be
	// empty.
	userAgent string
}

// newCustomUserAgentTransport returns a properly initialized
// *customUserAgentTransport.  rt must not be nil.  ua must not be empty.
func newCustomUserAgentTransport(rt http.RoundTripper, ua string) (t *customUserAgentTransport) {
	return &customUserAgentTransport{
		transport: rt,
		userAgent: ua,
	}
}

// type check
var _ http.RoundTripper = (*customUserAgentTransport)(nil)

// RoundTrip implements the [http.RoundTripper] interface for
// *customUserAgentTransport.
func (t *customUserAgentTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if req.Header.Get(httphdr.UserAgent) == "" {
		req = req.Clone(req.Context())
		req.Header.Set(httphdr.UserAgent, t.userAgent)
	}

	return t.transport.RoundTrip(req)
}

// httpClient returns a new HTTP client that uses the AdGuard Home's own DNS
// server for resolving hostnames.  The resulting client should not be used
// until [Context.dnsServer] is initialized.  tlsMgr must not be nil.
//
// TODO(a.garipov, e.burkov): This is rather messy.  Refactor.
func httpClient(tlsMgr *tlsManager) (c *http.Client) {
	// Do not use Context.dnsServer.DialContext directly in the struct literal
	// below, since Context.dnsServer may be nil when this function is called.
	dialContext := func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
		return globalContext.dnsServer.DialContext(ctx, network, addr)
	}

	tr := newCustomUserAgentTransport(&http.Transport{
		DialContext: dialContext,
		Proxy:       httpProxy,
		TLSClientConfig: &tls.Config{
			RootCAs:      tlsMgr.rootCerts,
			CipherSuites: tlsMgr.customCipherIDs,
			MinVersion:   tls.VersionTLS12,
		},
	}, aghhttp.UserAgent())

	return &http.Client{
		// TODO(a.garipov): Make configurable.
		Timeout:   writeTimeout,
		Transport: tr,
	}
}

// httpProxy returns parses and returns an HTTP proxy URL from the config, if
// any.
func httpProxy(_ *http.Request) (u *url.URL, err error) {
	if config.ProxyURL == "" {
		return nil, nil
	}

	return url.Parse(config.ProxyURL)
}
