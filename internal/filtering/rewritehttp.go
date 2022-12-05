package filtering

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rewrite"
	"github.com/AdguardTeam/golibs/log"
)

func (d *DNSFilter) handleRewriteList(w http.ResponseWriter, r *http.Request) {
	d.confLock.RLock()
	defer d.confLock.RUnlock()

	_ = aghhttp.WriteJSONResponse(w, r, d.rewriteStorage.List())
}

func (d *DNSFilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {
	rw := rewrite.Item{}
	err := json.NewDecoder(r.Body).Decode(&rw)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	d.confLock.Lock()
	defer d.confLock.Unlock()

	err = d.rewriteStorage.Add(&rw)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "add rewrite: %s", err)

		return
	}

	log.Debug("rewrite: added element: %s -> %s", rw.Domain, rw.Answer)

	d.Config.ConfigModified()
}

func (d *DNSFilter) handleRewriteDelete(w http.ResponseWriter, r *http.Request) {
	entDel := rewrite.Item{}
	err := json.NewDecoder(r.Body).Decode(&entDel)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	d.confLock.Lock()
	defer d.confLock.Unlock()

	err = d.rewriteStorage.Remove(&entDel)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "remove rewrite: %s", err)

		return
	}

	d.Config.ConfigModified()
}
