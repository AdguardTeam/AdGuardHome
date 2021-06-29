package dnsforward

import (
	"net"
	"testing"

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
	})
	require.NoError(t, err)

	testCases := []struct {
		name string
		host string
		want bool
	}{{
		name: "plain_match",
		host: "host1",
		want: true,
	}, {
		name: "plain_mismatch",
		host: "host2",
		want: false,
	}, {
		name: "subdomain_match_short",
		host: "asdf.host.com",
		want: true,
	}, {
		name: "subdomain_match_long",
		host: "qwer.asdf.host.com",
		want: true,
	}, {
		name: "subdomain_mismatch_no_lead",
		host: "host.com",
		want: false,
	}, {
		name: "subdomain_mismatch_bad_asterisk",
		host: "asdf.zhost.com",
		want: false,
	}, {
		name: "rule_match_simple",
		host: "host3.com",
		want: true,
	}, {
		name: "rule_match_complex",
		host: "asdf.host3.com",
		want: true,
	}, {
		name: "rule_mismatch",
		host: ".host3.com",
		want: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, a.isBlockedHost(tc.host))
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
		name        string
		wantRule    string
		ip          net.IP
		wantBlocked bool
	}{{
		name:        "match_ip",
		wantRule:    "1.2.3.4",
		ip:          net.IP{1, 2, 3, 4},
		wantBlocked: true,
	}, {
		name:        "match_cidr",
		wantRule:    "5.6.7.8/24",
		ip:          net.IP{5, 6, 7, 100},
		wantBlocked: true,
	}, {
		name:        "no_match_ip",
		wantRule:    "",
		ip:          net.IP{9, 2, 3, 4},
		wantBlocked: false,
	}, {
		name:        "no_match_cidr",
		wantRule:    "",
		ip:          net.IP{9, 6, 7, 100},
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
