package filtering

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/mathutil"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

// Legacy DNS rewrites

// LegacyRewrite is a single legacy DNS rewrite record.
//
// Instances of *LegacyRewrite must never be nil.
type LegacyRewrite struct {
	// Domain is the domain pattern for which this rewrite should work.
	Domain string `yaml:"domain"`

	// Answer is the IP address, canonical name, or one of the special
	// values: "A" or "AAAA".
	Answer string `yaml:"answer"`

	// IP is the IP address that should be used in the response if Type is
	// dns.TypeA or dns.TypeAAAA.
	IP netip.Addr `yaml:"-"`

	// Type is the DNS record type: A, AAAA, or CNAME.
	Type uint16 `yaml:"-"`
}

// equal returns true if the rw is equal to the other.
func (rw *LegacyRewrite) equal(other *LegacyRewrite) (ok bool) {
	return rw.Domain == other.Domain && rw.Answer == other.Answer
}

// matchesQType returns true if the entry matches the question type qt.
func (rw *LegacyRewrite) matchesQType(qt uint16) (ok bool) {
	// Add CNAMEs, since they match for all types requests.
	if rw.Type == dns.TypeCNAME {
		return true
	}

	// Reject types other than A and AAAA.
	if qt != dns.TypeA && qt != dns.TypeAAAA {
		return false
	}

	// If the types match or the entry is set to allow only the other type,
	// include them.
	return rw.Type == qt || rw.IP == netip.Addr{}
}

// normalize makes sure that the new or decoded entry is normalized with regards
// to domain name case, IP length, and so on.
//
// If rw is nil, it returns an errors.
func (rw *LegacyRewrite) normalize() (err error) {
	if rw == nil {
		return errors.Error("nil rewrite entry")
	}

	// TODO(a.garipov): Write a case-agnostic version of strings.HasSuffix and
	// use it in matchDomainWildcard instead of using strings.ToLower
	// everywhere.
	rw.Domain = strings.ToLower(rw.Domain)

	switch rw.Answer {
	case "AAAA":
		rw.IP = netip.Addr{}
		rw.Type = dns.TypeAAAA

		return nil
	case "A":
		rw.IP = netip.Addr{}
		rw.Type = dns.TypeA

		return nil
	default:
		// Go on.
	}

	ip, err := netip.ParseAddr(rw.Answer)
	if err != nil {
		log.Debug("normalizing legacy rewrite: %s", err)
		rw.Type = dns.TypeCNAME

		return nil
	}

	rw.IP = ip
	if ip.Is4() {
		rw.Type = dns.TypeA
	} else {
		rw.Type = dns.TypeAAAA
	}

	return nil
}

// isWildcard returns true if pat is a wildcard domain pattern.
func isWildcard(pat string) bool {
	return len(pat) > 1 && pat[0] == '*' && pat[1] == '.'
}

// matchDomainWildcard returns true if host matches the wildcard pattern.
func matchDomainWildcard(host, wildcard string) (ok bool) {
	return isWildcard(wildcard) && strings.HasSuffix(host, wildcard[1:])
}

// Compare is used to sort rewrites according to the following priority:
//
//  1. A and AAAA > CNAME;
//  2. wildcard > exact;
//  3. lower level wildcard > higher level wildcard;
func (rw *LegacyRewrite) Compare(b *LegacyRewrite) (res int) {
	if rw.Type == dns.TypeCNAME {
		if b.Type != dns.TypeCNAME {
			return -1
		}
	} else if b.Type == dns.TypeCNAME {
		return 1
	}

	if aIsWld, bIsWld := isWildcard(rw.Domain), isWildcard(b.Domain); aIsWld == bIsWld {
		// Both are either wildcards or both aren't.
		return len(b.Domain) - len(rw.Domain)
	} else if aIsWld {
		return 1
	} else {
		return -1
	}
}

// prepareRewrites normalizes and validates all legacy DNS rewrites.
func (d *DNSFilter) prepareRewrites() (err error) {
	for i, r := range d.conf.Rewrites {
		err = r.normalize()
		if err != nil {
			return fmt.Errorf("at index %d: %w", i, err)
		}
	}

	return nil
}

// findRewrites returns the list of matched rewrite entries.  If rewrites are
// empty, but matched is true, the domain is found among the rewrite rules but
// not for this question type.
//
// The result priority is: CNAME, then A and AAAA; exact, then wildcard.  If the
// host is matched exactly, wildcard entries aren't returned.  If the host
// matched by wildcards, return the most specific for the question type.
func findRewrites(
	entries []*LegacyRewrite,
	host string,
	qtype uint16,
) (rewrites []*LegacyRewrite, matched bool) {
	for _, e := range entries {
		if e.Domain != host && !matchDomainWildcard(host, e.Domain) {
			continue
		}

		matched = true
		if e.matchesQType(qtype) {
			rewrites = append(rewrites, e)
		}
	}

	if len(rewrites) == 0 {
		return nil, matched
	}

	slices.SortFunc(rewrites, (*LegacyRewrite).Compare)

	for i, r := range rewrites {
		if isWildcard(r.Domain) {
			// Don't use rewrites[:0], because we need to return at least one
			// item here.
			rewrites = rewrites[:mathutil.Max(1, i)]

			break
		}
	}

	return rewrites, matched
}

// setRewriteResult sets the Reason or IPList of res if necessary.  res must not
// be nil.
func setRewriteResult(res *Result, host string, rewrites []*LegacyRewrite, qtype uint16) {
	for _, rw := range rewrites {
		if rw.Type == qtype && (qtype == dns.TypeA || qtype == dns.TypeAAAA) {
			if rw.IP == (netip.Addr{}) {
				// "A"/"AAAA" exception: allow getting from upstream.
				res.Reason = NotFilteredNotFound

				return
			}

			res.IPList = append(res.IPList, rw.IP)

			log.Debug("rewrite: a/aaaa for %s is %s", host, rw.IP)
		}
	}
}

// cloneRewrites returns a deep copy of entries.
func cloneRewrites(entries []*LegacyRewrite) (clone []*LegacyRewrite) {
	clone = make([]*LegacyRewrite, len(entries))
	for i, rw := range entries {
		clone[i] = &LegacyRewrite{
			Domain: rw.Domain,
			Answer: rw.Answer,
			IP:     rw.IP,
			Type:   rw.Type,
		}
	}

	return clone
}
