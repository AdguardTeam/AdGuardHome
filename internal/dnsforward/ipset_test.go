package dnsforward

import (
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestIPSET(t *testing.T) {
	s := Server{}
	s.conf.IPSETList = append(s.conf.IPSETList, "HOST.com/name")
	s.conf.IPSETList = append(s.conf.IPSETList, "host2.com,host3.com/name23")
	s.conf.IPSETList = append(s.conf.IPSETList, "host4.com/name4,name41")
	c := ipsetCtx{}
	c.init(s.conf.IPSETList)

	assert.Equal(t, "name", c.ipsetList["host.com"][0])
	assert.Equal(t, "name23", c.ipsetList["host2.com"][0])
	assert.Equal(t, "name23", c.ipsetList["host3.com"][0])
	assert.Equal(t, "name4", c.ipsetList["host4.com"][0])
	assert.Equal(t, "name41", c.ipsetList["host4.com"][1])

	_, ok := c.ipsetList["host0.com"]
	assert.False(t, ok)

	ctx := &dnsContext{
		srv: &s,
	}
	ctx.proxyCtx = &proxy.DNSContext{}
	ctx.proxyCtx.Req = &dns.Msg{
		Question: []dns.Question{
			{
				Name:  "host.com.",
				Qtype: dns.TypeA,
			},
		},
	}
	assert.Equal(t, resultDone, c.process(ctx))
}
