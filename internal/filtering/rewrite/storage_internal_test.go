package rewrite

import (
	"net/netip"
	"testing"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultStorage(t *testing.T) {
	items := []*Item{{
		Domain: "example.com",
		Answer: "answer.com",
	}}

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)

	require.Len(t, s.List(), 1)
}

func TestDefaultStorage_CRUD(t *testing.T) {
	var items []*Item

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)
	require.Len(t, s.List(), 0)

	item := &Item{Domain: "example.com", Answer: "answer.com"}

	err = s.Add(item)
	require.NoError(t, err)

	list := s.List()
	require.Len(t, list, 1)
	require.True(t, item.equal(list[0]))

	err = s.Remove(item)
	require.NoError(t, err)
	require.Len(t, s.List(), 0)
}

func TestDefaultStorage_MatchRequest(t *testing.T) {
	var (
		addr1v4 = netip.AddrFrom4([4]byte{1, 2, 3, 4})
		addr2v4 = netip.AddrFrom4([4]byte{1, 2, 3, 5})
		addr3v4 = netip.AddrFrom4([4]byte{1, 2, 3, 6})
		addr4v4 = netip.AddrFrom4([4]byte{1, 2, 3, 7})

		addr1v6 = netip.MustParseAddr("1:2:3::4")
		addr2v6 = netip.MustParseAddr("1234::5678")
	)

	items := []*Item{{
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
	}}

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)

	testCases := []struct {
		name            string
		host            string
		wantDNSRewrites []*rules.DNSRewrite
		dtyp            uint16
	}{{
		name:            "not_filtered_not_found",
		host:            "hoost.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeA,
	}, {
		name:            "not_filtered_qtype",
		host:            "www.host.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeMX,
	}, {
		name: "rewritten_a",
		host: "www.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr1v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}, {
			Value:    addr2v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "rewritten_aaaa",
		host: "www.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr1v6,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeAAAA,
		}},
		dtyp: dns.TypeAAAA,
	}, {
		name: "wildcard_match",
		host: "abc.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr2v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
		//}, {
		// TODO(d.kolyshev): This is about matching in urlfilter.
		//	name: "wildcard_override",
		//	host: "a.host.com",
		//	wantDNSRewrites: []*rules.DNSRewrite{{
		//		Value:    net.IP{1, 2, 3, 4}.To16(),
		//		NewCNAME: "",
		//		RCode:    dns.RcodeSuccess,
		//		RRType:   dns.TypeA,
		//	}},
		//	dtyp: dns.TypeA,
	}, {
		name: "wildcard_cname_interaction",
		host: "www.host2.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr1v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}, {
			Value:    addr2v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "two_cnames",
		host: "b.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    netip.IPv4Unspecified(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "two_cnames_and_wildcard",
		host: "b.host3.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr2v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "issue3343",
		host: "www.hostboth.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr2v6,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeAAAA,
		}},
		dtyp: dns.TypeAAAA,
	}, {
		name: "issue3351",
		host: "bighost.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr4v4,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name:            "issue4008",
		host:            "somehost.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeHTTPS,
	}, {
		name: "issue4016",
		host: "www.issue4016.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    nil,
			NewCNAME: "sub.issue4016.com",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeNone,
		}},
		dtyp: dns.TypeA,
	}, {
		name:            "issue4016_self",
		host:            "sub.issue4016.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeA,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dnsRewrites := s.MatchRequest(&urlfilter.DNSRequest{
				Hostname: tc.host,
				DNSType:  tc.dtyp,
			})

			assert.Equal(t, tc.wantDNSRewrites, dnsRewrites)
		})
	}
}

