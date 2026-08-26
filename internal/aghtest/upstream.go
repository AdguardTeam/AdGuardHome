package aghtest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/AdguardTeam/dnsproxy/dnsproxytest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
)

// NewExchangingUpstream returns a new test upstream that responds with the
// provided CNAME, A, and AAAA records.
func NewExchangingUpstream(
	tb testing.TB,
	cname map[string][]string,
	ipv4 map[string][]net.IP,
	ipv6 map[string][]net.IP,
) (ups *dnsproxytest.Upstream) {
	tb.Helper()

	ups = &dnsproxytest.Upstream{
		OnAddress: func() (addr string) { return "upstream.example" },
		OnClose:   func() (err error) { return nil },
		OnExchange: func(m *dns.Msg) (resp *dns.Msg, err error) {
			resp = new(dns.Msg).SetReply(m)

			if len(m.Question) == 0 {
				return nil, fmt.Errorf("question should not be empty")
			}

			q := m.Question[0]

			resp.Answer = append(resp.Answer, cnameAnswers(q.Name, cname)...)
			resp.Answer = append(resp.Answer, ipAnswers(q.Name, q.Qtype, ipv4, ipv6)...)

			if len(resp.Answer) == 0 {
				resp.SetRcode(m, dns.RcodeNameError)
			}

			return resp, nil
		},
	}

	return ups
}

// cnameAnswers returns CNAME records for name from the cname map.
func cnameAnswers(name string, cname map[string][]string) (answers []dns.RR) {
	for _, t := range cname[name] {
		answers = append(answers, &dns.CNAME{
			Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME},
			Target: t,
		})
	}

	return answers
}

// ipAnswers returns A or AAAA records for name from the corresponding IP map,
// depending on qtype.
func ipAnswers(name string, qtype uint16, ipv4, ipv6 map[string][]net.IP) (ans []dns.RR) {
	var ips []net.IP
	switch qtype {
	case dns.TypeA:
		ips = ipv4[name]
	case dns.TypeAAAA:
		ips = ipv6[name]
	default:
		return nil
	}

	hdr := dns.RR_Header{Name: name, Rrtype: qtype}
	for _, ip := range ips {
		switch qtype {
		case dns.TypeA:
			ans = append(ans, &dns.A{Hdr: hdr, A: ip})
		case dns.TypeAAAA:
			ans = append(ans, &dns.AAAA{Hdr: hdr, AAAA: ip})
		}
	}

	return ans
}

// MatchedResponse is a test helper that returns a response with answer if req
// has question type qt, and target targ.  Otherwise, it returns nil.
//
// req must not be nil and req.Question must have a length of 1.  Answer is
// interpreted in the following ways:
//
//   - For A and AAAA queries, answer must be an IP address of the corresponding
//     protocol version.
//
//   - For PTR queries, answer should be a domain name in the response.
//
// If the answer does not correspond to the question type, MatchedResponse panics.
// Panics are used instead of [testing.TB], because the helper is intended to
// use in [UpstreamMock.OnExchange] callbacks, which are usually called in a
// separate goroutine.
//
// TODO(a.garipov): Consider adding version with DNS class as well.
func MatchedResponse(req *dns.Msg, qt uint16, targ, answer string) (resp *dns.Msg) {
	if req == nil || len(req.Question) != 1 {
		panic(fmt.Errorf("bad req: %+v", req))
	}

	q := req.Question[0]
	targ = dns.Fqdn(targ)
	if q.Qclass != dns.ClassINET || q.Qtype != qt || q.Name != targ {
		return nil
	}

	respHdr := dns.RR_Header{
		Name:   targ,
		Rrtype: qt,
		Class:  dns.ClassINET,
		Ttl:    60,
	}

	resp = new(dns.Msg).SetReply(req)
	switch qt {
	case dns.TypeA:
		resp.Answer = mustAnsA(respHdr, answer)
	case dns.TypeAAAA:
		resp.Answer = mustAnsAAAA(respHdr, answer)
	case dns.TypePTR:
		resp.Answer = []dns.RR{&dns.PTR{
			Hdr: respHdr,
			Ptr: answer,
		}}
	default:
		panic(fmt.Errorf("aghtest: bad question type: %s", dns.Type(qt)))
	}

	return resp
}

