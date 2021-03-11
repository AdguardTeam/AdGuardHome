package dnsfilter

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov): All the tests in this file may and should me merged together.

func TestRewrites(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)

	d.Rewrites = []RewriteEntry{{
		// This one and below are about CNAME, A and AAAA.
		Domain: "somecname",
		Answer: "somehost.com",
	}, {
		Domain: "somehost.com",
		Answer: "0.0.0.0",
	}, {
		Domain: "host.com",
		Answer: "1.2.3.4",
	}, {
		Domain: "host.com",
		Answer: "1.2.3.5",
	}, {
		Domain: "host.com",
		Answer: "1:2:3::4",
	}, {
		Domain: "www.host.com",
		Answer: "host.com",
	}, {
		// This one is a wildcard.
		Domain: "*.host.com",
		Answer: "1.2.3.5",
	}, {
		// This one and below are about wildcard overriding.
		Domain: "a.host.com",
		Answer: "1.2.3.4",
	}, {
		// This one is about CNAME and wildcard interacting.
		Domain: "*.host2.com",
		Answer: "host.com",
	}, {
		// This one and below are about 2 level CNAME.
		Domain: "b.host.com",
		Answer: "somecname",
	}, {
		// This one and below are about 2 level CNAME and wildcard.
		Domain: "b.host3.com",
		Answer: "a.host3.com",
	}, {
		Domain: "a.host3.com",
		Answer: "x.host.com",
	}}
	d.prepareRewrites()

	testCases := []struct {
		name      string
		host      string
		dtyp      uint16
		wantCName string
		wantVals  []net.IP
	}{{
		name: "not_filtered_not_found",
		host: "hoost.com",
		dtyp: dns.TypeA,
	}, {
		name:      "rewritten_a",
		host:      "www.host.com",
		dtyp:      dns.TypeA,
		wantCName: "host.com",
		wantVals:  []net.IP{{1, 2, 3, 4}, {1, 2, 3, 5}},
	}, {
		name:      "rewritten_aaaa",
		host:      "www.host.com",
		dtyp:      dns.TypeAAAA,
		wantCName: "host.com",
		wantVals:  []net.IP{net.ParseIP("1:2:3::4")},
	}, {
		name:     "wildcard_match",
		host:     "abc.host.com",
		dtyp:     dns.TypeA,
		wantVals: []net.IP{{1, 2, 3, 5}},
	}, {
		name:     "wildcard_override",
		host:     "a.host.com",
		dtyp:     dns.TypeA,
		wantVals: []net.IP{{1, 2, 3, 4}},
	}, {
		name:      "wildcard_cname_interaction",
		host:      "www.host2.com",
		dtyp:      dns.TypeA,
		wantCName: "host.com",
		wantVals:  []net.IP{{1, 2, 3, 4}, {1, 2, 3, 5}},
	}, {
		name:      "two_cnames",
		host:      "b.host.com",
		dtyp:      dns.TypeA,
		wantCName: "somehost.com",
		wantVals:  []net.IP{{0, 0, 0, 0}},
	}, {
		name:      "two_cnames_and_wildcard",
		host:      "b.host3.com",
		dtyp:      dns.TypeA,
		wantCName: "x.host.com",
		wantVals:  []net.IP{{1, 2, 3, 5}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valsNum := len(tc.wantVals)

			r := d.processRewrites(tc.host, tc.dtyp)
			if valsNum == 0 {
				assert.Equal(t, NotFilteredNotFound, r.Reason)

				return
			}

			require.Equal(t, Rewritten, r.Reason)
			if tc.wantCName != "" {
				assert.Equal(t, tc.wantCName, r.CanonName)
			}

			require.Len(t, r.IPList, valsNum)
			for i, ip := range tc.wantVals {
				assert.Equal(t, ip, r.IPList[i])
			}
		})
	}
}

