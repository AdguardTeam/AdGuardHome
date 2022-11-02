package aghtest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/miekg/dns"
)

// Additional Upstream Testing Utilities

// Upstream is a mock implementation of upstream.Upstream.
//
// TODO(a.garipov): Replace with UpstreamMock and rename it to just Upstream.
type Upstream struct {
	// CName is a map of hostname to canonical name.
	CName map[string][]string
	// IPv4 is a map of hostname to IPv4.
	IPv4 map[string][]net.IP
	// IPv6 is a map of hostname to IPv6.
	IPv6 map[string][]net.IP
}

var _ upstream.Upstream = (*Upstream)(nil)

// Exchange implements the [upstream.Upstream] interface for *Upstream.
//
// TODO(a.garipov): Split further into handlers.
func (u *Upstream) Exchange(m *dns.Msg) (resp *dns.Msg, err error) {
	resp = new(dns.Msg).SetReply(m)

	if len(m.Question) == 0 {
		return nil, fmt.Errorf("question should not be empty")
	}

	q := m.Question[0]
	name := q.Name
	for _, cname := range u.CName[name] {
		resp.Answer = append(resp.Answer, &dns.CNAME{
			Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME},
			Target: cname,
		})
	}

	qtype := q.Qtype
	hdr := dns.RR_Header{
		Name:   name,
		Rrtype: qtype,
	}

	switch qtype {
	case dns.TypeA:
		for _, ip := range u.IPv4[name] {
			resp.Answer = append(resp.Answer, &dns.A{Hdr: hdr, A: ip})
		}
	case dns.TypeAAAA:
		for _, ip := range u.IPv6[name] {
			resp.Answer = append(resp.Answer, &dns.AAAA{Hdr: hdr, AAAA: ip})
		}
	}
	if len(resp.Answer) == 0 {
		resp.SetRcode(m, dns.RcodeNameError)
	}

	return resp, nil
}

// Address implements [upstream.Upstream] interface for *Upstream.
func (u *Upstream) Address() string {
	return "todo.upstream.example"
}

// Close implements [upstream.Upstream] interface for *Upstream.
func (u *Upstream) Close() (err error) {
	return nil
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

// NewUpstreamMock returns an [*UpstreamMock], fields OnAddress and OnClose of
// which are set to stubs that return "upstream.example" and nil respectively.
// The field OnExchange is set to onExc.
func NewUpstreamMock(onExc func(req *dns.Msg) (resp *dns.Msg, err error)) (u *UpstreamMock) {
	return &UpstreamMock{
		OnAddress:  func() (addr string) { return "upstream.example" },
		OnExchange: onExc,
		OnClose:    func() (err error) { return nil },
	}
}

// NewBlockUpstream returns an [*UpstreamMock] that works like an upstream that
// supports hash-based safe-browsing/adult-blocking feature.  If shouldBlock is
// true, hostname's actual hash is returned, blocking it.  Otherwise, it returns
// a different hash.
func NewBlockUpstream(hostname string, shouldBlock bool) (u *UpstreamMock) {
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

	return &UpstreamMock{
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

// ErrUpstream is the error returned from the [*UpstreamMock] created by
// [NewErrorUpstream].
const ErrUpstream errors.Error = "test upstream error"

// NewErrorUpstream returns an [*UpstreamMock] that returns [ErrUpstream] from
// its Exchange method.
func NewErrorUpstream() (u *UpstreamMock) {
	return &UpstreamMock{
		OnAddress: func() (addr string) { return "error.upstream.example" },
		OnExchange: func(_ *dns.Msg) (resp *dns.Msg, err error) {
			return nil, errors.Error("test upstream error")
		},
		OnClose: func() (err error) { return nil },
	}
}
