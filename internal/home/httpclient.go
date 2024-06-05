package home

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
)

// httpClient returns a new HTTP client that uses the AdGuard Home's own DNS
// server for resolving hostnames.  The resulting client should not be used
// until [Context.dnsServer] is initialized.
//
// TODO(a.garipov, e.burkov): This is rather messy.  Refactor.
func httpClient() (c *http.Client) {
	// Do not use Context.dnsServer.DialContext directly in the struct literal
	// below, since Context.dnsServer may be nil when this function is called.
	dialContext := func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
		return Context.dnsServer.DialContext(ctx, network, addr)
	}

	return &http.Client{
		// TODO(a.garipov): Make configurable.
		Timeout: writeTimeout,
		Transport: &http.Transport{
			DialContext: dialContext,
			Proxy:       httpProxy,
			TLSClientConfig: &tls.Config{
				RootCAs:      Context.tlsRoots,
				CipherSuites: Context.tlsCipherIDs,
				MinVersion:   tls.VersionTLS12,
			},
		},
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