func TestRewritesLevels(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// Exact host, wildcard L2, wildcard L3.
	d.Rewrites = []RewriteEntry{{
		Domain: "host.com",
		Answer: "1.1.1.1",
	}, {
		Domain: "*.host.com",
		Answer: "2.2.2.2",
	}, {
		Domain: "*.sub.host.com",
		Answer: "3.3.3.3",
	}}
	d.prepareRewrites()

	testCases := []struct {
		name string
		host string
		want net.IP
	}{{
		name: "exact_match",
		host: "host.com",
		want: net.IP{1, 1, 1, 1},
	}, {
		name: "l2_match",
		host: "sub.host.com",
		want: net.IP{2, 2, 2, 2},
	}, {
		name: "l3_match",
		host: "my.sub.host.com",
		want: net.IP{3, 3, 3, 3},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := d.processRewrites(tc.host, dns.TypeA)
			assert.Equal(t, Rewritten, r.Reason)
			require.Len(t, r.IPList, 1)
		})
	}
}

func TestRewritesExceptionCNAME(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// Wildcard and exception for a sub-domain.
	d.Rewrites = []RewriteEntry{{
		Domain: "*.host.com",
		Answer: "2.2.2.2",
	}, {
		Domain: "sub.host.com",
		Answer: "sub.host.com",
	}, {
		Domain: "*.sub.host.com",
		Answer: "*.sub.host.com",
	}}
	d.prepareRewrites()

	testCases := []struct {
		name string
		host string
		want net.IP
	}{{
		name: "match_sub-domain",
		host: "my.host.com",
		want: net.IP{2, 2, 2, 2},
	}, {
		name: "exception_cname",
		host: "sub.host.com",
	}, {
		name: "exception_wildcard",
		host: "my.sub.host.com",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := d.processRewrites(tc.host, dns.TypeA)
			if tc.want == nil {
				assert.Equal(t, NotFilteredNotFound, r.Reason)

				return
			}

			assert.Equal(t, Rewritten, r.Reason)
			require.Len(t, r.IPList, 1)
			assert.True(t, tc.want.Equal(r.IPList[0]))
		})
	}
}

func TestRewritesExceptionIP(t *testing.T) {
	d := newForTest(nil, nil)
	t.Cleanup(d.Close)
	// Exception for AAAA record.
	d.Rewrites = []RewriteEntry{{
		Domain: "host.com",
		Answer: "1.2.3.4",
	}, {
		Domain: "host.com",
		Answer: "AAAA",
	}, {
		Domain: "host2.com",
		Answer: "::1",
	}, {
		Domain: "host2.com",
		Answer: "A",
	}, {
		Domain: "host3.com",
		Answer: "A",
	}}
	d.prepareRewrites()

	testCases := []struct {
		name string
		host string
		dtyp uint16
		want []net.IP
	}{{
		name: "match_A",
		host: "host.com",
		dtyp: dns.TypeA,
		want: []net.IP{{1, 2, 3, 4}},
	}, {
		name: "exception_AAAA_host.com",
		host: "host.com",
		dtyp: dns.TypeAAAA,
	}, {
		name: "exception_A_host2.com",
		host: "host2.com",
		dtyp: dns.TypeA,
	}, {
		name: "match_AAAA_host2.com",
		host: "host2.com",
		dtyp: dns.TypeAAAA,
		want: []net.IP{net.ParseIP("::1")},
	}, {
		name: "exception_A_host3.com",
		host: "host3.com",
		dtyp: dns.TypeA,
	}, {
		name: "match_AAAA_host3.com",
		host: "host3.com",
		dtyp: dns.TypeAAAA,
		want: []net.IP{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name+"_"+tc.host, func(t *testing.T) {
			r := d.processRewrites(tc.host, tc.dtyp)
			if tc.want == nil {
				assert.Equal(t, NotFilteredNotFound, r.Reason)

				return
			}

			assert.Equal(t, Rewritten, r.Reason)
			require.Len(t, r.IPList, len(tc.want))
			for _, ip := range tc.want {
				assert.True(t, ip.Equal(r.IPList[0]))
			}
		})
	}
}
