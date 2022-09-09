package websvc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/golibs/timeutil"
)

// DNS Settings Handlers

// TODO(a.garipov): !! Write tests!

// ReqPatchSettingsDNS describes the request to the PATCH /api/v1/settings/dns
// HTTP API.
type ReqPatchSettingsDNS struct {
	// TODO(a.garipov): Add more as we go.

	Addresses        []netip.AddrPort  `json:"addresses"`
	BootstrapServers []string          `json:"bootstrap_servers"`
	UpstreamServers  []string          `json:"upstream_servers"`
	UpstreamTimeout  timeutil.Duration `json:"upstream_timeout"`
}

// HTTPAPIDNSSettings are the DNS settings as used by the HTTP API.  See the
// DnsSettings object in the OpenAPI specification.
type HTTPAPIDNSSettings struct {
	// TODO(a.garipov): Add more as we go.

	Addresses        []netip.AddrPort  `json:"addresses"`
	BootstrapServers []string          `json:"bootstrap_servers"`
	UpstreamServers  []string          `json:"upstream_servers"`
	UpstreamTimeout  timeutil.Duration `json:"upstream_timeout"`
}

// handlePatchSettingsDNS is the handler for the PATCH /api/v1/settings/dns HTTP
// API.
func (svc *Service) handlePatchSettingsDNS(w http.ResponseWriter, r *http.Request) {
	req := &ReqPatchSettingsDNS{
		Addresses:        []netip.AddrPort{},
		BootstrapServers: []string{},
		UpstreamServers:  []string{},
	}

	// TODO(a.garipov): Validate nulls and proper JSON patch.

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeHTTPError(w, r, fmt.Errorf("decoding: %w", err))

		return
	}

	newConf := &dnssvc.Config{
		Addresses:        req.Addresses,
		BootstrapServers: req.BootstrapServers,
		UpstreamServers:  req.UpstreamServers,
		UpstreamTimeout:  req.UpstreamTimeout.Duration,
	}

	ctx := r.Context()
	err = svc.confMgr.UpdateDNS(ctx, newConf)
	if err != nil {
		writeHTTPError(w, r, fmt.Errorf("updating: %w", err))

		return
	}

	newSvc := svc.confMgr.DNS()
	err = newSvc.Start()
	if err != nil {
		writeHTTPError(w, r, fmt.Errorf("starting new service: %w", err))

		return
	}

	writeJSONResponse(w, r, &HTTPAPIDNSSettings{
		Addresses:        newConf.Addresses,
		BootstrapServers: newConf.BootstrapServers,
		UpstreamServers:  newConf.UpstreamServers,
		UpstreamTimeout:  timeutil.Duration{Duration: newConf.UpstreamTimeout},
	})
}
