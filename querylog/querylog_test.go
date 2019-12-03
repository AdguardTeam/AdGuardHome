package querylog

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// Check adding and loading (with filtering) entries from disk and memory
func TestQueryLog(t *testing.T) {
	conf := Config{
		Enabled:  true,
		Interval: 1,
		MemSize:  100,
	}
	conf.BaseDir = prepareTestDir()
	defer func() { _ = os.RemoveAll(conf.BaseDir) }()
	l := newQueryLog(conf)

	// add disk entries
	addEntry(l, "example.org", "1.2.3.4", "0.1.2.3")
	addEntry(l, "example.org", "1.2.3.4", "0.1.2.3")

	// write to disk
	l.flushLogBuffer(true)

	// add memory entries
	addEntry(l, "test.example.org", "2.2.3.4", "0.1.2.4")

	// get all entries
	params := getDataParams{
		OlderThan: time.Time{},
	}
	d := l.getData(params)
	mdata := d["data"].([]map[string]interface{})
	assert.True(t, len(mdata) == 2)
	assert.True(t, checkEntry(t, mdata[0], "test.example.org", "2.2.3.4", "0.1.2.4"))
	assert.True(t, checkEntry(t, mdata[1], "example.org", "1.2.3.4", "0.1.2.3"))

	// search by domain (strict)
	params = getDataParams{
		OlderThan:         time.Time{},
		Domain:            "test.example.org",
		StrictMatchDomain: true,
	}
	d = l.getData(params)
	mdata = d["data"].([]map[string]interface{})
	assert.True(t, len(mdata) == 1)
	assert.True(t, checkEntry(t, mdata[0], "test.example.org", "2.2.3.4", "0.1.2.4"))

	// search by domain
	params = getDataParams{
		OlderThan:         time.Time{},
		Domain:            "example.org",
		StrictMatchDomain: false,
	}
	d = l.getData(params)
	mdata = d["data"].([]map[string]interface{})
	assert.True(t, len(mdata) == 2)
	assert.True(t, checkEntry(t, mdata[0], "test.example.org", "2.2.3.4", "0.1.2.4"))
	assert.True(t, checkEntry(t, mdata[1], "example.org", "1.2.3.4", "0.1.2.3"))

	// search by client IP (strict)
	params = getDataParams{
		OlderThan:         time.Time{},
		Client:            "0.1.2.3",
		StrictMatchClient: true,
	}
	d = l.getData(params)
	mdata = d["data"].([]map[string]interface{})
	assert.True(t, len(mdata) == 1)
	assert.True(t, checkEntry(t, mdata[0], "example.org", "1.2.3.4", "0.1.2.3"))

	// search by client IP
	params = getDataParams{
		OlderThan:         time.Time{},
		Client:            "0.1.2",
		StrictMatchClient: false,
	}
	d = l.getData(params)
	mdata = d["data"].([]map[string]interface{})
	assert.True(t, len(mdata) == 2)
	assert.True(t, checkEntry(t, mdata[0], "test.example.org", "2.2.3.4", "0.1.2.4"))
	assert.True(t, checkEntry(t, mdata[1], "example.org", "1.2.3.4", "0.1.2.3"))
}

func addEntry(l *queryLog, host, answerStr, client string) {
	q := dns.Msg{}
	q.Question = append(q.Question, dns.Question{
		Name:   host + ".",
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
	answer.A = net.ParseIP(answerStr)
	a.Answer = append(a.Answer, answer)
	res := dnsfilter.Result{}
	params := AddParams{
		Question: &q,
		Answer:   &a,
		Result:   &res,
		ClientIP: net.ParseIP(client),
		Upstream: "upstream",
	}
	l.Add(params)
}

func checkEntry(t *testing.T, m map[string]interface{}, host, answer, client string) bool {
	mq := m["question"].(map[string]interface{})
	ma := m["answer"].([]map[string]interface{})
	ma0 := ma[0]
	if !assert.True(t, mq["host"].(string) == host) ||
		!assert.True(t, mq["class"].(string) == "IN") ||
		!assert.True(t, mq["type"].(string) == "A") ||
		!assert.True(t, ma0["value"].(string) == answer) ||
		!assert.True(t, m["client"].(string) == client) {
		return false
	}
	return true
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
