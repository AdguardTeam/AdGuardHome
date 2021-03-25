package home

import (
	"context"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareTestDNSServer(t *testing.T) {
	t.Helper()

	config.DNS.Port = 1234

	var err error
	Context.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{})
	require.NoError(t, err)

	conf := &dnsforward.ServerConfig{}
	conf.UpstreamDNS = []string{"8.8.8.8"}

	err = Context.dnsServer.Prepare(conf)
	require.NoError(t, err)
}

// TODO(e.burkov): It's kind of complicated to get rid of network access in this
// test.  The thing is that *Whois creates new *net.Dialer each time it requests
// the server, so it becomes hard to simulate handling of request from test even
// with substituted upstream.  However, it must be done.
func TestWhois(t *testing.T) {
	prepareTestDNSServer(t)

	w := Whois{timeoutMsec: 5000}
	resp, err := w.queryAll(context.Background(), "8.8.8.8")
	assert.NoError(t, err)

	m := whoisParse(resp)
	require.NotEmpty(t, m)

	assert.Equal(t, "Google LLC", m["orgname"])
	assert.Equal(t, "US", m["country"])
	assert.Equal(t, "Mountain View", m["city"])
}
