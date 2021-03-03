package dnsfilter

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestRewrites(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// CNAME, A, AAAA
	d.Rewrites = []RewriteEntry{
		{"somecname", "somehost.com", 0, nil},
		{"somehost.com", "0.0.0.0", 0, nil},

		{"host.com", "1.2.3.4", 0, nil},
		{"host.com", "1.2.3.5", 0, nil},
		{"host.com", "1:2:3::4", 0, nil},
		{"www.host.com", "host.com", 0, nil},
	}
	d.prepareRewrites()
	r := d.processRewrites("host2.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	r = d.processRewrites("www.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.Len(t, r.IPList, 2)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 4}))
	assert.True(t, r.IPList[1].Equal(net.IP{1, 2, 3, 5}))

	r = d.processRewrites("www.host.com", dns.TypeAAAA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.Len(t, r.IPList, 1)
	assert.True(t, r.IPList[0].Equal(net.ParseIP("1:2:3::4")))

	// wildcard
	d.Rewrites = []RewriteEntry{
		{"host.com", "1.2.3.4", 0, nil},
		{"*.host.com", "1.2.3.5", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 4}))

	r = d.processRewrites("www.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 5}))

	r = d.processRewrites("www.host2.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// override a wildcard
	d.Rewrites = []RewriteEntry{
		{"a.host.com", "1.2.3.4", 0, nil},
		{"*.host.com", "1.2.3.5", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("a.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 4}))

	// wildcard + CNAME
	d.Rewrites = []RewriteEntry{
		{"host.com", "1.2.3.4", 0, nil},
		{"*.host.com", "host.com", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("www.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 4}))

	// 2 CNAMEs
	d.Rewrites = []RewriteEntry{
		{"b.host.com", "a.host.com", 0, nil},
		{"a.host.com", "host.com", 0, nil},
		{"host.com", "1.2.3.4", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("b.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Equal(t, "host.com", r.CanonName)
	assert.Len(t, r.IPList, 1)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 4}))

	// 2 CNAMEs + wildcard
	d.Rewrites = []RewriteEntry{
		{"b.host.com", "a.host.com", 0, nil},
		{"a.host.com", "x.somehost.com", 0, nil},
		{"*.somehost.com", "1.2.3.4", 0, nil},
	}
	d.prepareRewrites()
	r = d.processRewrites("b.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Equal(t, "x.somehost.com", r.CanonName)
	assert.Len(t, r.IPList, 1)
	assert.True(t, r.IPList[0].Equal(net.IP{1, 2, 3, 4}))
}

func TestRewritesLevels(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// exact host, wildcard L2, wildcard L3
	d.Rewrites = []RewriteEntry{
		{"host.com", "1.1.1.1", 0, nil},
		{"*.host.com", "2.2.2.2", 0, nil},
		{"*.sub.host.com", "3.3.3.3", 0, nil},
	}
	d.prepareRewrites()

	// match exact
	r := d.processRewrites("host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, net.IP{1, 1, 1, 1}.Equal(r.IPList[0]))

	// match L2
	r = d.processRewrites("sub.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, net.IP{2, 2, 2, 2}.Equal(r.IPList[0]))

	// match L3
	r = d.processRewrites("my.sub.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, net.IP{3, 3, 3, 3}.Equal(r.IPList[0]))
}

func TestRewritesExceptionCNAME(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// wildcard; exception for a sub-domain
	d.Rewrites = []RewriteEntry{
		{"*.host.com", "2.2.2.2", 0, nil},
		{"sub.host.com", "sub.host.com", 0, nil},
	}
	d.prepareRewrites()

	// match sub-domain
	r := d.processRewrites("my.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, net.IP{2, 2, 2, 2}.Equal(r.IPList[0]))

	// match sub-domain, but handle exception
	r = d.processRewrites("sub.host.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)
}

func TestRewritesExceptionWC(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// wildcard; exception for a sub-wildcard
	d.Rewrites = []RewriteEntry{
		{"*.host.com", "2.2.2.2", 0, nil},
		{"*.sub.host.com", "*.sub.host.com", 0, nil},
	}
	d.prepareRewrites()

	// match sub-domain
	r := d.processRewrites("my.host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, net.IP{2, 2, 2, 2}.Equal(r.IPList[0]))

	// match sub-domain, but handle exception
	r = d.processRewrites("my.sub.host.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)
}

func TestRewritesExceptionIP(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// exception for AAAA record
	d.Rewrites = []RewriteEntry{
		{"host.com", "1.2.3.4", 0, nil},
		{"host.com", "AAAA", 0, nil},
		{"host2.com", "::1", 0, nil},
		{"host2.com", "A", 0, nil},
		{"host3.com", "A", 0, nil},
	}
	d.prepareRewrites()

	// match domain
	r := d.processRewrites("host.com", dns.TypeA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.True(t, net.IP{1, 2, 3, 4}.Equal(r.IPList[0]))

	// match exception
	r = d.processRewrites("host.com", dns.TypeAAAA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// match exception
	r = d.processRewrites("host2.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// match domain
	r = d.processRewrites("host2.com", dns.TypeAAAA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Len(t, r.IPList, 1)
	assert.Equal(t, "::1", r.IPList[0].String())

	// match exception
	r = d.processRewrites("host3.com", dns.TypeA)
	assert.Equal(t, NotFilteredNotFound, r.Reason)

	// match domain
	r = d.processRewrites("host3.com", dns.TypeAAAA)
	assert.Equal(t, Rewritten, r.Reason)
	assert.Empty(t, r.IPList)
}
