package rewrite

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/miekg/dns"
)

// Item is a single DNS rewrite record.
type Item struct {
	// Domain is the domain pattern for which this rewrite should work.
	Domain string `yaml:"domain"`

	// Answer is the IP address, canonical name, or one of the special
	// values: "A" or "AAAA".
	Answer string `yaml:"answer"`
}

// equal returns true if rw is equal to other.
func (rw *Item) equal(other *Item) (ok bool) {
	if rw == nil {
		return other == nil
	} else if other == nil {
		return false
	}

	return *rw == *other
}

// toRule converts rw to a filter rule.
func (rw *Item) toRule() (res string) {
	if rw == nil {
		return ""
	}

	domain := strings.ToLower(rw.Domain)

	dType, exception := rw.rewriteParams()
	dTypeKey := dns.TypeToString[dType]
	if exception {
		return fmt.Sprintf("@@||%s^$dnstype=%s,dnsrewrite", domain, dTypeKey)
	}

	return fmt.Sprintf("|%s^$dnsrewrite=NOERROR;%s;%s", domain, dTypeKey, rw.Answer)
}

// rewriteParams returns dns request type and exception flag for rw.
func (rw *Item) rewriteParams() (dType uint16, exception bool) {
	switch rw.Answer {
	case "AAAA":
		return dns.TypeAAAA, true
	case "A":
		return dns.TypeA, true
	default:
		// Go on.
	}

	addr, err := netip.ParseAddr(rw.Answer)
	if err != nil {
		// TODO(d.kolyshev): Validate rw.Answer as a domain name.
		return dns.TypeCNAME, false
	}

	if addr.Is4() {
		dType = dns.TypeA
	} else {
		dType = dns.TypeAAAA
	}

	return dType, false
}
