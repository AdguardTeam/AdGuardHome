package rewrite

import (
	"net"
	"testing"

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
	items := []*Item{{
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
	}, {
		Domain: "*.hostboth.com",
		Answer: "1.2.3.6",
	}, {
		Domain: "*.hostboth.com",
		Answer: "1234::5678",
	}, {
		Domain: "BIGHOST.COM",
		Answer: "1.2.3.7",
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
			Value:    net.IP{1, 2, 3, 4}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}, {
			Value:    net.IP{1, 2, 3, 5}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "rewritten_aaaa",
		host: "www.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.ParseIP("1:2:3::4"),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeAAAA,
		}},
		dtyp: dns.TypeAAAA,
	}, {
		name: "wildcard_match",
		host: "abc.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.IP{1, 2, 3, 5}.To16(),
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
			Value:    net.IP{1, 2, 3, 4}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}, {
			Value:    net.IP{1, 2, 3, 5}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "two_cnames",
		host: "b.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.IP{0, 0, 0, 0}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "two_cnames_and_wildcard",
		host: "b.host3.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.IP{1, 2, 3, 5}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "issue3343",
		host: "www.hostboth.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.ParseIP("1234::5678"),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeAAAA,
		}},
		dtyp: dns.TypeAAAA,
	}, {
		name: "issue3351",
		host: "bighost.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.IP{1, 2, 3, 7}.To16(),
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
	// Exact host, wildcard L2, wildcard L3.
	items := []*Item{{
		Domain: "host.com",
		Answer: "1.1.1.1",
	}, {
		Domain: "*.host.com",
		Answer: "2.2.2.2",
	}, {
		Domain: "*.sub.host.com",
		Answer: "3.3.3.3",
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
			Value:    net.IP{1, 1, 1, 1}.To16(),
			NewCNAME: "",
			RCode:    dns.RcodeSuccess,
			RRType:   dns.TypeA,
		}},
		dtyp: dns.TypeA,
	}, {
		name: "l2_match",
		host: "sub.host.com",
		wantDNSRewrites: []*rules.DNSRewrite{{
			Value:    net.IP{2, 2, 2, 2}.To16(),
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
		//		Value:    net.IP{3, 3, 3, 3}.To16(),
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
	// Wildcard and exception for a sub-domain.
	items := []*Item{{
		Domain: "*.host.com",
		Answer: "2.2.2.2",
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
			Value:    net.IP{2, 2, 2, 2}.To16(),
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
	// Exception for AAAA record.
	items := []*Item{{
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
			Value:    net.IP{1, 2, 3, 4}.To16(),
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
			Value:    net.ParseIP("::1"),
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
