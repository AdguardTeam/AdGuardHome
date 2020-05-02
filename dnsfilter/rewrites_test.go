package dnsfilter

import (
	"net"
	"testing"

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
	r := d.processRewrites("host2.com")
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	r = d.processRewrites("www.host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.True(t, len(r.IPList) == 3)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))
	assert.True(t, r.IPList[1].Equal(net.ParseIP("1.2.3.5")))
	assert.True(t, r.IPList[2].Equal(net.ParseIP("1:2:3::4")))

	// wildcard
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"*.host.com", "1.2.3.5", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))

	r = d.processRewrites("www.host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.5")))

	r = d.processRewrites("www.host2.com")
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// override a wildcard
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"a.host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"*.host.com", "1.2.3.5", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("a.host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.True(t, len(r.IPList) == 1)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1.2.3.4")))

	// wildcard + CNAME
	d.Rewrites = []RewriteEntry{
		RewriteEntry{"host.com", "1.2.3.4", 0, nil},
		RewriteEntry{"*.host.com", "host.com", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("www.host.com")
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
	r = d.processRewrites("b.host.com")
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
	r = d.processRewrites("b.host.com")
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
	r := d.processRewrites("host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "1.1.1.1", r.IPList[0].String())

	// match L2
	r = d.processRewrites("sub.host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "2.2.2.2", r.IPList[0].String())

	// match L3
	r = d.processRewrites("my.sub.host.com")
	assert.Equal(t, ReasonRewrite, r.Reason)
	assert.Equal(t, 1, len(r.IPList))
	assert.Equal(t, "3.3.3.3", r.IPList[0].String())
}
