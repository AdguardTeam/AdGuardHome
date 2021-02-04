package home

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/stretchr/testify/assert"
)

func TestResolveRDNS(t *testing.T) {
	ups := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"1.1.1.1.in-addr.arpa.": {"one.one.one.one"},
		},
	}
	dns := dnsforward.NewCustomServer(&proxy.Proxy{
		Config: proxy.Config{
			UpstreamConfig: &proxy.UpstreamConfig{
				Upstreams: []upstream.Upstream{ups},
			},
		},
	})

	clients := &clientsContainer{}
	rdns := InitRDNS(dns, clients)
	r := rdns.resolve(net.IP{1, 1, 1, 1})
	assert.Equal(t, "one.one.one.one", r, r)
}
