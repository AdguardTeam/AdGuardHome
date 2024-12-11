package websvc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/next/jsonpatch"
)

// ReqPatchSettingsDNS describes the request to the PATCH /api/v1/settings/dns
// HTTP API.
type ReqPatchSettingsDNS struct {
	// TODO(a.garipov): Add more as we go.

	Addresses        jsonpatch.NonRemovable[[]netip.AddrPort] `json:"addresses"`
	BootstrapServers jsonpatch.NonRemovable[[]string]         `json:"bootstrap_servers"`
	UpstreamServers  jsonpatch.NonRemovable[[]string]         `json:"upstream_servers"`
	DNS64Prefixes    jsonpatch.NonRemovable[[]netip.Prefix]   `json:"dns64_prefixes"`

	UpstreamTimeout jsonpatch.NonRemovable[aghhttp.JSONDuration] `json:"upstream_timeout"`

	BootstrapPreferIPv6 jsonpatch.NonRemovable[bool] `json:"bootstrap_prefer_ipv6"`
	UseDNS64            jsonpatch.NonRemovable[bool] `json:"use_dns64"`
}

// HTTPAPIDNSSettings are the DNS settings as used by the HTTP API.  See the
// DnsSettings object in the OpenAPI specification.
type HTTPAPIDNSSettings struct {
	// TODO(a.garipov): Add more as we go.

	Addresses           []netip.AddrPort     `json:"addresses"`
	BootstrapServers    []string             `json:"bootstrap_servers"`
	UpstreamServers     []string             `json:"upstream_servers"`
	DNS64Prefixes       []netip.Prefix       `json:"dns64_prefixes"`
	UpstreamTimeout     aghhttp.JSONDuration `json:"upstream_timeout"`
	BootstrapPreferIPv6 bool                 `json:"bootstrap_prefer_ipv6"`
	UseDNS64            bool                 `json:"use_dns64"`
}

// handlePatchSettingsDNS is the handler for the PATCH /api/v1/settings/dns HTTP
// API.
func (svc *Service) handlePatchSettingsDNS(w http.ResponseWriter, r *http.Request) {
	req := &ReqPatchSettingsDNS{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.WriteJSONResponseError(w, r, fmt.Errorf("decoding: %w", err))

		return
	}

	dnsSvc := svc.confMgr.DNS()
	newConf := dnsSvc.Config()

	// TODO(a.garipov): Add more as we go.

	req.Addresses.Set(&newConf.Addresses)
	req.BootstrapServers.Set(&newConf.BootstrapServers)
	req.UpstreamServers.Set(&newConf.UpstreamServers)
	req.DNS64Prefixes.Set(&newConf.DNS64Prefixes)

	req.UpstreamTimeout.Set((*aghhttp.JSONDuration)(&newConf.UpstreamTimeout))

	req.BootstrapPreferIPv6.Set(&newConf.BootstrapPreferIPv6)
	req.UseDNS64.Set(&newConf.UseDNS64)

	ctx := r.Context()
	err = svc.confMgr.UpdateDNS(ctx, newConf)
	if err != nil {
		aghhttp.WriteJSONResponseError(w, r, fmt.Errorf("updating: %w", err))

		return
	}

	newSvc := svc.confMgr.DNS()
	err = newSvc.Start(ctx)
	if err != nil {
		aghhttp.WriteJSONResponseError(w, r, fmt.Errorf("starting new service: %w", err))

		return
	}

	aghhttp.WriteJSONResponseOK(w, r, &HTTPAPIDNSSettings{
		Addresses:           newConf.Addresses,
		BootstrapServers:    newConf.BootstrapServers,
		UpstreamServers:     newConf.UpstreamServers,
		DNS64Prefixes:       newConf.DNS64Prefixes,
		UpstreamTimeout:     aghhttp.JSONDuration(newConf.UpstreamTimeout),
		BootstrapPreferIPv6: newConf.BootstrapPreferIPv6,
		UseDNS64:            newConf.UseDNS64,
	})
}
