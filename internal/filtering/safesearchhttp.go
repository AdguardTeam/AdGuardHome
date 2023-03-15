package filtering

import (
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// TODO(d.kolyshev): Replace handlers below with the new API.

func (d *DNSFilter) handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.SafeSearchConf.Enabled, true)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.SafeSearchConf.Enabled, false)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	resp := &struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: protectedBool(&d.confLock, &d.Config.SafeSearchConf.Enabled),
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}
