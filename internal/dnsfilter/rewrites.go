// DNS Rewrites

package dnsfilter

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
	Domain string `yaml:"domain"`
	Answer string `yaml:"answer"` // IP address or canonical name
	Type   uint16 `yaml:"-"`      // DNS record type: CNAME, A or AAAA
	IP     net.IP `yaml:"-"`      // Parsed IP address (if Type is A or AAAA)
}

func (r *RewriteEntry) equals(b RewriteEntry) bool {
	return r.Domain == b.Domain && r.Answer == b.Answer
}

func isWildcard(host string) bool {
	return len(host) >= 2 &&
		host[0] == '*' && host[1] == '.'
}

// Return TRUE of host name matches a wildcard pattern
func matchDomainWildcard(host, wildcard string) bool {
	return isWildcard(wildcard) &&
		strings.HasSuffix(host, wildcard[1:])
}

type rewritesArray []RewriteEntry

func (a rewritesArray) Len() int { return len(a) }

func (a rewritesArray) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Priority:
//  . CNAME < A/AAAA;
//  . exact < wildcard;
//  . higher level wildcard < lower level wildcard
func (a rewritesArray) Less(i, j int) bool {
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

// Prepare entry for use
func (r *RewriteEntry) prepare() {
	if r.Answer == "AAAA" {
		r.IP = nil
		r.Type = dns.TypeAAAA
		return
	} else if r.Answer == "A" {
		r.IP = nil
		r.Type = dns.TypeA
		return
	}

	ip := net.ParseIP(r.Answer)
	if ip == nil {
		r.Type = dns.TypeCNAME
		return
	}

	r.IP = ip
	r.Type = dns.TypeAAAA

	ip4 := ip.To4()
	if ip4 != nil {
		r.IP = ip4
		r.Type = dns.TypeA
	}
}

func (d *Dnsfilter) prepareRewrites() {
	for i := range d.Rewrites {
		d.Rewrites[i].prepare()
	}
}

// Get the list of matched rewrite entries.
// Priority: CNAME, A/AAAA;  exact, wildcard.
// If matched exactly, don't return wildcard entries.
// If matched by several wildcards, select the more specific one
func findRewrites(a []RewriteEntry, host string) []RewriteEntry {
	rr := rewritesArray{}
	for _, r := range a {
		if r.Domain != host {
			if !matchDomainWildcard(host, r.Domain) {
				continue
			}
		}
		rr = append(rr, r)
	}

	if len(rr) == 0 {
		return nil
	}

	sort.Sort(rr)

	isWC := isWildcard(rr[0].Domain)
	if !isWC {
		for i, r := range rr {
			if isWildcard(r.Domain) {
				rr = rr[:i]
				break
			}
		}
	} else {
		rr = rr[:1]
	}

	return rr
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

func (d *Dnsfilter) handleRewriteList(w http.ResponseWriter, r *http.Request) {

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

func (d *Dnsfilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {

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

func (d *Dnsfilter) handleRewriteDelete(w http.ResponseWriter, r *http.Request) {

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

func (d *Dnsfilter) registerRewritesHandlers() {
	d.Config.HTTPRegister("GET", "/control/rewrite/list", d.handleRewriteList)
	d.Config.HTTPRegister("POST", "/control/rewrite/add", d.handleRewriteAdd)
	d.Config.HTTPRegister("POST", "/control/rewrite/delete", d.handleRewriteDelete)
}
