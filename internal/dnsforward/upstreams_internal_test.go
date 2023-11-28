package dnsforward

import (
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamConfigValidator(t *testing.T) {
	goodHandler := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		err := w.WriteMsg(new(dns.Msg).SetReply(m))
		require.NoError(testutil.PanicT{}, err)
	})
	badHandler := dns.HandlerFunc(func(w dns.ResponseWriter, _ *dns.Msg) {
		err := w.WriteMsg(new(dns.Msg))
		require.NoError(testutil.PanicT{}, err)
	})

	goodUps := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, goodHandler).String(),
	}).String()
	badUps := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, badHandler).String(),
	}).String()

	goodAndBadUps := strings.Join([]string{goodUps, badUps}, " ")

	// upsTimeout restricts the checking process to prevent the test from
	// hanging.
	const upsTimeout = 100 * time.Millisecond

	testCases := []struct {
		want     map[string]string
		name     string
		general  []string
		fallback []string
		private  []string
	}{{
		name:    "success",
		general: []string{goodUps},
		want: map[string]string{
			goodUps: "OK",
		},
	}, {
		name:    "broken",
		general: []string{badUps},
		want: map[string]string{
			badUps: `couldn't communicate with upstream: exchanging with ` +
				badUps + ` over tcp: dns: id mismatch`,
		},
	}, {
		name:    "both",
		general: []string{goodUps, badUps, goodUps},
		want: map[string]string{
			goodUps: "OK",
			badUps: `couldn't communicate with upstream: exchanging with ` +
				badUps + ` over tcp: dns: id mismatch`,
		},
	}, {
		name:    "domain_specific_error",
		general: []string{"[/domain.example/]" + badUps},
		want: map[string]string{
			badUps: `WARNING: couldn't communicate ` +
				`with upstream: exchanging with ` + badUps + ` over tcp: ` +
				`dns: id mismatch`,
		},
	}, {
		name:     "fallback_success",
		fallback: []string{goodUps},
		want: map[string]string{
			goodUps: "OK",
		},
	}, {
		name:     "fallback_broken",
		fallback: []string{badUps},
		want: map[string]string{
			badUps: `couldn't communicate with upstream: exchanging with ` +
				badUps + ` over tcp: dns: id mismatch`,
		},
	}, {
		name:    "multiple_domain_specific_upstreams",
		general: []string{"[/domain.example/]" + goodAndBadUps},
		want: map[string]string{
			goodUps: "OK",
			badUps: `WARNING: couldn't communicate ` +
				`with upstream: exchanging with ` + badUps + ` over tcp: ` +
				`dns: id mismatch`,
		},
	}, {
		name:    "bad_specification",
		general: []string{"[/domain.example/]/]1.2.3.4"},
		want: map[string]string{
			"[/domain.example/]/]1.2.3.4": `splitting upstream line ` +
				`"[/domain.example/]/]1.2.3.4": duplicated separator`,
		},
	}, {
		name:     "all_different",
		general:  []string{"[/domain.example/]" + goodAndBadUps},
		fallback: []string{"[/domain.example/]" + goodAndBadUps},
		private:  []string{"[/domain.example/]" + goodAndBadUps},
		want: map[string]string{
			goodUps: "OK",
			badUps: `WARNING: couldn't communicate ` +
				`with upstream: exchanging with ` + badUps + ` over tcp: ` +
				`dns: id mismatch`,
		},
	}, {
		name:     "bad_specific_domains",
		general:  []string{"[/example/]/]" + goodUps},
		fallback: []string{"[/example/" + goodUps},
		private:  []string{"[/example//bad.123/]" + goodUps},
		want: map[string]string{
			`[/example/]/]` + goodUps: `splitting upstream line ` +
				`"[/example/]/]` + goodUps + `": duplicated separator`,
			`[/example/` + goodUps: `splitting upstream line ` +
				`"[/example/` + goodUps + `": missing separator`,
			`[/example//bad.123/]` + goodUps: `splitting upstream line ` +
				`"[/example//bad.123/]` + goodUps + `": domain at index 2: ` +
				`bad domain name "bad.123": ` +
				`bad top-level domain name label "123": all octets are numeric`,
		},
	}, {
		name: "non-specific_default",
		general: []string{
			"#",
			"[/example/]#",
		},
		want: map[string]string{
			"#": "not a domain-specific upstream",
		},
	}, {
		name: "bad_proto",
		general: []string{
			"bad://1.2.3.4",
		},
		want: map[string]string{
			"bad://1.2.3.4": `bad protocol "bad"`,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cv := newUpstreamConfigValidator(tc.general, tc.fallback, tc.private, &upstream.Options{
				Timeout:   upsTimeout,
				Bootstrap: net.DefaultResolver,
			})
			cv.check()
			cv.close()

			assert.Equal(t, tc.want, cv.status())
		})
	}
}

func TestUpstreamConfigValidator_Check_once(t *testing.T) {
	reqs := atomic.Int32{}
	reset := func() { reqs.Store(0) }

	hdlr := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		err := w.WriteMsg(new(dns.Msg).SetReply(m))
		require.NoError(testutil.PanicT{}, err)
		reqs.Add(1)
	})

	addr := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, hdlr).String(),
	}).String()
	twoAddrs := strings.Join([]string{addr, addr}, " ")

	testCases := []struct {
		name string
		ups  []string
	}{{
		name: "common",
		ups:  []string{addr, addr, addr},
	}, {
		name: "domain-specific",
		ups:  []string{"[/one.example/]" + addr, "[/two.example/]" + twoAddrs},
	}, {
		name: "both",
		ups:  []string{addr, "[/one.example/]" + addr, addr, "[/two.example/]" + twoAddrs},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(reset)

			cv := newUpstreamConfigValidator(tc.ups, nil, nil, &upstream.Options{
				Timeout: 100 * time.Millisecond,
			})
			cv.check()
			cv.close()

			assert.Equal(t, int32(1), reqs.Load())
		})
	}
}
