package dnsforward

import (
	"net"
	"net/url"
	"strings"
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
			"[/domain.example/]/]1.2.3.4": generalTextLabel + " 1: parsing error",
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
			"[/example/]/]" + goodUps:        generalTextLabel + " 1: parsing error",
			"[/example/" + goodUps:           fallbackTextLabel + " 1: parsing error",
			"[/example//bad.123/]" + goodUps: privateTextLabel + " 1: parsing error",
		},
	}, {
		name: "bad_proto",
		general: []string{
			"bad://1.2.3.4",
		},
		want: map[string]string{
			"bad://1.2.3.4": generalTextLabel + " 1: parsing error",
		},
	}, {
		name: "truncated_line",
		general: []string{
			"This is a very long line.  It will cause a parsing error and will be truncated here.",
		},
		want: map[string]string{
			"This is a very long line.  It will cause a parsing error and will be truncated …": "upstream_dns 1: parsing error",
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
	type signal = struct{}

	reqCh := make(chan signal)
	hdlr := dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
		pt := testutil.PanicT{}

		err := w.WriteMsg(new(dns.Msg).SetReply(m))
		require.NoError(pt, err)

		testutil.RequireSend(pt, reqCh, signal{}, testTimeout)
	})

	addr := (&url.URL{
		Scheme: "tcp",
		Host:   newLocalUpstreamListener(t, 0, hdlr).String(),
	}).String()
	twoAddrs := strings.Join([]string{addr, addr}, " ")

	wantStatus := map[string]string{
		addr: "OK",
	}

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
			cv := newUpstreamConfigValidator(tc.ups, nil, nil, &upstream.Options{
				Timeout: testTimeout,
			})

			go func() {
				cv.check()
				testutil.RequireSend(testutil.PanicT{}, reqCh, signal{}, testTimeout)
			}()

			// Wait for the only request to be sent.
			testutil.RequireReceive(t, reqCh, testTimeout)

			// Wait for the check to finish.
			testutil.RequireReceive(t, reqCh, testTimeout)

			cv.close()
			require.Equal(t, wantStatus, cv.status())
		})
	}
}
