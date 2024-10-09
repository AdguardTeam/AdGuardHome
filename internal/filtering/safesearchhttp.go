package filtering

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// handleSafeSearchEnable is the handler for POST /control/safesearch/enable
// HTTP API.
//
// Deprecated: Use handleSafeSearchSettings.
func (d *DNSFilter) handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(d.confMu, &d.conf.SafeSearchConf.Enabled, true)
	d.conf.ConfigModified()
}

// handleSafeSearchDisable is the handler for POST /control/safesearch/disable
// HTTP API.
//
// Deprecated: Use handleSafeSearchSettings.
func (d *DNSFilter) handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(d.confMu, &d.conf.SafeSearchConf.Enabled, false)
	d.conf.ConfigModified()
}

// handleSafeSearchStatus is the handler for GET /control/safesearch/status
// HTTP API.
func (d *DNSFilter) handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	var resp SafeSearchConfig
	func() {
		d.confMu.RLock()
		defer d.confMu.RUnlock()

		resp = d.conf.SafeSearchConf
	}()

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// handleSafeSearchSettings is the handler for PUT /control/safesearch/settings
// HTTP API.
func (d *DNSFilter) handleSafeSearchSettings(w http.ResponseWriter, r *http.Request) {
	req := &SafeSearchConfig{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	conf := *req
	err = d.safeSearch.Update(r.Context(), conf)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "updating: %s", err)

		return
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()

		d.conf.SafeSearchConf = conf
	}()

	d.conf.ConfigModified()

	aghhttp.OK(w)
}
