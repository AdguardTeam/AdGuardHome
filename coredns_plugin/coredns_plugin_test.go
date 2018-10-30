package dnsfilter

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
)

func TestSetup(t *testing.T) {
	for i, testcase := range []struct {
		config  string
		failing bool
	}{
		{`dnsfilter`, false},
		{`dnsfilter /dev/nonexistent/abcdef`, true},
		{`dnsfilter ../tests/dns.txt`, false},
		{`dnsfilter ../tests/dns.txt { safebrowsing }`, false},
		{`dnsfilter ../tests/dns.txt { parental }`, true},
	} {
		c := caddy.NewTestController("dns", testcase.config)
		err := setup(c)
		if err != nil {
			if !testcase.failing {
				t.Fatalf("Test #%d expected no errors, but got: %v", i, err)
			}
			continue
		}
		if testcase.failing {
			t.Fatalf("Test #%d expected to fail but it didn't", i)
		}
	}
}

func TestEtcHostsFilter(t *testing.T) {
	text := []byte("127.0.0.1 doubleclick.net\n" + "127.0.0.1 example.org example.net www.example.org www.example.net")
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tmpfile.Write(text); err != nil {
		t.Fatal(err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	c := caddy.NewTestController("dns", fmt.Sprintf("dnsfilter %s", tmpfile.Name()))
	p, err := setupPlugin(c)
	if err != nil {
		t.Fatal(err)
	}

	p.Next = zeroTTLBackend()

	ctx := context.TODO()

	for _, testcase := range []struct {
		host     string
		filtered bool
	}{
		{"www.doubleclick.net", false},
		{"doubleclick.net", true},
		{"www2.example.org", false},
		{"www2.example.net", false},
		{"test.www.example.org", false},
		{"test.www.example.net", false},
		{"example.org", true},
		{"example.net", true},
		{"www.example.org", true},
		{"www.example.net", true},
	} {
		req := new(dns.Msg)
		req.SetQuestion(testcase.host+".", dns.TypeA)

		resp := test.ResponseWriter{}
		rrw := dnstest.NewRecorder(&resp)
		rcode, err := p.ServeDNS(ctx, rrw, req)
		if err != nil {
			t.Fatalf("ServeDNS returned error: %s", err)
		}
		if rcode != rrw.Rcode {
			t.Fatalf("ServeDNS return value for host %s has rcode %d that does not match captured rcode %d", testcase.host, rcode, rrw.Rcode)
		}
		A, ok := rrw.Msg.Answer[0].(*dns.A)
		if !ok {
			t.Fatalf("Host %s expected to have result A", testcase.host)
		}
		ip := net.IPv4(127, 0, 0, 1)
		filtered := ip.Equal(A.A)
		if testcase.filtered && testcase.filtered != filtered {
			t.Fatalf("Host %s expected to be filtered, instead it is not filtered", testcase.host)
		}
		if !testcase.filtered && testcase.filtered != filtered {
			t.Fatalf("Host %s expected to be not filtered, instead it is filtered", testcase.host)
		}
	}
}

func zeroTTLBackend() plugin.Handler {
	return plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Response, m.RecursionAvailable = true, true

		m.Answer = []dns.RR{test.A("example.org. 0 IN A 127.0.0.53")}
		w.WriteMsg(m)
		return dns.RcodeSuccess, nil
	})
}
