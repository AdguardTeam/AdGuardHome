package home

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/stretchr/testify/assert"
)

func TestResolveRDNS(t *testing.T) {
	dns := &dnsforward.Server{}
	conf := &dnsforward.ServerConfig{}
	conf.UpstreamDNS = []string{"8.8.8.8"}
	err := dns.Prepare(conf)
	assert.True(t, err == nil, "%s", err)

	clients := &clientsContainer{}
	rdns := InitRDNS(dns, clients)
	r := rdns.resolve("1.1.1.1")
	assert.True(t, r == "one.one.one.one", "%s", r)
}
