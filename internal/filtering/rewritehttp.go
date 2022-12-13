package filtering

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rewrite"
	"github.com/AdguardTeam/golibs/log"
)

// handleRewriteList is the handler for the GET /control/rewrite/list HTTP API.
func (d *DNSFilter) handleRewriteList(w http.ResponseWriter, r *http.Request) {
	_ = aghhttp.WriteJSONResponse(w, r, d.rewriteStorage.List())
}

// handleRewriteAdd is the handler for the POST /control/rewrite/add HTTP API.
func (d *DNSFilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {
	rw := &rewrite.Item{}
	err := json.NewDecoder(r.Body).Decode(rw)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	err = d.rewriteStorage.Add(rw)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "add rewrite: %s", err)

		return
	}

	log.Debug("rewrite: added element: %s -> %s", rw.Domain, rw.Answer)

	d.confLock.Lock()
	d.Config.Rewrites = d.rewriteStorage.List()
	d.confLock.Unlock()

	d.Config.ConfigModified()
}

// handleRewriteDelete is the handler for the POST /control/rewrite/delete HTTP
// API.
func (d *DNSFilter) handleRewriteDelete(w http.ResponseWriter, r *http.Request) {
	entDel := rewrite.Item{}
	err := json.NewDecoder(r.Body).Decode(&entDel)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	err = d.rewriteStorage.Remove(&entDel)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "remove rewrite: %s", err)

		return
	}

	d.confLock.Lock()
	d.Config.Rewrites = d.rewriteStorage.List()
	d.confLock.Unlock()

	d.Config.ConfigModified()
}
