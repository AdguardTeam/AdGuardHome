package home

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/stretchr/testify/assert"
)

func prepareTestDNSServer() error {
	config.DNS.Port = 1234
	Context.dnsServer = dnsforward.NewServer(dnsforward.DNSCreateParams{})
	conf := &dnsforward.ServerConfig{}
	conf.UpstreamDNS = []string{"8.8.8.8"}
	return Context.dnsServer.Prepare(conf)
}

func TestWhois(t *testing.T) {
	assert.Nil(t, prepareTestDNSServer())

	w := Whois{timeoutMsec: 5000}
	resp, err := w.queryAll("8.8.8.8")
	assert.Nil(t, err)
	m := whoisParse(resp)
	assert.Equal(t, "Google LLC", m["orgname"])
	assert.Equal(t, "US", m["country"])
	assert.Equal(t, "Mountain View", m["city"])
}
