package dnsforward

import (
	"net/netip"
	"testing"

	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockedClientID(t *testing.T) {
	clientID := "client-1"
	clients := []string{clientID}

	a, err := newAccessCtx(clients, nil, nil)
	require.NoError(t, err)

	assert.False(t, a.isBlockedClientID(clientID))

	a, err = newAccessCtx(nil, clients, nil)
	require.NoError(t, err)

	assert.True(t, a.isBlockedClientID(clientID))
}

func TestIsBlockedHost(t *testing.T) {
	a, err := newAccessCtx(nil, nil, []string{
		"host1",
		"*.host.com",
		"||host3.com^",
		"||*^$dnstype=HTTPS",
		"|.^",
	})
	require.NoError(t, err)

	testCases := []struct {
		want assert.BoolAssertionFunc
		name string
		host string
		qt   rules.RRType
	}{{
		want: assert.True,
		name: "plain_match",
		host: "host1",
		qt:   dns.TypeA,
	}, {
		want: assert.False,
		name: "plain_mismatch",
		host: "host2",
		qt:   dns.TypeA,
	}, {
		want: assert.True,
		name: "subdomain_match_short",
		host: "asdf.host.com",
		qt:   dns.TypeA,
	}, {
		want: assert.True,
		name: "subdomain_match_long",
		host: "qwer.asdf.host.com",
		qt:   dns.TypeA,
	}, {
		want: assert.False,
		name: "subdomain_mismatch_no_lead",
		host: "host.com",
		qt:   dns.TypeA,
	}, {
		want: assert.False,
		name: "subdomain_mismatch_bad_asterisk",
		host: "asdf.zhost.com",
		qt:   dns.TypeA,
	}, {
		want: assert.True,
		name: "rule_match_simple",
		host: "host3.com",
		qt:   dns.TypeA,
	}, {
		want: assert.True,
		name: "rule_match_complex",
		host: "asdf.host3.com",
		qt:   dns.TypeA,
	}, {
		want: assert.False,
		name: "rule_mismatch",
		host: ".host3.com",
		qt:   dns.TypeA,
	}, {
		want: assert.True,
		name: "by_qtype",
		host: "site-with-https-record.example",
		qt:   dns.TypeHTTPS,
	}, {
		want: assert.False,
		name: "by_qtype_other",
		host: "site-with-https-record.example",
		qt:   dns.TypeA,
	}, {
		want: assert.True,
		name: "ns_root",
		host: ".",
		qt:   dns.TypeNS,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.want(t, a.isBlockedHost(tc.host, tc.qt))
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	clients := []string{
		"1.2.3.4",
		"5.6.7.8/24",
	}

	allowCtx, err := newAccessCtx(clients, nil, nil)
	require.NoError(t, err)

	blockCtx, err := newAccessCtx(nil, clients, nil)
	require.NoError(t, err)

	testCases := []struct {
		ip          netip.Addr
		name        string
		wantRule    string
		wantBlocked bool
	}{{
		ip:          netip.MustParseAddr("1.2.3.4"),
		name:        "match_ip",
		wantRule:    "1.2.3.4",
		wantBlocked: true,
	}, {
		ip:          netip.MustParseAddr("5.6.7.100"),
		name:        "match_cidr",
		wantRule:    "5.6.7.8/24",
		wantBlocked: true,
	}, {
		ip:          netip.MustParseAddr("9.2.3.4"),
		name:        "no_match_ip",
		wantRule:    "",
		wantBlocked: false,
	}, {
		ip:          netip.MustParseAddr("9.6.7.100"),
		name:        "no_match_cidr",
		wantRule:    "",
		wantBlocked: false,
	}}

	t.Run("allow", func(t *testing.T) {
		for _, tc := range testCases {
			blocked, rule := allowCtx.isBlockedIP(tc.ip)
			assert.Equal(t, !tc.wantBlocked, blocked)
			assert.Equal(t, tc.wantRule, rule)
		}
	})

	t.Run("block", func(t *testing.T) {
		for _, tc := range testCases {
			blocked, rule := blockCtx.isBlockedIP(tc.ip)
			assert.Equal(t, tc.wantBlocked, blocked)
			assert.Equal(t, tc.wantRule, rule)
		}
	})
}
