package querylog

import (
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestQueryLog(t *testing.T) {
	conf := Config{
		Enabled:  true,
		Interval: 1,
	}
	l := newQueryLog(conf)

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

	params := getDataParams{
		OlderThan: time.Now(),
	}
	d := l.getData(params)
	mdata := d["data"].([]map[string]interface{})
	m := mdata[0]
	mq := m["question"].(map[string]interface{})
	assert.True(t, mq["host"].(string) == "example.org")
}

func TestJSON(t *testing.T) {
	s := `
	{"keystr":"val","obj":{"keybool":true,"keyint":123456}}
	`
	k, v, jtype := readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTStr))
	assert.Equal(t, "keystr", k)
	assert.Equal(t, "val", v)

	k, v, jtype = readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTObj))
	assert.Equal(t, "obj", k)

	k, v, jtype = readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTBool))
	assert.Equal(t, "keybool", k)
	assert.Equal(t, "true", v)

	k, v, jtype = readJSON(&s)
	assert.Equal(t, jtype, int32(jsonTNum))
	assert.Equal(t, "keyint", k)
	assert.Equal(t, "123456", v)

	k, v, jtype = readJSON(&s)
	assert.True(t, jtype == jsonTErr)
}
