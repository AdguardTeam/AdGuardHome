package filtering

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// TODO(d.kolyshev): Use [rewrite.Item] instead.
type rewriteEntryJSON struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

// handleRewriteList is the handler for the GET /control/rewrite/list HTTP API.
func (d *DNSFilter) handleRewriteList(w http.ResponseWriter, r *http.Request) {
	arr := []*rewriteEntryJSON{}

	func() {
		d.confMu.RLock()
		defer d.confMu.RUnlock()

		for _, ent := range d.conf.Rewrites {
			jsonEnt := rewriteEntryJSON{
				Domain: ent.Domain,
				Answer: ent.Answer,
			}
			arr = append(arr, &jsonEnt)
		}
	}()

	aghhttp.WriteJSONResponseOK(w, r, arr)
}

// handleRewriteAdd is the handler for the POST /control/rewrite/add HTTP API.
func (d *DNSFilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	err = rw.normalize(ctx, d.logger)
	if err != nil {
		// Shouldn't happen currently, since normalize only returns a non-nil
		// error when a rewrite is nil, but be change-proof.
		aghhttp.Error(r, w, http.StatusBadRequest, "normalizing: %s", err)

		return
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		d.conf.Rewrites = append(d.conf.Rewrites, rw)
		d.logger.DebugContext(
			ctx,
			"added rewrite element",
			"domain", rw.Domain,
			"answer", rw.Answer,
			"rewrites_len", len(d.conf.Rewrites),
		)
	}()

	d.conf.ConfModifier.Apply(ctx)
}

// handleRewriteDelete is the handler for the POST /control/rewrite/delete HTTP
// API.
func (d *DNSFilter) handleRewriteDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	defer d.conf.ConfModifier.Apply(ctx)

	d.confMu.Lock()
	defer d.confMu.Unlock()

	for _, ent := range d.conf.Rewrites {
		if !ent.equal(entDel) {
			arr = append(arr, ent)

			continue
		}

		d.logger.DebugContext(
			ctx,
			"removed rewrite element",
			"domain", ent.Domain,
			"answer", ent.Answer,
		)
	}

	d.conf.Rewrites = arr
}

// rewriteUpdateJSON is a struct for JSON object with rewrite rule update info.
type rewriteUpdateJSON struct {
	Target rewriteEntryJSON `json:"target"`
	Update rewriteEntryJSON `json:"update"`
}

// handleRewriteUpdate is the handler for the PUT /control/rewrite/update HTTP
// API.
func (d *DNSFilter) handleRewriteUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	err = rwAdd.normalize(ctx, d.logger)
	if err != nil {
		// Shouldn't happen currently, since normalize only returns a non-nil
		// error when a rewrite is nil, but be change-proof.
		aghhttp.Error(r, w, http.StatusBadRequest, "normalizing: %s", err)

		return
	}

	index := -1
	defer func() {
		if index >= 0 {
			d.conf.ConfModifier.Apply(ctx)
		}
	}()

	d.confMu.Lock()
	defer d.confMu.Unlock()

	index = slices.IndexFunc(d.conf.Rewrites, rwDel.equal)
	if index == -1 {
		aghhttp.Error(r, w, http.StatusBadRequest, "target rule not found")

		return
	}

	d.conf.Rewrites = slices.Replace(d.conf.Rewrites, index, index+1, rwAdd)

	d.logger.DebugContext(
		ctx,
		"removed rewrite element",
		"domain", rwDel.Domain,
		"answer", rwDel.Answer,
	)
	d.logger.DebugContext(
		ctx,
		"added rewrite element",
		"domain", rwAdd.Domain,
		"answer", rwAdd.Answer,
	)
}
