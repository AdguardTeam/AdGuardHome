package websvc

import (
	"net/http"
	"runtime"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
)

// System Handlers

// RespGetV1SystemInfo describes the response of the GET /api/v1/system/info
// HTTP API.
type RespGetV1SystemInfo struct {
	Arch       string           `json:"arch"`
	Channel    string           `json:"channel"`
	OS         string           `json:"os"`
	NewVersion string           `json:"new_version,omitempty"`
	Start      aghhttp.JSONTime `json:"start"`
	Version    string           `json:"version"`
}

// handleGetV1SystemInfo is the handler for the GET /api/v1/system/info HTTP
// API.
func (svc *Service) handleGetV1SystemInfo(w http.ResponseWriter, r *http.Request) {
	aghhttp.WriteJSONResponseOK(w, r, &RespGetV1SystemInfo{
		Arch:    runtime.GOARCH,
		Channel: version.Channel(),
		OS:      runtime.GOOS,
		// TODO(a.garipov): Fill this when we have an updater.
		NewVersion: "",
		Start:      aghhttp.JSONTime(svc.start),
		Version:    version.Version(),
	})
}
