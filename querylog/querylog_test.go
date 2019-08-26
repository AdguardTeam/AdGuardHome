package querylog

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestQueryLog(t *testing.T) {
	conf := Config{
		Interval: 1,
	}
	l := New(conf)

	q := dns.Msg{}
	q.Question = append(q.Question, dns.Question{
		Name:   "example.org.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	})

	a := dns.Msg{}
	a.Question = append(a.Question, q.Question[0])
	answer := new(dns.A)
	answer.Hdr = dns.RR_Header{
		Name:   q.Question[0].Name,
		Rrtype: dns.TypeA,
		Class:  dns.ClassINET,
	}
	answer.A = net.IP{1, 2, 3, 4}
	a.Answer = append(a.Answer, answer)

	res := dnsfilter.Result{}
	l.Add(&q, &a, &res, 0, nil, "upstream")

	d := l.GetData()
	m := d[0]
	mq := m["question"].(map[string]interface{})
	assert.True(t, mq["host"].(string) == "example.org")
}
