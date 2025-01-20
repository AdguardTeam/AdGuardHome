package websvc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/jsonpatch"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// ReqPatchSettingsHTTP describes the request to the PATCH /api/v1/settings/http
// HTTP API.
type ReqPatchSettingsHTTP struct {
	// TODO(a.garipov): Add more as we go.
	//
	// TODO(a.garipov): Add wait time.

	Addresses       jsonpatch.NonRemovable[[]netip.AddrPort] `json:"addresses"`
	SecureAddresses jsonpatch.NonRemovable[[]netip.AddrPort] `json:"secure_addresses"`

	Timeout jsonpatch.NonRemovable[aghhttp.JSONDuration] `json:"timeout"`

	ForceHTTPS jsonpatch.NonRemovable[bool] `json:"force_https"`
}

// HTTPAPIHTTPSettings are the HTTP settings as used by the HTTP API.  See the
// HttpSettings object in the OpenAPI specification.
type HTTPAPIHTTPSettings struct {
	// TODO(a.garipov): Add more as we go.

	Addresses       []netip.AddrPort     `json:"addresses"`
	SecureAddresses []netip.AddrPort     `json:"secure_addresses"`
	Timeout         aghhttp.JSONDuration `json:"timeout"`
	ForceHTTPS      bool                 `json:"force_https"`
}

// handlePatchSettingsHTTP is the handler for the PATCH /api/v1/settings/http
// HTTP API.
func (svc *Service) handlePatchSettingsHTTP(w http.ResponseWriter, r *http.Request) {
	req := &ReqPatchSettingsHTTP{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.WriteJSONResponseError(w, r, fmt.Errorf("decoding: %w", err))

		return
	}

	newConf := svc.Config()

	// TODO(a.garipov): Add more as we go.

	req.Addresses.Set(&newConf.Addresses)
	req.SecureAddresses.Set(&newConf.SecureAddresses)
	req.Timeout.Set((*aghhttp.JSONDuration)(&newConf.Timeout))
	req.ForceHTTPS.Set(&newConf.ForceHTTPS)

	aghhttp.WriteJSONResponseOK(w, r, &HTTPAPIHTTPSettings{
		Addresses:       newConf.Addresses,
		SecureAddresses: newConf.SecureAddresses,
		Timeout:         aghhttp.JSONDuration(newConf.Timeout),
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
	defer slogutil.RecoverAndLog(ctx, svc.logger)

	defer cancel()

	err := svc.confMgr.UpdateWeb(ctx, newConf)
	if err != nil {
		svc.logger.ErrorContext(ctx, "updating web", slogutil.KeyError, err)

		return
	}

	// TODO(a.garipov): Consider better ways to do this.
	const maxUpdDur = 5 * time.Second
	updStart := time.Now()
	var newSvc agh.ServiceWithConfig[*Config]
	for newSvc = svc.confMgr.Web(); newSvc == svc; {
		if time.Since(updStart) >= maxUpdDur {
			svc.logger.ErrorContext(ctx, "failed to update service on time", "duration", maxUpdDur)

			return
		}

		svc.logger.DebugContext(ctx, "waiting for new service")

		time.Sleep(100 * time.Millisecond)
	}

	err = newSvc.Start(ctx)
	if err != nil {
		svc.logger.ErrorContext(ctx, "new service failed", slogutil.KeyError, err)
	}
}
