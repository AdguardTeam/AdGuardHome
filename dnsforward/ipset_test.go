package dnsforward

import (
	"net"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

var s Server
var c ipsetCtx
var ctx *dnsContext

type Binding struct {
	host  string
	ipset string
	ipStr string
}

var b map[Binding]int

func setup() {
	s = Server{}
	s.conf.IPSETList = []string{
		"HOST.com/name",
		"host2.com,host3.com/name23",
		"host4.com/name4,name41",
		"sub.host4.com/subhost4",
	}

	c = ipsetCtx{}
	c.init(s.conf.IPSETList)

	ctx = &dnsContext{
		srv: &s,
	}
	ctx.responseFromUpstream = true
	ctx.proxyCtx = &proxy.DNSContext{}

	b = make(map[Binding]int)
}

func makeReq(fqdn string, qtype uint16) *dns.Msg {
	return &dns.Msg{
		Question: []dns.Question{
			{
				Name:  fqdn,
				Qtype: qtype,
			},
		},
	}
}

func makeReqA(fqdn string) *dns.Msg {
	return makeReq(fqdn, dns.TypeA)
}

func makeReqAAAA(fqdn string) *dns.Msg {
	return makeReq(fqdn, dns.TypeAAAA)
}

func makeA(fqdn string, ip net.IP) *dns.A {
	return &dns.A{
		Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   ip,
	}
}

func makeAAAA(fqdn string, ip net.IP) *dns.AAAA {
	return &dns.AAAA{
		Hdr:  dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
		AAAA: ip,
	}
}

func makeCNAME(fqdn string, cnameFqdn string) *dns.CNAME {
	return &dns.CNAME{
		Hdr:    dns.RR_Header{Name: fqdn, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 0},
		Target: cnameFqdn,
	}
}

func addToBindings(host string, ipset string, ipStr string) {
	binding := Binding{host, ipset, ipStr}
	count := b[binding]
	b[binding] = count + 1
}

func doProcess(t *testing.T) {
	assert.Equal(t, resultDone, c.processMembers(ctx, addToBindings))
}

func TestIpsetParsing(t *testing.T) {
	setup()

	assert.Equal(t, "name", c.ipsetList["host.com"][0])
	assert.Equal(t, "name23", c.ipsetList["host2.com"][0])
	assert.Equal(t, "name23", c.ipsetList["host3.com"][0])
	assert.Equal(t, "name4", c.ipsetList["host4.com"][0])
	assert.Equal(t, "name41", c.ipsetList["host4.com"][1])

	_, ok := c.ipsetList["host0.com"]
	assert.False(t, ok)
}

func TestIpsetNoQuestion(t *testing.T) {
	setup()

	doProcess(t)
	assert.Equal(t, 0, len(b))
}

func TestIpsetNoAnswer(t *testing.T) {
	setup()

	ctx.proxyCtx.Req = makeReqA("HOST4.COM.")

	doProcess(t)
	assert.Equal(t, 0, len(b))
}

func TestIpsetCache(t *testing.T) {
	setup()

	ctx.proxyCtx.Req = makeReqA("HOST4.COM.")
	ctx.proxyCtx.Res = &dns.Msg{
		Answer: []dns.RR{
			makeA("HOST4.COM.", net.IPv4(127, 0, 0, 1)),
			makeAAAA("HOST4.COM.", net.IPv6loopback),
		},
	}

	doProcess(t)

	assert.Equal(t, 1, b[Binding{"host4.com", "name4", "127.0.0.1"}])
	assert.Equal(t, 1, b[Binding{"host4.com", "name41", "127.0.0.1"}])
	assert.Equal(t, 1, b[Binding{"host4.com", "name4", net.IPv6loopback.String()}])
	assert.Equal(t, 1, b[Binding{"host4.com", "name41", net.IPv6loopback.String()}])
	assert.Equal(t, 4, len(b))

	doProcess(t)

	assert.Equal(t, 1, b[Binding{"host4.com", "name4", "127.0.0.1"}])
	assert.Equal(t, 1, b[Binding{"host4.com", "name41", "127.0.0.1"}])
	assert.Equal(t, 1, b[Binding{"host4.com", "name4", net.IPv6loopback.String()}])
	assert.Equal(t, 1, b[Binding{"host4.com", "name41", net.IPv6loopback.String()}])
	assert.Equal(t, 4, len(b))
}

func TestIpsetSubdomainOverride(t *testing.T) {
	setup()

	ctx.proxyCtx.Req = makeReqA("sub.host4.com.")
	ctx.proxyCtx.Res = &dns.Msg{
		Answer: []dns.RR{
			makeA("sub.host4.com.", net.IPv4(127, 0, 0, 1)),
		},
	}

	doProcess(t)

	assert.Equal(t, 1, b[Binding{"sub.host4.com", "subhost4", "127.0.0.1"}])
	assert.Equal(t, 1, len(b))
}

func TestIpsetSubdomainWildcard(t *testing.T) {
	setup()

	ctx.proxyCtx.Req = makeReqA("sub.host.com.")
	ctx.proxyCtx.Res = &dns.Msg{
		Answer: []dns.RR{
			makeA("sub.host.com.", net.IPv4(127, 0, 0, 1)),
		},
	}

	doProcess(t)

	assert.Equal(t, 1, b[Binding{"sub.host.com", "name", "127.0.0.1"}])
	assert.Equal(t, 1, len(b))
}

func TestIpsetCnameThirdParty(t *testing.T) {
	setup()

	ctx.proxyCtx.Req = makeReqA("host.com.")
	ctx.proxyCtx.Res = &dns.Msg{
		Answer: []dns.RR{
			makeCNAME("host.com.", "foo.bar.baz.elb.amazonaws.com."),
			makeA("foo.bar.baz.elb.amazonaws.com.", net.IPv4(8, 8, 8, 8)),
		},
	}

	doProcess(t)

	assert.Equal(t, 1, b[Binding{"host.com", "name", "8.8.8.8"}])
	assert.Equal(t, 1, len(b))
}
