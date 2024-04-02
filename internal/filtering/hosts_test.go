package filtering

import (
	"fmt"
	"net/netip"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_CheckHost_hostsContainer(t *testing.T) {
	addrv4 := netip.MustParseAddr("1.2.3.4")
	addrv6 := netip.MustParseAddr("::1")
	addrMapped := netip.MustParseAddr("::ffff:1.2.3.4")
	addrv4Dup := netip.MustParseAddr("4.3.2.1")

	data := fmt.Sprintf(
		""+
			"%[1]s v4.host.example\n"+
			"%[2]s v6.host.example\n"+
			"%[3]s mapped.host.example\n"+
			"%[4]s v4.host.with-dup\n"+
			"%[4]s v4.host.with-dup\n",
		addrv4,
		addrv6,
		addrMapped,
		addrv4Dup,
	)

	files := fstest.MapFS{
		"hosts": &fstest.MapFile{
			Data: []byte(data),
		},
	}
	watcher := &aghtest.FSWatcher{
		OnStart:  func() (_ error) { panic("not implemented") },
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { return nil },
	}
	hc, err := aghnet.NewHostsContainer(files, watcher, "hosts")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	conf := &Config{
		EtcHosts: hc,
	}
	f, err := New(conf, nil)
	require.NoError(t, err)

	setts := &Settings{
		FilteringEnabled: true,
	}

	testCases := []struct {
		name      string
		host      string
		wantRules []*ResultRule
		wantResps []rules.RRValue
		dtyp      uint16
	}{{
		name: "v4",
		host: "v4.host.example",
		dtyp: dns.TypeA,
		wantRules: []*ResultRule{{
			Text:         "1.2.3.4 v4.host.example",
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: []rules.RRValue{addrv4},
	}, {
		name: "v6",
		host: "v6.host.example",
		dtyp: dns.TypeAAAA,
		wantRules: []*ResultRule{{
			Text:         "::1 v6.host.example",
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: []rules.RRValue{addrv6},
	}, {
		name: "mapped",
		host: "mapped.host.example",
		dtyp: dns.TypeAAAA,
		wantRules: []*ResultRule{{
			Text:         "::ffff:1.2.3.4 mapped.host.example",
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: []rules.RRValue{addrMapped},
	}, {
		name: "ptr",
		host: "4.3.2.1.in-addr.arpa",
		dtyp: dns.TypePTR,
		wantRules: []*ResultRule{{
			Text:         "1.2.3.4 v4.host.example",
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: []rules.RRValue{"v4.host.example"},
	}, {
		name: "ptr-mapped",
		host: "4.0.3.0.2.0.1.0.f.f.f.f.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		dtyp: dns.TypePTR,
		wantRules: []*ResultRule{{
			Text:         "::ffff:1.2.3.4 mapped.host.example",
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: []rules.RRValue{"mapped.host.example"},
	}, {
		name:      "not_found_v4",
		host:      "non.existent.example",
		dtyp:      dns.TypeA,
		wantRules: nil,
		wantResps: nil,
	}, {
		name:      "not_found_v6",
		host:      "non.existent.example",
		dtyp:      dns.TypeAAAA,
		wantRules: nil,
		wantResps: nil,
	}, {
		name:      "not_found_ptr",
		host:      "4.3.2.2.in-addr.arpa",
		dtyp:      dns.TypePTR,
		wantRules: nil,
		wantResps: nil,
	}, {
		name: "v4_mismatch",
		host: "v4.host.example",
		dtyp: dns.TypeAAAA,
		wantRules: []*ResultRule{{
			Text:         fmt.Sprintf("%s v4.host.example", addrv4),
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: nil,
	}, {
		name: "v6_mismatch",
		host: "v6.host.example",
		dtyp: dns.TypeA,
		wantRules: []*ResultRule{{
			Text:         fmt.Sprintf("%s v6.host.example", addrv6),
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: nil,
	}, {
		name:      "wrong_ptr",
		host:      "4.3.2.1.ip6.arpa",
		dtyp:      dns.TypePTR,
		wantRules: nil,
		wantResps: nil,
	}, {
		name:      "unsupported_type",
		host:      "v4.host.example",
		dtyp:      dns.TypeCNAME,
		wantRules: nil,
		wantResps: nil,
	}, {
		name: "v4_dup",
		host: "v4.host.with-dup",
		dtyp: dns.TypeA,
		wantRules: []*ResultRule{{
			Text:         "4.3.2.1 v4.host.with-dup",
			FilterListID: rulelist.URLFilterIDEtcHosts,
		}},
		wantResps: []rules.RRValue{addrv4Dup},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var res Result
			res, err = f.CheckHost(tc.host, tc.dtyp, setts)
			require.NoError(t, err)

			if len(tc.wantRules) == 0 {
				assert.Empty(t, res.Rules)
				assert.Nil(t, res.DNSRewriteResult)

				return
			}

			require.NotNil(t, res.DNSRewriteResult)
			require.Contains(t, res.DNSRewriteResult.Response, tc.dtyp)

			assert.Equal(t, tc.wantResps, res.DNSRewriteResult.Response[tc.dtyp])
			assert.Equal(t, tc.wantRules, res.Rules)
		})
	}
}
