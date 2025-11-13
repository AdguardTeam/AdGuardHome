package filtering

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// rewriteEntryJSON is a single entry of the DNS rewrite.
//
// TODO(d.kolyshev): Use [rewrite.Item] instead.
type rewriteEntryJSON struct {
	Domain  string          `json:"domain"`
	Answer  string          `json:"answer"`
	Enabled aghalg.NullBool `json:"enabled"`
}

// rewriteSettings contains DNS rewrite settings.
type rewriteSettings struct {
	// Enabled indicates whether legacy rewrites are applied.
	//
	// TODO(s.chzhen):  Consider using [aghalg.NullBool] so "{}" won't
	// accidentally disable rewrites on decode.
	Enabled bool `json:"enabled"`
}

// handleRewriteList is the handler for the GET /control/rewrite/list HTTP API.
func (d *DNSFilter) handleRewriteList(w http.ResponseWriter, r *http.Request) {
	arr := []*rewriteEntryJSON{}

	func() {
		d.confMu.RLock()
		defer d.confMu.RUnlock()

		for _, ent := range d.conf.Rewrites {
			jsonEnt := rewriteEntryJSON{
				Domain:  ent.Domain,
				Answer:  ent.Answer,
				Enabled: aghalg.BoolToNullBool(ent.Enabled),
			}
			arr = append(arr, &jsonEnt)
		}
	}()

	aghhttp.WriteJSONResponseOK(r.Context(), d.logger, w, r, arr)
}

// handleRewriteAdd is the handler for the POST /control/rewrite/add HTTP API.
func (d *DNSFilter) handleRewriteAdd(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := d.logger

	rwJSON := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&rwJSON)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	enabled := true
	if rwJSON.Enabled != aghalg.NBNull {
		enabled = rwJSON.Enabled == aghalg.NBTrue
	}

	rw := &LegacyRewrite{
		Domain:  rwJSON.Domain,
		Answer:  rwJSON.Answer,
		Enabled: enabled,
	}

	err = rw.normalize(ctx, l)
	if err != nil {
		// Shouldn't happen currently, since normalize only returns a non-nil
		// error when a rewrite is nil, but be change-proof.
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "normalizing: %s", err)

		return
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		d.conf.Rewrites = append(d.conf.Rewrites, rw)
		l.DebugContext(
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
	l := d.logger

	jsent := rewriteEntryJSON{}
	err := json.NewDecoder(r.Body).Decode(&jsent)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "json.Decode: %s", err)

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

		l.DebugContext(
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
	l := d.logger

	updateJSON := rewriteUpdateJSON{}
	err := json.NewDecoder(r.Body).Decode(&updateJSON)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "json.Decode: %s", err)

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

	err = rwAdd.normalize(ctx, l)
	if err != nil {
		// Shouldn't happen currently, since normalize only returns a non-nil
		// error when a rewrite is nil, but be change-proof.
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "normalizing: %s", err)

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
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "target rule not found")

		return
	}

	rwDel.Enabled = d.conf.Rewrites[index].Enabled
	if updateJSON.Update.Enabled == aghalg.NBNull {
		rwAdd.Enabled = rwDel.Enabled
	} else {
		rwAdd.Enabled = updateJSON.Update.Enabled == aghalg.NBTrue
	}

	d.conf.Rewrites = slices.Replace(d.conf.Rewrites, index, index+1, rwAdd)

	l.DebugContext(
		ctx,
		"removed rewrite element",
		"domain", rwDel.Domain,
		"answer", rwDel.Answer,
		"enabled", rwDel.Enabled,
	)
	l.DebugContext(
		ctx,
		"added rewrite element",
		"domain", rwAdd.Domain,
		"answer", rwAdd.Answer,
		"enabled", rwAdd.Enabled,
	)
}

// handleRewriteSettings is the handler for the GET /control/rewrite/settings
// HTTP API.
func (d *DNSFilter) handleRewriteSettings(w http.ResponseWriter, r *http.Request) {
	resp := &rewriteSettings{
		Enabled: protectedBool(d.confMu, &d.conf.RewritesEnabled),
	}

	aghhttp.WriteJSONResponseOK(r.Context(), d.logger, w, r, resp)
}

// handleRewriteSettingsUpdate is the handler for the PUT
// /control/rewrite/settings/update HTTP API.
func (d *DNSFilter) handleRewriteSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := &rewriteSettings{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, d.logger, r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	setProtectedBool(d.confMu, &d.conf.RewritesEnabled, req.Enabled)
	d.conf.ConfModifier.Apply(ctx)
}
