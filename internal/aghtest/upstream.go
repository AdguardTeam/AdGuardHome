package aghtest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
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
	// Reverse is a map of address to domain name.
	Reverse map[string][]string
	// Addr is the address for Address method.
	Addr string
}

// RespondTo returns a response with answer if req has class cl, question type
// qt, and target targ.
func RespondTo(t testing.TB, req *dns.Msg, cl, qt uint16, targ, answer string) (resp *dns.Msg) {
	t.Helper()

	require.NotNil(t, req)
	require.Len(t, req.Question, 1)

	q := req.Question[0]
	targ = dns.Fqdn(targ)
	if q.Qclass != cl || q.Qtype != qt || q.Name != targ {
		return nil
	}

	respHdr := dns.RR_Header{
		Name:   targ,
		Rrtype: qt,
		Class:  cl,
		Ttl:    60,
	}

	resp = new(dns.Msg).SetReply(req)
	switch qt {
	case dns.TypePTR:
		resp.Answer = []dns.RR{
			&dns.PTR{
				Hdr: respHdr,
				Ptr: answer,
			},
		}
	default:
		t.Fatalf("unsupported question type: %s", dns.Type(qt))
	}

	return resp
}

// Exchange implements the upstream.Upstream interface for *Upstream.
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
	case dns.TypePTR:
		for _, name := range u.Reverse[name] {
			resp.Answer = append(resp.Answer, &dns.PTR{Hdr: hdr, Ptr: name})
		}
	}
	if len(resp.Answer) == 0 {
		resp.SetRcode(m, dns.RcodeNameError)
	}

	return resp, nil
}

// Address implements upstream.Upstream interface for *Upstream.
func (u *Upstream) Address() string {
	return u.Addr
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
		OnAddress: func() (addr string) {
			return "sbpc.upstream.example"
		},
		OnExchange: func(req *dns.Msg) (resp *dns.Msg, err error) {
			resp = respTmpl.Copy()
			resp.SetReply(req)
			resp.Answer[0].(*dns.TXT).Hdr.Name = req.Question[0].Name

			return resp, nil
		},
	}
}

// ErrUpstream is the error returned from the [*UpstreamMock] created by
// [NewErrorUpstream].
const ErrUpstream errors.Error = "test upstream error"

// NewErrorUpstream returns an [*UpstreamMock] that returns [ErrUpstream] from
// its Exchange method.
func NewErrorUpstream() (u *UpstreamMock) {
	return &UpstreamMock{
		OnAddress: func() (addr string) {
			return "error.upstream.example"
		},
		OnExchange: func(_ *dns.Msg) (resp *dns.Msg, err error) {
			return nil, errors.Error("test upstream error")
		},
	}
}