// mustAnsA returns valid answer records if s is a valid IPv4 address.
// Otherwise, mustAnsA panics.
func mustAnsA(respHdr dns.RR_Header, s string) (ans []dns.RR) {
	ip, err := netip.ParseAddr(s)
	if err != nil || !ip.Is4() {
		panic(fmt.Errorf("aghtest: bad A answer: %+v", s))
	}

	return []dns.RR{&dns.A{
		Hdr: respHdr,
		A:   ip.AsSlice(),
	}}
}

// mustAnsAAAA returns valid answer records if s is a valid IPv6 address.
// Otherwise, mustAnsAAAA panics.
func mustAnsAAAA(respHdr dns.RR_Header, s string) (ans []dns.RR) {
	ip, err := netip.ParseAddr(s)
	if err != nil || !ip.Is6() {
		panic(fmt.Errorf("aghtest: bad AAAA answer: %+v", s))
	}

	return []dns.RR{&dns.AAAA{
		Hdr:  respHdr,
		AAAA: ip.AsSlice(),
	}}
}

// NewUpstream returns an *dnsproxytest.Upstream, fields OnAddress and
// OnClose of which are set to stubs that return "upstream.example" and nil
// respectively. The field OnExchange is a stub that panics on call.
func NewUpstream() (u *dnsproxytest.Upstream) {
	return &dnsproxytest.Upstream{
		OnAddress: func() (addr string) { return "upstream.example" },
		OnClose:   func() (err error) { return nil },
		OnExchange: func(req *dns.Msg) (resp *dns.Msg, err error) {
			panic(testutil.UnexpectedCall(req))
		},
	}
}

// NewBlockUpstream returns an *dnsproxytest.Upstream that works like an
// upstream that supports hash-based safe-browsing/adult-blocking feature.  If
// shouldBlock is true, hostname's actual hash is returned, blocking it.
// Otherwise, it returns a different hash.
func NewBlockUpstream(hostname string, shouldBlock bool) (u *dnsproxytest.Upstream) {
	hash := sha256.Sum256([]byte(hostname))
	hashStr := hex.EncodeToString(hash[:])
	if !shouldBlock {
		hashStr = hex.EncodeToString(hash[:])[:2] + strings.Repeat("ab", 28)
	}

	ans := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    60,
		},
		Txt: []string{hashStr},
	}
	respTmpl := &dns.Msg{
		Answer: []dns.RR{ans},
	}

	return &dnsproxytest.Upstream{
		OnAddress: func() (addr string) { return "sbpc.upstream.example" },
		OnExchange: func(req *dns.Msg) (resp *dns.Msg, err error) {
			resp = respTmpl.Copy()
			resp.SetReply(req)
			resp.Answer[0].(*dns.TXT).Hdr.Name = req.Question[0].Name

			return resp, nil
		},
		OnClose: func() (err error) { return nil },
	}
}

// ErrUpstream is the error returned from the *dnsproxytest.Upstream created
// by NewErrorUpstream.
const ErrUpstream errors.Error = "test upstream error"

// NewErrorUpstream returns an *dnsproxytest.Upstream that returns
// ErrUpstream from its Exchange method.
func NewErrorUpstream() (u *dnsproxytest.Upstream) {
	return &dnsproxytest.Upstream{
		OnAddress: func() (addr string) { return "error.upstream.example" },
		OnExchange: func(_ *dns.Msg) (resp *dns.Msg, err error) {
			return nil, ErrUpstream
		},
		OnClose: func() (err error) { return nil },
	}
}
