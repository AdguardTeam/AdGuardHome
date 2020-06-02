package home

import (
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/stretchr/testify/assert"
)

func prepareTestDNSServer() error {
	config.DNS.Port = 1234
	Context.dnsServer = dnsforward.NewServer(nil, nil, nil)
	conf := &dnsforward.ServerConfig{}
	uc, err := proxy.ParseUpstreamsConfig([]string{"1.1.1.1"}, nil, time.Second*5)
	if err != nil {
		return err
	}
	conf.UpstreamConfig = &uc
	return Context.dnsServer.Prepare(conf)
}

func TestWhois(t *testing.T) {
	err := prepareTestDNSServer()
	assert.Nil(t, err)

	w := Whois{timeoutMsec: 5000}
	resp, err := w.queryAll("8.8.8.8")
	assert.True(t, err == nil)
	m := whoisParse(resp)
	assert.True(t, m["orgname"] == "Google LLC")
	assert.True(t, m["country"] == "US")
	assert.True(t, m["city"] == "Mountain View")
}
