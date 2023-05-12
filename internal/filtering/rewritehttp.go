package filtering

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/exp/slices"
)

// TODO(d.kolyshev): Use [rewrite.Item] instead.
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

// rewriteUpdateJSON is a struct for JSON object with rewrite rule update info.
type rewriteUpdateJSON struct {
	Target rewriteEntryJSON `json:"target"`
	Update rewriteEntryJSON `json:"update"`
}

// handleRewriteUpdate is the handler for the PUT /control/rewrite/update HTTP
// API.
func (d *DNSFilter) handleRewriteUpdate(w http.ResponseWriter, r *http.Request) {
	updateJSON := rewriteUpdateJSON{}
	err := json.NewDecoder(r.Body).Decode(&updateJSON)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	rwDel := &LegacyRewrite{
		Domain: updateJSON.Target.Domain,
		Answer: updateJSON.Target.Answer,
	}

	rwAdd := &LegacyRewrite{
		Domain: updateJSON.Update.Domain,
		Answer: updateJSON.Update.Answer,
	}

	err = rwAdd.normalize()
	if err != nil {
		// Shouldn't happen currently, since normalize only returns a non-nil
		// error when a rewrite is nil, but be change-proof.
		aghhttp.Error(r, w, http.StatusBadRequest, "normalizing: %s", err)

		return
	}

	index := -1
	defer func() {
		if index >= 0 {
			d.Config.ConfigModified()
		}
	}()

	d.confLock.Lock()
	defer d.confLock.Unlock()

	index = slices.IndexFunc(d.Config.Rewrites, rwDel.equal)
	if index == -1 {
		aghhttp.Error(r, w, http.StatusBadRequest, "target rule not found")

		return
	}

	d.Config.Rewrites = slices.Replace(d.Config.Rewrites, index, index+1, rwAdd)

	log.Debug("rewrite: removed element: %s -> %s", rwDel.Domain, rwDel.Answer)
	log.Debug("rewrite: added element: %s -> %s", rwAdd.Domain, rwAdd.Answer)
}
