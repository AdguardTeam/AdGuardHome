package filtering

import (
	"fmt"
	"net/netip"
	"path"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_CheckHostRules_dnsrewrite(t *testing.T) {
	const text = `
|cname^$dnsrewrite=new-cname

|a-record^$dnsrewrite=127.0.0.1
|aaaa-record^$dnsrewrite=::1

|txt-record^$dnsrewrite=NOERROR;TXT;hello-world
|refused^$dnsrewrite=REFUSED

|mapped^$dnsrewrite=NOERROR;AAAA;::ffff:127.0.0.1

|a-records^$dnsrewrite=127.0.0.1
|a-records^$dnsrewrite=127.0.0.2

|aaaa-records^$dnsrewrite=::1
|aaaa-records^$dnsrewrite=::2

|disable-one^$dnsrewrite=127.0.0.1
|disable-one^$dnsrewrite=127.0.0.2
@@||disable-one^$dnsrewrite=127.0.0.1

|disable-cname^$dnsrewrite=127.0.0.1
|disable-cname^$dnsrewrite=new-cname
@@||disable-cname^$dnsrewrite=new-cname

|disable-cname-many^$dnsrewrite=127.0.0.1
|disable-cname-many^$dnsrewrite=new-cname-1
|disable-cname-many^$dnsrewrite=new-cname-2
@@||disable-cname-many^$dnsrewrite=new-cname-1

|disable-all^$dnsrewrite=127.0.0.1
|disable-all^$dnsrewrite=127.0.0.2
@@||disable-all^$dnsrewrite

|1.2.3.4.in-addr.arpa^$dnsrewrite=NOERROR;PTR;new-ptr
|1.2.3.5.in-addr.arpa^$dnsrewrite=NOERROR;PTR;new-ptr-with-dot.
`

	f, _ := newForTest(t, nil, []Filter{{ID: 0, Data: []byte(text)}})
	setts := &Settings{
		FilteringEnabled: true,
	}

	ipv4p1 := netutil.IPv4Localhost()
	ipv4p2 := ipv4p1.Next()
	ipv6p1 := netutil.IPv6Localhost()
	ipv6p2 := ipv6p1.Next()
	mapped := netip.AddrFrom16(ipv4p1.As16())

	testCasesA := []struct {
		name  string
		want  []any
		rcode int
		dtyp  uint16
	}{{
		name:  "a-record",
		rcode: dns.RcodeSuccess,
		want:  []any{ipv4p1},
		dtyp:  dns.TypeA,
	}, {
		name:  "aaaa-record",
		want:  []any{ipv6p1},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeAAAA,
	}, {
		name:  "txt-record",
		want:  []any{"hello-world"},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeTXT,
	}, {
		name:  "refused",
		want:  nil,
		rcode: dns.RcodeRefused,
		dtyp:  0,
	}, {
		name:  "a-records",
		want:  []any{ipv4p1, ipv4p2},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeA,
	}, {
		name:  "aaaa-records",
		want:  []any{ipv6p1, ipv6p2},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeAAAA,
	}, {
		name:  "disable-one",
		want:  []any{ipv4p2},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeA,
	}, {
		name:  "disable-cname",
		want:  []any{ipv4p1},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeA,
	}, {
		name:  "mapped",
		want:  []any{mapped},
		rcode: dns.RcodeSuccess,
		dtyp:  dns.TypeAAAA,
	}}

	for _, tc := range testCasesA {
		t.Run(tc.name, func(t *testing.T) {
			host := path.Base(tc.name)

			res, err := f.CheckHostRules(host, tc.dtyp, setts)
			require.NoError(t, err)

			dnsrr := res.DNSRewriteResult
			require.NotNil(t, dnsrr)

			assert.Equal(t, tc.rcode, dnsrr.RCode)
			if tc.rcode == dns.RcodeRefused {
				return
			}

			ipVals := dnsrr.Response[tc.dtyp]
			require.Len(t, ipVals, len(tc.want))

			for i, val := range tc.want {
				require.Equal(t, val, ipVals[i])
			}
		})
	}

	t.Run("cname", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.NoError(t, err)

		assert.Equal(t, "new-cname", res.CanonName)
	})

	t.Run("disable-cname-many", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.NoError(t, err)

		assert.Equal(t, "new-cname-2", res.CanonName)
		assert.Nil(t, res.DNSRewriteResult)
	})

	t.Run("disable-all", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.NoError(t, err)

		assert.Empty(t, res.CanonName)
		assert.Empty(t, res.Rules)
	})

	t.Run("1.2.3.4.in-addr.arpa", func(t *testing.T) {
		dtyp := dns.TypePTR
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.NoError(t, err)
		require.NotNil(t, res.DNSRewriteResult)

		rr := res.DNSRewriteResult
		require.NotEmpty(t, rr.Response)

		resps := rr.Response[dtyp]
		require.Len(t, resps, 1)

		ptr, ok := resps[0].(string)
		require.True(t, ok)

		assert.Equal(t, "new-ptr.", ptr)
	})

	t.Run("1.2.3.5.in-addr.arpa", func(t *testing.T) {
		dtyp := dns.TypePTR
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.NoError(t, err)
		require.NotNil(t, res.DNSRewriteResult)

		rr := res.DNSRewriteResult
		require.NotEmpty(t, rr.Response)

		resps := rr.Response[dtyp]
		require.Len(t, resps, 1)

		ptr, ok := resps[0].(string)
		require.True(t, ok)

		assert.Equal(t, "new-ptr-with-dot.", ptr)
	})
}

