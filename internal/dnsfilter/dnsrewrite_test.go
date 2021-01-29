package dnsfilter

import (
	"net"
	"path"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
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
	setts := &RequestFilteringSettings{
		FilteringEnabled: true,
	}

	ipv4p1 := net.IPv4(127, 0, 0, 1)
	ipv4p2 := net.IPv4(127, 0, 0, 2)
	ipv6p1 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	ipv6p2 := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	t.Run("cname", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Equal(t, "new-cname", res.CanonName)
	})

	t.Run("a-record", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 1) {
				assert.Equal(t, ipv4p1, ipVals[0])
			}
		}
	})

	t.Run("aaaa-record", func(t *testing.T) {
		dtyp := dns.TypeAAAA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 1) {
				assert.Equal(t, ipv6p1, ipVals[0])
			}
		}
	})

	t.Run("txt-record", func(t *testing.T) {
		dtyp := dns.TypeTXT
		host := path.Base(t.Name())
		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if strVals := dnsrr.Response[dtyp]; assert.Len(t, strVals, 1) {
				assert.Equal(t, "hello-world", strVals[0])
			}
		}
	})

	t.Run("refused", func(t *testing.T) {
		host := path.Base(t.Name())
		res, err := f.CheckHostRules(host, dns.TypeA, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeRefused, dnsrr.RCode)
		}
	})

	t.Run("a-records", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 2) {
				assert.Equal(t, ipv4p1, ipVals[0])
				assert.Equal(t, ipv4p2, ipVals[1])
			}
		}
	})

	t.Run("aaaa-records", func(t *testing.T) {
		dtyp := dns.TypeAAAA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 2) {
				assert.Equal(t, ipv6p1, ipVals[0])
				assert.Equal(t, ipv6p2, ipVals[1])
			}
		}
	})

	t.Run("disable-one", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 1) {
				assert.Equal(t, ipv4p2, ipVals[0])
			}
		}
	})

	t.Run("disable-cname", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Empty(t, res.CanonName)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 1) {
				assert.Equal(t, ipv4p1, ipVals[0])
			}
		}
	})

	t.Run("disable-cname-many", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Equal(t, "new-cname-2", res.CanonName)
		assert.Nil(t, res.DNSRewriteResult)
	})

	t.Run("disable-all", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Empty(t, res.CanonName)
		assert.Empty(t, res.Rules)
	})
}