func TestDefaultStorage_MatchRequest_Levels(t *testing.T) {
	var (
		addr1 = netip.AddrFrom4([4]byte{1, 1, 1, 1})
		addr2 = netip.AddrFrom4([4]byte{2, 2, 2, 2})
		addr3 = netip.AddrFrom4([4]byte{3, 3, 3, 3})
	)

	// Exact host, wildcard L2, wildcard L3.
	items := []*Item{{
		Domain: "host.com",
		Answer: addr1.String(),
	}, {
		Domain: "*.host.com",
		Answer: addr2.String(),
	}, {
		Domain: "*.sub.host.com",
		Answer: addr3.String(),
	}}

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)

	testCases := []struct {
		name            string
		host            string
		wantDNSRewrites []*rules.DNSRewrite
		dtyp            uint16
	}{{
		name: "exact_match",
		host: "host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr1,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "l2_match",
		host: "sub.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr2,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
		//}, {
		// TODO(d.kolyshev): This is about matching in urlfilter.
		//	name: "l3_match",
		//	host: "my.sub.host.com",
		//	wantDNSRewrites: []*rules.DNSRewrite{{
		//		Value:    addr3,
		//		NewCNAME: "",
		//		RCode:    dns.RcodeSuccess,
		//		RRType:   dns.TypeA,
		//	}},
		//	dtyp: dns.TypeA,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dnsRewrites := s.MatchRequest(&urlfilter.DNSRequest{
				Hostname: tc.host,
				DNSType:  tc.dtyp,
			})

			assert.Equal(t, tc.wantDNSRewrites, dnsRewrites)
		})
	}
}

func TestDefaultStorage_MatchRequest_ExceptionCNAME(t *testing.T) {
	addr := netip.AddrFrom4([4]byte{2, 2, 2, 2})

	// Wildcard and exception for a sub-domain.
	items := []*Item{{
		Domain: "*.host.com",
		Answer: addr.String(),
	}, {
		Domain: "sub.host.com",
		Answer: "sub.host.com",
	}, {
		Domain: "*.sub.host.com",
		Answer: "*.sub.host.com",
	}}

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)

	testCases := []struct {
		name            string
		host            string
		wantDNSRewrites []*rules.DNSRewrite
		dtyp            uint16
	}{{
		name: "match_subdomain",
		host: "my.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name:            "exception_cname",
		host:            "sub.host.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeA,
		//}, {
		// TODO(d.kolyshev): This is about matching in urlfilter.
		//	name:            "exception_wildcard",
		//	host:            "my.sub.host.com",
		//	wantDNSRewrites: nil,
		//	dtyp:            dns.TypeA,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dnsRewrites := s.MatchRequest(&urlfilter.DNSRequest{
				Hostname: tc.host,
				DNSType:  tc.dtyp,
			})

			assert.Equal(t, tc.wantDNSRewrites, dnsRewrites)
		})
	}
}

func TestDefaultStorage_MatchRequest_ExceptionIP(t *testing.T) {
	addr := netip.AddrFrom4([4]byte{1, 2, 3, 4})

	// Exception for AAAA record.
	items := []*Item{{
		Domain: "host.com",
		Answer: addr.String(),
	}, {
		Domain: "host.com",
		Answer: "AAAA",
	}, {
		Domain: "host2.com",
		Answer: netutil.IPv6Localhost().String(),
	}, {
		Domain: "host2.com",
		Answer: "A",
	}, {
		Domain: "host3.com",
		Answer: "A",
	}}

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)

	testCases := []struct {
		name            string
		host            string
		wantDNSRewrites []*rules.DNSRewrite
		dtyp            uint16
	}{{
		name: "match_A",
		host: "host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    addr,
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name:            "exception_AAAA_host.com",
		host:            "host.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeAAAA,
	}, {
		name:            "exception_A_host2.com",
		host:            "host2.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeA,
	}, {
		name: "match_AAAA_host2.com",
		host: "host2.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    netutil.IPv6Localhost(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeAAAA,
		}},
		dtyp: dns.TypeAAAA,
	}, {
		name:            "exception_A_host3.com",
		host:            "host3.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeA,
	}, {
		name:            "match_AAAA_host3.com",
		host:            "host3.com",
		wantDNSRewrites: nil,
		dtyp:            dns.TypeAAAA,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dnsRewrites := s.MatchRequest(&urlfilter.DNSRequest{
				Hostname: tc.host,
				DNSType:  tc.dtyp,
			})

			assert.Equal(t, tc.wantDNSRewrites, dnsRewrites)
		})
	}
}