func TestDNSFilter_CheckHost_hostsContainer(t *testing.T) {
	addrv4 := netip.MustParseAddr("1.2.3.4")
	addrv6 := netip.MustParseAddr("::1")
	addrMapped := netip.MustParseAddr("::ffff:1.2.3.4")

	data := fmt.Sprintf(
		""+
			"%s v4.host.example\n"+
			"%s v6.host.example\n"+
			"%s mapped.host.example\n",
		addrv4,
		addrv6,
		addrMapped,
	)

	files := fstest.MapFS{
		"hosts": &fstest.MapFile{
			Data: []byte(data),
		},
	}
	watcher := &aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { return nil },
	}
	hc, err := aghnet.NewHostsContainer(files, watcher, "hosts")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	f, _ := newForTest(t, &Config{EtcHosts: hc}, nil)
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
			FilterListID: SysHostsListID,
		}},
		wantResps: []rules.RRValue{addrv4},
	}, {
		name: "v6",
		host: "v6.host.example",
		dtyp: dns.TypeAAAA,
		wantRules: []*ResultRule{{
			Text:         "::1 v6.host.example",
			FilterListID: SysHostsListID,
		}},
		wantResps: []rules.RRValue{addrv6},
	}, {
		name: "mapped",
		host: "mapped.host.example",
		dtyp: dns.TypeAAAA,
		wantRules: []*ResultRule{{
			Text:         "::ffff:1.2.3.4 mapped.host.example",
			FilterListID: SysHostsListID,
		}},
		wantResps: []rules.RRValue{addrMapped},
	}, {
		name: "ptr",
		host: "4.3.2.1.in-addr.arpa",
		dtyp: dns.TypePTR,
		wantRules: []*ResultRule{{
			Text:         "1.2.3.4 v4.host.example",
			FilterListID: SysHostsListID,
		}},
		wantResps: []rules.RRValue{"v4.host.example"},
	}, {
		name: "ptr-mapped",
		host: "4.0.3.0.2.0.1.0.f.f.f.f.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		dtyp: dns.TypePTR,
		wantRules: []*ResultRule{{
			Text:         "::ffff:1.2.3.4 mapped.host.example",
			FilterListID: SysHostsListID,
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
		name:      "v4_mismatch",
		host:      "v4.host.example",
		dtyp:      dns.TypeAAAA,
		wantRules: nil,
		wantResps: nil,
	}, {
		name:      "v6_mismatch",
		host:      "v6.host.example",
		dtyp:      dns.TypeA,
		wantRules: nil,
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
