package home

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
)

type rewriteEntryJSON struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

func handleRewriteList(w http.ResponseWriter, r *http.Request) {

	arr := []*rewriteEntryJSON{}

	config.RLock()
	for _, ent := range config.DNS.Rewrites {
		jsent := rewriteEntryJSON{
			Domain: ent.Domain,
			Answer: ent.Answer,
		}
		arr = append(arr, &jsent)
	}
	config.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(arr)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json.Encode: %s", err)
		return
	}
}

func handleRewriteAdd(w http.ResponseWriter, r *http.Request) {

	jsent := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&jsent)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	ent := dnsfilter.RewriteEntry{
		Domain: jsent.Domain,
		Answer: jsent.Answer,
	}
	config.Lock()
	config.DNS.Rewrites = append(config.DNS.Rewrites, ent)
	config.Unlock()
	log.Debug("Rewrites: added element: %s -> %s [%d]",
		ent.Domain, ent.Answer, len(config.DNS.Rewrites))

	err = writeAllConfigsAndReloadDNS()
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	returnOK(w)
}

func handleRewriteDelete(w http.ResponseWriter, r *http.Request) {

	jsent := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&jsent)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	entDel := dnsfilter.RewriteEntry{
		Domain: jsent.Domain,
		Answer: jsent.Answer,
	}
	arr := []dnsfilter.RewriteEntry{}
	config.Lock()
	for _, ent := range config.DNS.Rewrites {
		if ent == entDel {
			log.Debug("Rewrites: removed element: %s -> %s", ent.Domain, ent.Answer)
			continue
		}
		arr = append(arr, ent)
	}
	config.DNS.Rewrites = arr
	config.Unlock()

	err = writeAllConfigsAndReloadDNS()
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	returnOK(w)
}

func registerRewritesHandlers() {
	httpRegister(http.MethodGet, "/control/rewrite/list", handleRewriteList)
	httpRegister(http.MethodPost, "/control/rewrite/add", handleRewriteAdd)
	httpRegister(http.MethodPost, "/control/rewrite/delete", handleRewriteDelete)
}
