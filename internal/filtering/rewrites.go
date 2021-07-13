// DNS Rewrites

package filtering

import (
	"encoding/json"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// RewriteEntry is a rewrite array element
type RewriteEntry struct {
	// Domain is the domain for which this rewrite should work.
	Domain string `yaml:"domain"`
	// Answer is the IP address, canonical name, or one of the special
	// values: "A" or "AAAA".
	Answer string `yaml:"answer"`
	// IP is the IP address that should be used in the response if Type is
	// A or AAAA.
	IP net.IP `yaml:"-"`
	// Type is the DNS record type: A, AAAA, or CNAME.
	Type uint16 `yaml:"-"`
}

func (r *RewriteEntry) equals(b RewriteEntry) bool {
	return r.Domain == b.Domain && r.Answer == b.Answer
}

func isWildcard(host string) bool {
	return len(host) > 1 && host[0] == '*' && host[1] == '.'
}

// matchDomainWildcard returns true if host matches the wildcard pattern.
func matchDomainWildcard(host, wildcard string) (ok bool) {
	return isWildcard(wildcard) && strings.HasSuffix(host, wildcard[1:])
}

// rewritesSorted is a slice of legacy rewrites for sorting.
//
// The sorting priority:
//
//   A and AAAA > CNAME
//   wildcard > exact
//   lower level wildcard > higher level wildcard
//
type rewritesSorted []RewriteEntry

// Len implements the sort.Interface interface for legacyRewritesSorted.
func (a rewritesSorted) Len() int { return len(a) }

// Swap implements the sort.Interface interface for legacyRewritesSorted.
func (a rewritesSorted) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements the sort.Interface interface for legacyRewritesSorted.
func (a rewritesSorted) Less(i, j int) bool {
	if a[i].Type == dns.TypeCNAME && a[j].Type != dns.TypeCNAME {
		return true
	} else if a[i].Type != dns.TypeCNAME && a[j].Type == dns.TypeCNAME {
		return false
	}

	if isWildcard(a[i].Domain) {
		if !isWildcard(a[j].Domain) {
			return false
		}
	} else {
		if isWildcard(a[j].Domain) {
			return true
		}
	}

	// both are wildcards
	return len(a[i].Domain) > len(a[j].Domain)
}

// prepare prepares the a new or decoded entry.
func (r *RewriteEntry) prepare() {
	// TODO(a.garipov): Write a case-agnostic version of strings.HasSuffix
	// and use it in matchDomainWildcard instead of using strings.ToLower
	// everywhere.
	r.Domain = strings.ToLower(r.Domain)

	switch r.Answer {
	case "AAAA":
		r.IP = nil
		r.Type = dns.TypeAAAA

		return
	case "A":
		r.IP = nil
		r.Type = dns.TypeA

		return
	default:
		// Go on.
	}

	ip := net.ParseIP(r.Answer)
	if ip == nil {
		r.Type = dns.TypeCNAME

		return
	}

	ip4 := ip.To4()
	if ip4 != nil {
		r.IP = ip4
		r.Type = dns.TypeA
	} else {
		r.IP = ip
		r.Type = dns.TypeAAAA
	}
}

func (d *DNSFilter) prepareRewrites() {
	for i := range d.Rewrites {
		d.Rewrites[i].prepare()
	}
}

// findRewrites returns the list of matched rewrite entries.  The priority is:
// CNAME, then A and AAAA; exact, then wildcard.  If the host is matched
// exactly, wildcard entries aren't returned.  If the host matched by wildcards,
// return the most specific for the question type.
func findRewrites(a []RewriteEntry, host string, qtype uint16) []RewriteEntry {
	rr := rewritesSorted{}
	for _, r := range a {
		if r.Domain != host && !matchDomainWildcard(host, r.Domain) {
			continue
		}

		// Return CNAMEs for all types requests, but only the
		// appropriate ones for A and AAAA.
		if r.Type == dns.TypeCNAME ||
			(r.Type == qtype && (qtype == dns.TypeA || qtype == dns.TypeAAAA)) {
			rr = append(rr, r)
		}
	}

	if len(rr) == 0 {
		return nil
	}

	sort.Sort(rr)

	for i, r := range rr {
		if isWildcard(r.Domain) {
			// Don't use rr[:0], because we need to return at least
			// one item here.
			rr = rr[:max(1, i)]

			break
		}
	}

	return rr
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func rewriteArrayDup(a []RewriteEntry) []RewriteEntry {
	a2 := make([]RewriteEntry, len(a))
	copy(a2, a)
	return a2
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

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(arr)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Encode: %s", err)
		return
	}
}

func (d *DNSFilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {
	jsent := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&jsent)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	ent := RewriteEntry{
		Domain: jsent.Domain,
		Answer: jsent.Answer,
	}
	ent.prepare()
	d.confLock.Lock()
	d.Config.Rewrites = append(d.Config.Rewrites, ent)
	d.confLock.Unlock()
	log.Debug("Rewrites: added element: %s -> %s [%d]",
		ent.Domain, ent.Answer, len(d.Config.Rewrites))

	d.Config.ConfigModified()
}

func (d *DNSFilter) handleRewriteDelete(w http.ResponseWriter, r *http.Request) {
	jsent := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&jsent)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	entDel := RewriteEntry{
		Domain: jsent.Domain,
		Answer: jsent.Answer,
	}
	arr := []RewriteEntry{}
	d.confLock.Lock()
	for _, ent := range d.Config.Rewrites {
		if ent.equals(entDel) {
			log.Debug("Rewrites: removed element: %s -> %s", ent.Domain, ent.Answer)
			continue
		}
		arr = append(arr, ent)
	}
	d.Config.Rewrites = arr
	d.confLock.Unlock()

	d.Config.ConfigModified()
}

func (d *DNSFilter) registerRewritesHandlers() {
	d.Config.HTTPRegister(http.MethodGet, "/control/rewrite/list", d.handleRewriteList)
	d.Config.HTTPRegister(http.MethodPost, "/control/rewrite/add", d.handleRewriteAdd)
	d.Config.HTTPRegister(http.MethodPost, "/control/rewrite/delete", d.handleRewriteDelete)
}
