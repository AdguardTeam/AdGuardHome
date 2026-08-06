package querylog

import (
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/miekg/dns"
)

var (
	// testLogger is a common logger for tests.
	testLogger = slogutil.NewDiscardLogger()

	// testAnswerIPv4 is a test DNS answer IPv4 value.
	testAnswerIPv4 = net.IPv4(192, 0, 2, 0)

	// testClientIPv4 is a test client IPv4 value.
	testClientIPv4 = net.IPv4(192, 0, 2, 1)
)

// addTestEntry is a helper that adds an entry to l.
func addTestEntry(l *queryLog, host string, answerStr, client net.IP, reason filtering.Reason) {
	q := dns.Msg{
		Question: []dns.Question{{
			Name:   host + ".",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	a := dns.Msg{
		Question: q.Question,
		Answer: []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
			},
			A: answerStr,
		}},
	}

	res := filtering.Result{
		ServiceName: "SomeService",
		Rules: []*filtering.ResultRule{{
			FilterListID: 1,
			Text:         "SomeRule",
		}},
		Reason:     reason,
		IsFiltered: true,
	}

	params := &AddParams{
		Question:   &q,
		Answer:     &a,
		OrigAnswer: &a,
		Result:     &res,
		Upstream:   "upstream",
		ClientIP:   client,
	}

	l.Add(params)
}
