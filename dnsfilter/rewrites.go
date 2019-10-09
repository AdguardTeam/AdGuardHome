// DNS Rewrites

package dnsfilter

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/golibs/log"
)

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
		if ent == entDel {
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
