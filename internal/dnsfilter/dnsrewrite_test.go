package dnsfilter

import (
	"net"
	"path"
	"testing"

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
`

	f := newForTest(nil, []Filter{{ID: 0, Data: []byte(text)}})
	setts := &FilteringSettings{
		FilteringEnabled: true,
	}

	ipv4p1 := net.IPv4(127, 0, 0, 1)
	ipv4p2 := net.IPv4(127, 0, 0, 2)
	ipv6p1 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	ipv6p2 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	testCasesA := []struct {
		name  string
		dtyp  uint16
		rcode int
		want  []interface{}
	}{{
		name:  "a-record",
		dtyp:  dns.TypeA,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{ipv4p1},
	}, {
		name:  "aaaa-record",
		dtyp:  dns.TypeAAAA,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{ipv6p1},
	}, {
		name:  "txt-record",
		dtyp:  dns.TypeTXT,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{"hello-world"},
	}, {
		name:  "refused",
		rcode: dns.RcodeRefused,
	}, {
		name:  "a-records",
		dtyp:  dns.TypeA,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{ipv4p1, ipv4p2},
	}, {
		name:  "aaaa-records",
		dtyp:  dns.TypeAAAA,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{ipv6p1, ipv6p2},
	}, {
		name:  "disable-one",
		dtyp:  dns.TypeA,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{ipv4p2},
	}, {
		name:  "disable-cname",
		dtyp:  dns.TypeA,
		rcode: dns.RcodeSuccess,
		want:  []interface{}{ipv4p1},
	}}

	for _, tc := range testCasesA {
		t.Run(tc.name, func(t *testing.T) {
			host := path.Base(tc.name)

			res, err := f.CheckHostRules(host, tc.dtyp, setts)
			require.Nil(t, err)

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
		require.Nil(t, err)
		assert.Equal(t, "new-cname", res.CanonName)
	})

	t.Run("disable-cname-many", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.Nil(t, err)
		assert.Equal(t, "new-cname-2", res.CanonName)
		assert.Nil(t, res.DNSRewriteResult)
	})

	t.Run("disable-all", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		require.Nil(t, err)
		assert.Empty(t, res.CanonName)
		assert.Empty(t, res.Rules)
	})
}
