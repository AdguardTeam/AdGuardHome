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
	// Addr is the address for Address method.
	Addr string
	// CName is a map of hostname to canonical name.
	CName map[string]string
	// IPv4 is a map of hostname to IPv4.
	IPv4 map[string][]net.IP
	// IPv6 is a map of hostname to IPv6.
	IPv6 map[string][]net.IP
	// Reverse is a map of address to domain name.
	Reverse map[string][]string
}

// Exchange implements upstream.Upstream interface for *TestUpstream.
func (u *TestUpstream) Exchange(m *dns.Msg) (resp *dns.Msg, err error) {
	resp = &dns.Msg{}
	resp.SetReply(m)

	if len(m.Question) == 0 {
		return nil, fmt.Errorf("question should not be empty")
	}
	name := m.Question[0].Name

	if cname, ok := u.CName[name]; ok {
		resp.Answer = append(resp.Answer, &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeCNAME,
			},
			Target: cname,
		})
	}

	var hasRec bool
	var rrType uint16
	var ips []net.IP
	switch m.Question[0].Qtype {
	case dns.TypeA:
		rrType = dns.TypeA
		if ipv4addr, ok := u.IPv4[name]; ok {
			hasRec = true
			ips = ipv4addr
		}
	case dns.TypeAAAA:
		rrType = dns.TypeAAAA
		if ipv6addr, ok := u.IPv6[name]; ok {
			hasRec = true
			ips = ipv6addr
		}
	case dns.TypePTR:
		names, ok := u.Reverse[name]
		if !ok {
			break
		}

		for _, n := range names {
			resp.Answer = append(resp.Answer, &dns.PTR{
				Hdr: dns.RR_Header{
					Name:   n,
					Rrtype: rrType,
				},
				Ptr: n,
			})
		}
	}

	for _, ip := range ips {
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: rrType,
			},
			A: ip,
		})
	}

	if len(resp.Answer) == 0 {
		if hasRec {
			// Set no error RCode if there are some records for
			// given Qname but we didn't apply them.
			resp.SetRcode(m, dns.RcodeSuccess)

			return resp, nil
		}
		// Set NXDomain RCode otherwise.
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
	Hostname      string
	Block         bool
	requestsCount int
	lock          sync.RWMutex
}

// Exchange returns a message unique for TestBlockUpstream's Hostname-Block
// pair.
func (u *TestBlockUpstream) Exchange(r *dns.Msg) (*dns.Msg, error) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.requestsCount++

	hash := sha256.Sum256([]byte(u.Hostname))
	hashToReturn := hex.EncodeToString(hash[:])
	if !u.Block {
		hashToReturn = hex.EncodeToString(hash[:])[:2] + strings.Repeat("ab", 28)
	}

	m := &dns.Msg{}
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

	return u.requestsCount
}

// TestErrUpstream implements upstream.Upstream interface for replacing real
// upstream in tests.
type TestErrUpstream struct {
	// The error returned by Exchange may be unwraped to the Err.
	Err error
}

// Exchange always returns nil Msg and non-nil error.
func (u *TestErrUpstream) Exchange(*dns.Msg) (*dns.Msg, error) {
	// We don't use an agherr.Error to avoid the import cycle since aghtests
	// used to provide the utilities for testing which agherr (and any other
	// testable package) should be able to use.
	return nil, fmt.Errorf("errupstream: %w", u.Err)
}

// Address always returns an empty string.
func (u *TestErrUpstream) Address() string {
	return ""
}
