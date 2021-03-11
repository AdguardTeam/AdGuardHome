package dnsforward

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockedIP(t *testing.T) {
	const (
		ip int = iota
		cidr
	)

	rules := []string{
		ip:   "1.1.1.1",
		cidr: "2.2.0.0/16",
	}

	testCases := []struct {
		name     string
		allowed  bool
		ip       net.IP
		wantDis  bool
		wantRule string
	}{{
		name:     "allow_ip",
		allowed:  true,
		ip:       net.IPv4(1, 1, 1, 1),
		wantDis:  false,
		wantRule: "",
	}, {
		name:     "disallow_ip",
		allowed:  true,
		ip:       net.IPv4(1, 1, 1, 2),
		wantDis:  true,
		wantRule: "",
	}, {
		name:     "allow_cidr",
		allowed:  true,
		ip:       net.IPv4(2, 2, 1, 1),
		wantDis:  false,
		wantRule: "",
	}, {
		name:     "disallow_cidr",
		allowed:  true,
		ip:       net.IPv4(2, 3, 1, 1),
		wantDis:  true,
		wantRule: "",
	}, {
		name:     "allow_ip",
		allowed:  false,
		ip:       net.IPv4(1, 1, 1, 1),
		wantDis:  true,
		wantRule: rules[ip],
	}, {
		name:     "disallow_ip",
		allowed:  false,
		ip:       net.IPv4(1, 1, 1, 2),
		wantDis:  false,
		wantRule: "",
	}, {
		name:     "allow_cidr",
		allowed:  false,
		ip:       net.IPv4(2, 2, 1, 1),
		wantDis:  true,
		wantRule: rules[cidr],
	}, {
		name:     "disallow_cidr",
		allowed:  false,
		ip:       net.IPv4(2, 3, 1, 1),
		wantDis:  false,
		wantRule: "",
	}}

	for _, tc := range testCases {
		prefix := "allowed_"
		if !tc.allowed {
			prefix = "disallowed_"
		}

		t.Run(prefix+tc.name, func(t *testing.T) {
			aCtx := &accessCtx{}
			allowedRules := rules
			var disallowedRules []string

			if !tc.allowed {
				allowedRules, disallowedRules = disallowedRules, allowedRules
			}

			require.Nil(t, aCtx.Init(allowedRules, disallowedRules, nil))

			disallowed, rule := aCtx.IsBlockedIP(tc.ip)
			assert.Equal(t, tc.wantDis, disallowed)
			assert.Equal(t, tc.wantRule, rule)
		})
	}
}

func TestIsBlockedDomain(t *testing.T) {
	aCtx := &accessCtx{}
	require.Nil(t, aCtx.Init(nil, nil, []string{
		"host1",
		"*.host.com",
		"||host3.com^",
	}))

	testCases := []struct {
		name   string
		domain string
		want   bool
	}{{
		name:   "plain_match",
		domain: "host1",
		want:   true,
	}, {
		name:   "plain_mismatch",
		domain: "host2",
		want:   false,
	}, {
		name:   "wildcard_type-1_match_short",
		domain: "asdf.host.com",
		want:   true,
	}, {
		name:   "wildcard_type-1_match_long",
		domain: "qwer.asdf.host.com",
		want:   true,
	}, {
		name:   "wildcard_type-1_mismatch_no-lead",
		domain: "host.com",
		want:   false,
	}, {
		name:   "wildcard_type-1_mismatch_bad-asterisk",
		domain: "asdf.zhost.com",
		want:   false,
	}, {
		name:   "wildcard_type-2_match_simple",
		domain: "host3.com",
		want:   true,
	}, {
		name:   "wildcard_type-2_match_complex",
		domain: "asdf.host3.com",
		want:   true,
	}, {
		name:   "wildcard_type-2_mismatch",
		domain: ".host3.com",
		want:   false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, aCtx.IsBlockedDomain(tc.domain))
		})
	}
}
