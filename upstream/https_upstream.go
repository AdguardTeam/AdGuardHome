package upstream

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	dnsMessageContentType = "application/dns-message"
	defaultKeepAlive      = 30 * time.Second
)

// HttpsUpstream is the upstream implementation for DNS-over-HTTPS
type HttpsUpstream struct {
	client   *http.Client
	endpoint *url.URL
}

// NewHttpsUpstream creates a new DNS-over-HTTPS upstream from hostname
func NewHttpsUpstream(endpoint string, bootstrap string) (Upstream, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	// Initialize bootstrap resolver
	bootstrapResolver := net.DefaultResolver
	if bootstrap != "" {
		bootstrapResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				var d net.Dialer
				conn, err := d.DialContext(ctx, network, bootstrap)
				return conn, err
			},
		}
	}

	dialer := &net.Dialer{
		Timeout:   defaultTimeout,
		KeepAlive: defaultKeepAlive,
		DualStack: true,
		Resolver:  bootstrapResolver,
	}

	// Update TLS and HTTP client configuration
	tlsConfig := &tls.Config{ServerName: u.Hostname()}
	transport := &http.Transport{
		TLSClientConfig:    tlsConfig,
		DisableCompression: true,
		MaxIdleConns:       1,
		DialContext:        dialer.DialContext,
	}
	http2.ConfigureTransport(transport)

	client := &http.Client{
		Timeout:   defaultTimeout,
		Transport: transport,
	}

	return &HttpsUpstream{client: client, endpoint: u}, nil
}

// Exchange provides an implementation for the Upstream interface
func (u *HttpsUpstream) Exchange(ctx context.Context, query *dns.Msg) (*dns.Msg, error) {
	queryBuf, err := query.Pack()
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack DNS query")
	}

	// No content negotiation for now, use DNS wire format
	buf, backendErr := u.exchangeWireformat(queryBuf)
	if backendErr == nil {
		response := &dns.Msg{}
		if err := response.Unpack(buf); err != nil {
			return nil, errors.Wrap(err, "failed to unpack DNS response from body")
		}

		response.Id = query.Id
		return response, nil
	}

	log.Printf("failed to connect to an HTTPS backend %q due to %s", u.endpoint, backendErr)
	return nil, backendErr
}

// Perform message exchange with the default UDP wireformat defined in current draft
// https://tools.ietf.org/html/draft-ietf-doh-dns-over-https-10
func (u *HttpsUpstream) exchangeWireformat(msg []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", u.endpoint.String(), bytes.NewBuffer(msg))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create an HTTPS request")
	}

	req.Header.Add("Content-Type", dnsMessageContentType)
	req.Header.Add("Accept", dnsMessageContentType)
	req.Host = u.endpoint.Hostname()

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform an HTTPS request")
	}

	// Check response status code
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("returned status code %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != dnsMessageContentType {
		return nil, fmt.Errorf("return wrong content type %s", contentType)
	}

	// Read application/dns-message response from the body
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read the response body")
	}

	return buf, nil
}

// Clear resources
func (u *HttpsUpstream) Close() error {
	return nil
}
