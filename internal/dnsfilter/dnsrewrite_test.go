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
|cname^$dnsrewrite=new_cname

|a_record^$dnsrewrite=127.0.0.1

|aaaa_record^$dnsrewrite=::1

|txt_record^$dnsrewrite=NOERROR;TXT;hello_world

|refused^$dnsrewrite=REFUSED

|a_records^$dnsrewrite=127.0.0.1
|a_records^$dnsrewrite=127.0.0.2

|aaaa_records^$dnsrewrite=::1
|aaaa_records^$dnsrewrite=::2

|disable_one^$dnsrewrite=127.0.0.1
|disable_one^$dnsrewrite=127.0.0.2
@@||disable_one^$dnsrewrite=127.0.0.1

|disable_cname^$dnsrewrite=127.0.0.1
|disable_cname^$dnsrewrite=new_cname
@@||disable_cname^$dnsrewrite=new_cname

|disable_cname_many^$dnsrewrite=127.0.0.1
|disable_cname_many^$dnsrewrite=new_cname_1
|disable_cname_many^$dnsrewrite=new_cname_2
@@||disable_cname_many^$dnsrewrite=new_cname_1

|disable_all^$dnsrewrite=127.0.0.1
|disable_all^$dnsrewrite=127.0.0.2
@@||disable_all^$dnsrewrite
`
	f := NewForTest(nil, []Filter{{ID: 0, Data: []byte(text)}})
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
		assert.Equal(t, "new_cname", res.CanonName)
	})

	t.Run("a_record", func(t *testing.T) {
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

	t.Run("aaaa_record", func(t *testing.T) {
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

	t.Run("txt_record", func(t *testing.T) {
		dtyp := dns.TypeTXT
		host := path.Base(t.Name())
		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if strVals := dnsrr.Response[dtyp]; assert.Len(t, strVals, 1) {
				assert.Equal(t, "hello_world", strVals[0])
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

	t.Run("a_records", func(t *testing.T) {
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

	t.Run("aaaa_records", func(t *testing.T) {
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

	t.Run("disable_one", func(t *testing.T) {
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

	t.Run("disable_cname", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Equal(t, "", res.CanonName)

		if dnsrr := res.DNSRewriteResult; assert.NotNil(t, dnsrr) {
			assert.Equal(t, dns.RcodeSuccess, dnsrr.RCode)
			if ipVals := dnsrr.Response[dtyp]; assert.Len(t, ipVals, 1) {
				assert.Equal(t, ipv4p1, ipVals[0])
			}
		}
	})

	t.Run("disable_cname_many", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Equal(t, "new_cname_2", res.CanonName)
		assert.Nil(t, res.DNSRewriteResult)
	})

	t.Run("disable_all", func(t *testing.T) {
		dtyp := dns.TypeA
		host := path.Base(t.Name())

		res, err := f.CheckHostRules(host, dtyp, setts)
		assert.Nil(t, err)
		assert.Equal(t, "", res.CanonName)
		assert.Len(t, res.Rules, 0)
	})
}
