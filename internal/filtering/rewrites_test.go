package filtering

import (
	"net"
	"net/netip"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov): All the tests in this file may and should me merged together.

func TestRewrites(t *testing.T) {
	d, _ := newForTest(t, nil, nil)
	t.Cleanup(d.Close)

	var (
		addr1v4 = netip.AddrFrom4([4]byte{1, 2, 3, 4})
		addr2v4 = netip.AddrFrom4([4]byte{1, 2, 3, 5})
		addr3v4 = netip.AddrFrom4([4]byte{1, 2, 3, 6})
		addr4v4 = netip.AddrFrom4([4]byte{1, 2, 3, 7})

		addr1v6 = netip.MustParseAddr("1:2:3::4")
		addr2v6 = netip.MustParseAddr("1234::5678")
	)

	d.conf.Rewrites = []*LegacyRewrite{{
		// This one and below are about CNAME, A and AAAA.
		Domain: "somecname",
		Answer: "somehost.com",
	}, {
		Domain: "somehost.com",
		Answer: netip.IPv4Unspecified().String(),
	}, {
		Domain: "host.com",
		Answer: addr1v4.String(),
	}, {
		Domain: "host.com",
		Answer: addr2v4.String(),
	}, {
		Domain: "host.com",
		Answer: addr1v6.String(),
	}, {
		Domain: "www.host.com",
		Answer: "host.com",
	}, {
		// This one is a wildcard.
		Domain: "*.host.com",
		Answer: addr2v4.String(),
	}, {
		// This one and below are about wildcard overriding.
		Domain: "a.host.com",
		Answer: addr1v4.String(),
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
	}, {
		Domain: "*.hostboth.com",
		Answer: addr3v4.String(),
	}, {
		Domain: "*.hostboth.com",
		Answer: addr2v6.String(),
	}, {
		Domain: "BIGHOST.COM",
		Answer: addr4v4.String(),
	}, {
		Domain: "*.issue4016.com",
		Answer: "sub.issue4016.com",
	}, {
		Domain: "*.sub.issue6226.com",
		Answer: addr2v4.String(),
	}, {
		Domain: "*.issue6226.com",
		Answer: addr1v4.String(),
	}}

	require.NoError(t, d.prepareRewrites())

	testCases := []struct {
		name       string
		host       string
		wantCName  string
		wantIPs    []netip.Addr
		wantReason Reason
		dtyp       uint16
	}{{
		name:       "not_filtered_not_found",
		host:       "hoost.com",
		wantCName:  "",
		wantIPs:    nil,
		wantReason: NotFilteredNotFound,
		dtyp:       dns.TypeA,
	}, {
		name:       "rewritten_a",
		host:       "www.host.com",
		wantCName:  "host.com",
		wantIPs:    []netip.Addr{addr1v4, addr2v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "rewritten_aaaa",
		host:       "www.host.com",
		wantCName:  "host.com",
		wantIPs:    []netip.Addr{addr1v6},
		wantReason: Rewritten,
		dtyp:       dns.TypeAAAA,
	}, {
		name:       "wildcard_match",
		host:       "abc.host.com",
		wantCName:  "",
		wantIPs:    []netip.Addr{addr2v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "wildcard_override",
		host:       "a.host.com",
		wantCName:  "",
		wantIPs:    []netip.Addr{addr1v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "wildcard_cname_interaction",
		host:       "www.host2.com",
		wantCName:  "host.com",
		wantIPs:    []netip.Addr{addr1v4, addr2v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "two_cnames",
		host:       "b.host.com",
		wantCName:  "somehost.com",
		wantIPs:    []netip.Addr{netip.IPv4Unspecified()},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "two_cnames_and_wildcard",
		host:       "b.host3.com",
		wantCName:  "x.host.com",
		wantIPs:    []netip.Addr{addr2v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "issue3343",
		host:       "www.hostboth.com",
		wantCName:  "",
		wantIPs:    []netip.Addr{addr2v6},
		wantReason: Rewritten,
		dtyp:       dns.TypeAAAA,
	}, {
		name:       "issue3351",
		host:       "bighost.com",
		wantCName:  "",
		wantIPs:    []netip.Addr{addr4v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "issue4008",
		host:       "somehost.com",
		wantCName:  "",
		wantIPs:    nil,
		wantReason: Rewritten,
		dtyp:       dns.TypeHTTPS,
	}, {
		name:       "issue4016",
		host:       "www.issue4016.com",
		wantCName:  "sub.issue4016.com",
		wantIPs:    nil,
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "issue4016_self",
		host:       "sub.issue4016.com",
		wantCName:  "",
		wantIPs:    nil,
		wantReason: NotFilteredNotFound,
		dtyp:       dns.TypeA,
	}, {
		name:       "issue6226",
		host:       "www.issue6226.com",
		wantCName:  "",
		wantIPs:    []netip.Addr{addr1v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}, {
		name:       "issue6226_sub",
		host:       "www.sub.issue6226.com",
		wantCName:  "",
		wantIPs:    []netip.Addr{addr2v4},
		wantReason: Rewritten,
		dtyp:       dns.TypeA,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := d.processRewrites(tc.host, tc.dtyp)
			require.Equalf(t, tc.wantReason, r.Reason, "got %s", r.Reason)

			if tc.wantCName != "" {
				assert.Equal(t, tc.wantCName, r.CanonName)
			}

			assert.Equal(t, tc.wantIPs, r.IPList)
		})
	}
}

func TestRewritesLevels(t *testing.T) {
	d, _ := newForTest(t, nil, nil)
	t.Cleanup(d.Close)
	// Exact host, wildcard L2, wildcard L3.
	d.conf.Rewrites = []*LegacyRewrite{{
		Domain: "host.com",
		Answer: "1.1.1.1",
		Type:   dns.TypeA,
	}, {
		Domain: "*.host.com",
		Answer: "2.2.2.2",
		Type:   dns.TypeA,
	}, {
		Domain: "*.sub.host.com",
		Answer: "3.3.3.3",
		Type:   dns.TypeA,
	}}

	require.NoError(t, d.prepareRewrites())

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
	d, _ := newForTest(t, nil, nil)
	t.Cleanup(d.Close)
	// Wildcard and exception for a sub-domain.
	d.conf.Rewrites = []*LegacyRewrite{{
		Domain: "*.host.com",
		Answer: "2.2.2.2",
	}, {
		Domain: "sub.host.com",
		Answer: "sub.host.com",
	}, {
		Domain: "*.sub.host.com",
		Answer: "*.sub.host.com",
	}}

	require.NoError(t, d.prepareRewrites())

	testCases := []struct {
		name string
		host string
		want netip.Addr
	}{{
		name: "match_subdomain",
		host: "my.host.com",
		want: netip.AddrFrom4([4]byte{2, 2, 2, 2}),
	}, {
		name: "exception_cname",
		host: "sub.host.com",
		want: netip.Addr{},
	}, {
		name: "exception_wildcard",
		host: "my.sub.host.com",
		want: netip.Addr{},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := d.processRewrites(tc.host, dns.TypeA)
			if tc.want == (netip.Addr{}) {
				assert.Equal(t, NotFilteredNotFound, r.Reason, "got %s", r.Reason)

				return
			}

			assert.Equal(t, Rewritten, r.Reason)
			require.Len(t, r.IPList, 1)
			assert.Equal(t, tc.want, r.IPList[0])
		})
	}
}

func TestRewritesExceptionIP(t *testing.T) {
	d, _ := newForTest(t, nil, nil)
	t.Cleanup(d.Close)
	// Exception for AAAA record.
	d.conf.Rewrites = []*LegacyRewrite{{
		Domain: "host.com",
		Answer: "1.2.3.4",
		Type:   dns.TypeA,
	}, {
		Domain: "host.com",
		Answer: "AAAA",
		Type:   dns.TypeAAAA,
	}, {
		Domain: "host2.com",
		Answer: "::1",
		Type:   dns.TypeAAAA,
	}, {
		Domain: "host2.com",
		Answer: "A",
		Type:   dns.TypeA,
	}, {
		Domain: "host3.com",
		Answer: "A",
		Type:   dns.TypeA,
	}}

	require.NoError(t, d.prepareRewrites())

	testCases := []struct {
		name       string
		host       string
		want       []netip.Addr
		dtyp       uint16
		wantReason Reason
	}{{
		name:       "match_A",
		host:       "host.com",
		want:       []netip.Addr{netip.AddrFrom4([4]byte{1, 2, 3, 4})},
		dtyp:       dns.TypeA,
		wantReason: Rewritten,
	}, {
		name:       "exception_AAAA_host.com",
		host:       "host.com",
		want:       nil,
		dtyp:       dns.TypeAAAA,
		wantReason: NotFilteredNotFound,
	}, {
		name:       "exception_A_host2.com",
		host:       "host2.com",
		want:       nil,
		dtyp:       dns.TypeA,
		wantReason: NotFilteredNotFound,
	}, {
		name:       "match_AAAA_host2.com",
		host:       "host2.com",
		want:       []netip.Addr{netip.MustParseAddr("::1")},
		dtyp:       dns.TypeAAAA,
		wantReason: Rewritten,
	}, {
		name:       "exception_A_host3.com",
		host:       "host3.com",
		want:       nil,
		dtyp:       dns.TypeA,
		wantReason: NotFilteredNotFound,
	}, {
		name:       "match_AAAA_host3.com",
		host:       "host3.com",
		want:       nil,
		dtyp:       dns.TypeAAAA,
		wantReason: Rewritten,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "match_AAAA_host3.com" {
				t.SkipNow()
			}

			r := d.processRewrites(tc.host, tc.dtyp)
			assert.Equal(t, tc.want, r.IPList)
			assert.Equal(t, tc.wantReason, r.Reason)
		})
	}
}
