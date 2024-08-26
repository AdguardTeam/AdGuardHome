package dnsforward

import (
	"context"
	"net"
	"testing"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

// fakeIpsetMgr is a fake aghnet.IpsetManager for tests.
type fakeIpsetMgr struct {
	ip4s []net.IP
	ip6s []net.IP
}

// Add implements the aghnet.IpsetManager interface for *fakeIpsetMgr.
func (m *fakeIpsetMgr) Add(_ context.Context, host string, ip4s, ip6s []net.IP) (n int, err error) {
	m.ip4s = append(m.ip4s, ip4s...)
	m.ip6s = append(m.ip6s, ip6s...)

	return len(ip4s) + len(ip6s), nil
}

// Close implements the aghnet.IpsetManager interface for *fakeIpsetMgr.
func (*fakeIpsetMgr) Close() (err error) {
	return nil
}

func TestIpsetCtx_process(t *testing.T) {
	ip4 := net.IP{1, 2, 3, 4}
	ip6 := net.IP{
		0x12, 0x34, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x56, 0x78,
	}

	req4 := createTestMessageWithType("example.com", dns.TypeA)
	req6 := createTestMessageWithType("example.com", dns.TypeAAAA)

	resp4 := &dns.Msg{
		Answer: []dns.RR{&dns.A{
			A: ip4,
		}},
	}
	resp6 := &dns.Msg{
		Answer: []dns.RR{&dns.AAAA{
			AAAA: ip6,
		}},
	}

	t.Run("nil", func(t *testing.T) {
		dctx := &dnsContext{
			proxyCtx: &proxy.DNSContext{},

			responseFromUpstream: true,
		}

		ictx := &ipsetHandler{
			logger: slogutil.NewDiscardLogger(),
		}
		rc := ictx.process(dctx)
		assert.Equal(t, resultCodeSuccess, rc)

		err := ictx.close()
		assert.NoError(t, err)
	})

	t.Run("ipv4", func(t *testing.T) {
		dctx := &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: req4,
				Res: resp4,
			},

			responseFromUpstream: true,
		}

		m := &fakeIpsetMgr{}
		ictx := &ipsetHandler{
			ipsetMgr: m,
			logger:   slogutil.NewDiscardLogger(),
		}

		rc := ictx.process(dctx)
		assert.Equal(t, resultCodeSuccess, rc)
		assert.Equal(t, []net.IP{ip4}, m.ip4s)
		assert.Empty(t, m.ip6s)

		err := ictx.close()
		assert.NoError(t, err)
	})

	t.Run("ipv6", func(t *testing.T) {
		dctx := &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: req6,
				Res: resp6,
			},

			responseFromUpstream: true,
		}

		m := &fakeIpsetMgr{}
		ictx := &ipsetHandler{
			ipsetMgr: m,
			logger:   slogutil.NewDiscardLogger(),
		}

		rc := ictx.process(dctx)
		assert.Equal(t, resultCodeSuccess, rc)
		assert.Empty(t, m.ip4s)
		assert.Equal(t, []net.IP{ip6}, m.ip6s)

		err := ictx.close()
		assert.NoError(t, err)
	})
}

func TestIpsetCtx_SkipIpsetProcessing(t *testing.T) {
	req4 := createTestMessage("example.com")
	resp4 := &dns.Msg{
		Answer: []dns.RR{&dns.A{
			A: net.IP{1, 2, 3, 4},
		}},
	}

	m := &fakeIpsetMgr{}
	ictx := &ipsetHandler{
		ipsetMgr: m,
		logger:   slogutil.NewDiscardLogger(),
	}

	testCases := []struct {
		dctx *dnsContext
		name string
		want bool
	}{{
		name: "basic",
		want: false,
		dctx: &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: req4,
				Res: resp4,
			},

			responseFromUpstream: true,
		},
	}, {
		name: "rewrite",
		want: true,
		dctx: &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: req4,
				Res: resp4,
			},

			responseFromUpstream: false,
		},
	}, {
		name: "empty_req",
		want: true,
		dctx: &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: nil,
				Res: resp4,
			},

			responseFromUpstream: true,
		},
	}, {
		name: "empty_res",
		want: true,
		dctx: &dnsContext{
			proxyCtx: &proxy.DNSContext{
				Req: req4,
				Res: nil,
			},

			responseFromUpstream: true,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ictx.skipIpsetProcessing(tc.dctx)
			assert.Equal(t, tc.want, got)
		})
	}
}
