package aghtest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// TestUpstream is a mock of real upstream.
type TestUpstream struct {
	// CName is a map of hostname to canonical name.
	CName map[string]string
	// IPv4 is a map of hostname to IPv4.
	IPv4 map[string][]net.IP
	// IPv6 is a map of hostname to IPv6.
	IPv6 map[string][]net.IP
	// Reverse is a map of address to domain name.
	Reverse map[string][]string
	// Addr is the address for Address method.
	Addr string
}

// Exchange implements upstream.Upstream interface for *TestUpstream.
//
// TODO(a.garipov): Split further into handlers.
func (u *TestUpstream) Exchange(m *dns.Msg) (resp *dns.Msg, err error) {
	resp = &dns.Msg{}
	resp.SetReply(m)

	if len(m.Question) == 0 {
		return nil, fmt.Errorf("question should not be empty")
	}

	name := m.Question[0].Name

	if cname, ok := u.CName[name]; ok {
		ans := &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeCNAME,
			},
			Target: cname,
		}

		resp.Answer = append(resp.Answer, ans)
	}

	rrType := m.Question[0].Qtype
	hdr := dns.RR_Header{
		Name:   name,
		Rrtype: rrType,
	}

	var names []string
	var ips []net.IP
	switch m.Question[0].Qtype {
	case dns.TypeA:
		ips = u.IPv4[name]
	case dns.TypeAAAA:
		ips = u.IPv6[name]
	case dns.TypePTR:
		names = u.Reverse[name]
	}

	for _, ip := range ips {
		var ans dns.RR
		if rrType == dns.TypeA {
			ans = &dns.A{
				Hdr: hdr,
				A:   ip,
			}

			resp.Answer = append(resp.Answer, ans)

			continue
		}

		ans = &dns.AAAA{
			Hdr:  hdr,
			AAAA: ip,
		}

		resp.Answer = append(resp.Answer, ans)
	}

	for _, n := range names {
		ans := &dns.PTR{
			Hdr: hdr,
			Ptr: n,
		}

		resp.Answer = append(resp.Answer, ans)
	}

	if len(resp.Answer) == 0 {
		resp.SetRcode(m, dns.RcodeNameError)
	}

	return resp, nil
}

// Address implements upstream.Upstream interface for *TestUpstream.
func (u *TestUpstream) Address() string {
	return u.Addr
}

// TestBlockUpstream implements upstream.Upstream interface for replacing real
// upstream in tests.
type TestBlockUpstream struct {
	Hostname string

	// lock protects reqNum.
	lock   sync.RWMutex
	reqNum int

	Block bool
}

// Exchange returns a message unique for TestBlockUpstream's Hostname-Block
// pair.
func (u *TestBlockUpstream) Exchange(r *dns.Msg) (*dns.Msg, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.reqNum++

	hash := sha256.Sum256([]byte(u.Hostname))
	hashToReturn := hex.EncodeToString(hash[:])
	if !u.Block {
		hashToReturn = hex.EncodeToString(hash[:])[:2] + strings.Repeat("ab", 28)
	}

	m := &dns.Msg{}
	m.SetReply(r)
	m.Answer = []dns.RR{
		&dns.TXT{
			Hdr: dns.RR_Header{
				Name: r.Question[0].Name,
			},
			Txt: []string{
				hashToReturn,
			},
		},
	}

	return m, nil
}

// Address always returns an empty string.
func (u *TestBlockUpstream) Address() string {
	return ""
}

// RequestsCount returns the number of handled requests. It's safe for
// concurrent use.
func (u *TestBlockUpstream) RequestsCount() int {
	u.lock.Lock()
	defer u.lock.Unlock()

	return u.reqNum
}

// TestErrUpstream implements upstream.Upstream interface for replacing real
// upstream in tests.
type TestErrUpstream struct {
	// The error returned by Exchange may be unwraped to the Err.
	Err error
}

// Exchange always returns nil Msg and non-nil error.
func (u *TestErrUpstream) Exchange(*dns.Msg) (*dns.Msg, error) {
	return nil, fmt.Errorf("errupstream: %w", u.Err)
}

// Address always returns an empty string.
func (u *TestErrUpstream) Address() string {
	return ""
}
