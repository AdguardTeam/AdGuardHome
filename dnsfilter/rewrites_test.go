package dnsfilter

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestRewrites(t *testing.T) {
	d := Dnsfilter{}
	// CNAME, A, AAAA
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"somecname", "somehost.com", 0, nil},
		RewriteEntry{"somehost.com", "0.0.0.0", 0, nil},

		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"host.com", "1.2.3.5", 0, nil},
		RewriteEntry{"host.com", "1:2:3::4", 0, nil},
		RewriteEntry{"www.host.com", "host.com", 0, nil},
	}
	d.prepareRewrites()
	r := d.processRewrites("host2.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	r = d.processRewrites("www.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.Equal(t, 2, len(r.IPList))
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))
	assert.True(t, r.IPList[1].Equal(net.ParseIP("1.2.3.5")))

	r = d.processRewrites("www.host.com", dns.TypeAAAA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.Equal(t, 1, len(r.IPList))
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1:2:3::4")))

	// wildcard
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"*.host.com", "1.2.3.5", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))

	r = d.processRewrites("www.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.5")))

	r = d.processRewrites("www.host2.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// override a wildcard
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"a.host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"*.host.com", "1.2.3.5", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("a.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.True(t, len(r.IPList) == 1)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))

	// wildcard + CNAME
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"*.host.com", "host.com", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("www.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))

	// 2 CNAMEs
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"b.host.com", "a.host.com", 0, nil},
		RewriteEntry{"a.host.com", "host.com", 0, nil},
		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("b.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.True(t, len(r.IPList) == 1)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))

	// 2 CNAMEs + wildcard
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"b.host.com", "a.host.com", 0, nil},
		RewriteEntry{"a.host.com", "x.somehost.com", 0, nil},
		RewriteEntry{"*.somehost.com", "1.2.3.4", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("b.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, "x.somehost.com", r.CanonName)
	assert.True(t, len(r.IPList) == 1)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))
}

func TestRewritesLevels(t *testing.T) {
	d := Dnsfilter{}
	// exact host, wildcard L2, wildcard L3
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"host.com", "1.1.1.1", 0, nil},
		RewriteEntry{"*.host.com", "2.2.2.2", 0, nil},
		RewriteEntry{"*.sub.host.com", "3.3.3.3", 0, nil},
	}
	d.prepareRewrites()

	// match exact
	r := d.processRewrites("host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "1.1.1.1", r.IPList[0].String())

	// match L2
	r = d.processRewrites("sub.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "2.2.2.2", r.IPList[0].String())

	// match L3
	r = d.processRewrites("my.sub.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "3.3.3.3", r.IPList[0].String())
}

func TestRewritesExceptionCNAME(t *testing.T) {
	d := Dnsfilter{}
	// wildcard; exception for a sub-domain
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"*.host.com", "2.2.2.2", 0, nil},
		RewriteEntry{"sub.host.com", "sub.host.com", 0, nil},
	}
	d.prepareRewrites()

	// match sub-domain
	r := d.processRewrites("my.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "2.2.2.2", r.IPList[0].String())

	// match sub-domain, but handle exception
	r = d.processRewrites("sub.host.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)
}

func TestRewritesExceptionWC(t *testing.T) {
	d := Dnsfilter{}
	// wildcard; exception for a sub-wildcard
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"*.host.com", "2.2.2.2", 0, nil},
		RewriteEntry{"*.sub.host.com", "*.sub.host.com", 0, nil},
	}
	d.prepareRewrites()

	// match sub-domain
	r := d.processRewrites("my.host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "2.2.2.2", r.IPList[0].String())

	// match sub-domain, but handle exception
	r = d.processRewrites("my.sub.host.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)
}

func TestRewritesExceptionIP(t *testing.T) {
	d := Dnsfilter{}
	// exception for AAAA record
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"host.com", "AAAA", 0, nil},
		RewriteEntry{"host2.com", "::1", 0, nil},
		RewriteEntry{"host2.com", "A", 0, nil},
		RewriteEntry{"host3.com", "A", 0, nil},
	}
	d.prepareRewrites()

	// match domain
	r := d.processRewrites("host.com", dns.TypeA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "1.2.3.4", r.IPList[0].String())

	// match exception
	r = d.processRewrites("host.com", dns.TypeAAAA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// match exception
	r = d.processRewrites("host2.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// match domain
	r = d.processRewrites("host2.com", dns.TypeAAAA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "::1", r.IPList[0].String())

	// match exception
	r = d.processRewrites("host3.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// match domain
	r = d.processRewrites("host3.com", dns.TypeAAAA)
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 0, len(r.IPList))
}
