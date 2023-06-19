package websvc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/golibs/log"
)

// HTTP Settings Handlers

// ReqPatchSettingsHTTP describes the request to the PATCH /api/v1/settings/http
// HTTP API.
type ReqPatchSettingsHTTP struct {
	// TODO(a.garipov): Add more as we go.
	//
	// TODO(a.garipov): Add wait time.

	Addresses       []netip.AddrPort `json:"addresses"`
	SecureAddresses []netip.AddrPort `json:"secure_addresses"`
	Timeout         JSONDuration     `json:"timeout"`
}

// HTTPAPIHTTPSettings are the HTTP settings as used by the HTTP API.  See the
// HttpSettings object in the OpenAPI specification.
type HTTPAPIHTTPSettings struct {
	// TODO(a.garipov): Add more as we go.

	Addresses       []netip.AddrPort `json:"addresses"`
	SecureAddresses []netip.AddrPort `json:"secure_addresses"`
	Timeout         JSONDuration     `json:"timeout"`
	ForceHTTPS      bool             `json:"force_https"`
}

// handlePatchSettingsHTTP is the handler for the PATCH /api/v1/settings/http
// HTTP API.
func (svc *Service) handlePatchSettingsHTTP(w http.ResponseWriter, r *http.Request) {
	req := &ReqPatchSettingsHTTP{}

	// TODO(a.garipov): Validate nulls and proper JSON patch.

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONErrorResponse(w, r, fmt.Errorf("decoding: %w", err))

		return
	}

	newConf := &Config{
		ConfigManager:   svc.confMgr,
		Frontend:        svc.frontend,
		TLS:             svc.tls,
		Addresses:       req.Addresses,
		SecureAddresses: req.SecureAddresses,
		Timeout:         time.Duration(req.Timeout),
		ForceHTTPS:      svc.forceHTTPS,
	}

	writeJSONOKResponse(w, r, &HTTPAPIHTTPSettings{
		Addresses:       newConf.Addresses,
		SecureAddresses: newConf.SecureAddresses,
		Timeout:         JSONDuration(newConf.Timeout),
		ForceHTTPS:      newConf.ForceHTTPS,
	})

	cancelUpd := func() {}
	updCtx := context.Background()

	ctx := r.Context()
	if deadline, ok := ctx.Deadline(); ok {
		updCtx, cancelUpd = context.WithDeadline(updCtx, deadline)
	}

	// Launch the new HTTP service in a separate goroutine to let this handler
	// finish and thus, this server to shutdown.
	go svc.relaunch(updCtx, cancelUpd, newConf)
}

// relaunch updates the web service in the configuration manager and starts it.
// It is intended to be used as a goroutine.
func (svc *Service) relaunch(ctx context.Context, cancel context.CancelFunc, newConf *Config) {
	defer log.OnPanic("websvc: relaunching")

	defer cancel()

	err := svc.confMgr.UpdateWeb(ctx, newConf)
	if err != nil {
		log.Error("websvc: updating web: %s", err)

		return
	}

	// TODO(a.garipov): Consider better ways to do this.
	const maxUpdDur = 5 * time.Second
	updStart := time.Now()
	var newSvc agh.ServiceWithConfig[*Config]
	for newSvc = svc.confMgr.Web(); newSvc == svc; {
		if time.Since(updStart) >= maxUpdDur {
			log.Error("websvc: failed to update svc after %s", maxUpdDur)

			return
		}

		log.Debug("websvc: waiting for new websvc to be configured")

		time.Sleep(100 * time.Millisecond)
	}

	err = newSvc.Start()
	if err != nil {
		log.Error("websvc: new svc failed to start with error: %s", err)
	}
}
