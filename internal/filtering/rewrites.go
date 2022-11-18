// DNS Rewrites

package filtering

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

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
	IP net.IP `yaml:"-"`

	// Type is the DNS record type: A, AAAA, or CNAME.
	Type uint16 `yaml:"-"`
}

// clone returns a deep clone of rw.
func (rw *LegacyRewrite) clone() (cloneRW *LegacyRewrite) {
	return &LegacyRewrite{
		Domain: rw.Domain,
		Answer: rw.Answer,
		IP:     slices.Clone(rw.IP),
		Type:   rw.Type,
	}
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
	return rw.Type == qt || rw.IP == nil
}

// normalize makes sure that the a new or decoded entry is normalized with
// regards to domain name case, IP length, and so on.
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
		rw.IP = nil
		rw.Type = dns.TypeAAAA

		return nil
	case "A":
		rw.IP = nil
		rw.Type = dns.TypeA

		return nil
	default:
		// Go on.
	}

	ip := net.ParseIP(rw.Answer)
	if ip == nil {
		rw.Type = dns.TypeCNAME

		return nil
	}

	ip4 := ip.To4()
	if ip4 != nil {
		rw.IP = ip4
		rw.Type = dns.TypeA
	} else {
		rw.IP = ip
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

// rewritesSorted is a slice of legacy rewrites for sorting.
//
// The sorting priority:
//
//  1. A and AAAA > CNAME
//  2. wildcard > exact
//  3. lower level wildcard > higher level wildcard
//
// TODO(a.garipov):  Replace with slices.Sort.
type rewritesSorted []*LegacyRewrite

// Len implements the sort.Interface interface for rewritesSorted.
func (a rewritesSorted) Len() (l int) { return len(a) }

// Swap implements the sort.Interface interface for rewritesSorted.
func (a rewritesSorted) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements the sort.Interface interface for rewritesSorted.
func (a rewritesSorted) Less(i, j int) (less bool) {
	ith, jth := a[i], a[j]
	if ith.Type == dns.TypeCNAME && jth.Type != dns.TypeCNAME {
		return true
	} else if ith.Type != dns.TypeCNAME && jth.Type == dns.TypeCNAME {
		return false
	}

	if iw, jw := isWildcard(ith.Domain), isWildcard(jth.Domain); iw != jw {
		return jw
	}

	// Both are either wildcards or not.
	return len(ith.Domain) > len(jth.Domain)
}

// prepareRewrites normalizes and validates all legacy DNS rewrites.
func (d *DNSFilter) prepareRewrites() (err error) {
	for i, r := range d.Rewrites {
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

	sort.Sort(rewritesSorted(rewrites))

	for i, r := range rewrites {
		if isWildcard(r.Domain) {
			// Don't use rewrites[:0], because we need to return at least one
			// item here.
			rewrites = rewrites[:max(1, i)]

			break
		}
	}

	return rewrites, matched
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

type rewriteEntryJSON struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

func (d *DNSFilter) handleRewriteList(w http.ResponseWriter, r *http.Request) {
	arr := []*rewriteEntryJSON{}

	d.confLock.Lock()
	for _, ent := range d.Config.Rewrites {
		jsent := rewriteEntryJSON{
			Domain: ent.Domain,
			Answer: ent.Answer,
		}
		arr = append(arr, &jsent)
	}
	d.confLock.Unlock()

	_ = aghhttp.WriteJSONResponse(w, r, arr)
}

func (d *DNSFilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {
	rwJSON := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&rwJSON)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	rw := &LegacyRewrite{
		Domain: rwJSON.Domain,
		Answer: rwJSON.Answer,
	}

	err = rw.normalize()
	if err != nil {
		// Shouldn't happen currently, since normalize only returns a non-nil
		// error when a rewrite is nil, but be change-proof.
		aghhttp.Error(r, w, http.StatusBadRequest, "normalizing: %s", err)

		return
	}

	d.confLock.Lock()
	d.Config.Rewrites = append(d.Config.Rewrites, rw)
	d.confLock.Unlock()
	log.Debug("rewrite: added element: %s -> %s [%d]", rw.Domain, rw.Answer, len(d.Config.Rewrites))

	d.Config.ConfigModified()
}

func (d *DNSFilter) handleRewriteDelete(w http.ResponseWriter, r *http.Request) {
	jsent := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&jsent)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	entDel := &LegacyRewrite{
		Domain: jsent.Domain,
		Answer: jsent.Answer,
	}
	arr := []*LegacyRewrite{}

	d.confLock.Lock()
	for _, ent := range d.Config.Rewrites {
		if ent.equal(entDel) {
			log.Debug("rewrite: removed element: %s -> %s", ent.Domain, ent.Answer)

			continue
		}

		arr = append(arr, ent)
	}
	d.Config.Rewrites = arr
	d.confLock.Unlock()

	d.Config.ConfigModified()
}
