package websvc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/next/jsonpatch"
	"github.com/AdguardTeam/dnsproxy/proxy"
)

// ReqPatchSettingsDNS describes the request to the PATCH /api/v1/settings/dns
// HTTP API.
type ReqPatchSettingsDNS struct {
	// TODO(a.garipov): Add more as we go.

	UpstreamMode jsonpatch.NonRemovable[proxy.UpstreamMode] `json:"upstream_mode"`

	Addresses        jsonpatch.NonRemovable[[]netip.AddrPort] `json:"addresses"`
	BootstrapServers jsonpatch.NonRemovable[[]string]         `json:"bootstrap_servers"`
	UpstreamServers  jsonpatch.NonRemovable[[]string]         `json:"upstream_servers"`
	DNS64Prefixes    jsonpatch.NonRemovable[[]netip.Prefix]   `json:"dns64_prefixes"`

	UpstreamTimeout jsonpatch.NonRemovable[aghhttp.JSONDuration] `json:"upstream_timeout"`

	CacheSize jsonpatch.NonRemovable[int] `json:"cache_size"`
	Ratelimit jsonpatch.NonRemovable[int] `json:"ratelimit"`

	BootstrapPreferIPv6 jsonpatch.NonRemovable[bool] `json:"bootstrap_prefer_ipv6"`
	RefuseAny           jsonpatch.NonRemovable[bool] `json:"refuse_any"`
	UseDNS64            jsonpatch.NonRemovable[bool] `json:"use_dns64"`
}

// HTTPAPIDNSSettings are the DNS settings as used by the HTTP API.  See the
// DnsSettings object in the OpenAPI specification.
type HTTPAPIDNSSettings struct {
	// TODO(a.garipov): Add more as we go.

	UpstreamMode proxy.UpstreamMode `json:"upstream_mode"`

	Addresses []netip.AddrPort `json:"addresses"`

	BootstrapServers []string `json:"bootstrap_servers"`
	UpstreamServers  []string `json:"upstream_servers"`

	DNS64Prefixes []netip.Prefix `json:"dns64_prefixes"`

	UpstreamTimeout aghhttp.JSONDuration `json:"upstream_timeout"`

	Ratelimit int `json:"ratelimit"`
	CacheSize int `json:"cache_size"`

	BootstrapPreferIPv6 bool `json:"bootstrap_prefer_ipv6"`
	RefuseAny           bool `json:"refuse_any"`
	UseDNS64            bool `json:"use_dns64"`
}

// handlePatchSettingsDNS is the handler for the PATCH /api/v1/settings/dns HTTP
// API.
func (svc *Service) handlePatchSettingsDNS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := svc.logger
	req := &ReqPatchSettingsDNS{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.WriteJSONResponseError(ctx, l, w, r, fmt.Errorf("decoding: %w", err))

		return
	}

	dnsSvc := svc.confMgr.DNS()
	newConf := dnsSvc.Config()

	// TODO(a.garipov): Add more as we go.

	req.UpstreamMode.Set(&newConf.UpstreamMode)

	req.Addresses.Set(&newConf.Addresses)
	req.BootstrapServers.Set(&newConf.BootstrapServers)
	req.UpstreamServers.Set(&newConf.UpstreamServers)
	req.DNS64Prefixes.Set(&newConf.DNS64Prefixes)

	req.UpstreamTimeout.Set((*aghhttp.JSONDuration)(&newConf.UpstreamTimeout))

	if req.CacheSize.IsSet {
		newConf.CacheSize = req.CacheSize.Value
		newConf.CacheEnabled = req.CacheSize.Value > 0
	}
	req.Ratelimit.Set(&newConf.Ratelimit)

	req.BootstrapPreferIPv6.Set(&newConf.BootstrapPreferIPv6)
	req.RefuseAny.Set(&newConf.RefuseAny)
	req.UseDNS64.Set(&newConf.UseDNS64)

	err = svc.confMgr.UpdateDNS(ctx, newConf)
	if err != nil {
		aghhttp.WriteJSONResponseError(ctx, l, w, r, fmt.Errorf("updating: %w", err))

		return
	}

	newSvc := svc.confMgr.DNS()
	err = newSvc.Start(ctx)
	if err != nil {
		aghhttp.WriteJSONResponseError(ctx, l, w, r, fmt.Errorf("starting new service: %w", err))

		return
	}

	aghhttp.WriteJSONResponseOK(ctx, l, w, r, &HTTPAPIDNSSettings{
		UpstreamMode:        newConf.UpstreamMode,
		Addresses:           newConf.Addresses,
		BootstrapServers:    newConf.BootstrapServers,
		UpstreamServers:     newConf.UpstreamServers,
		DNS64Prefixes:       newConf.DNS64Prefixes,
		UpstreamTimeout:     aghhttp.JSONDuration(newConf.UpstreamTimeout),
		Ratelimit:           newConf.Ratelimit,
		BootstrapPreferIPv6: newConf.BootstrapPreferIPv6,
		CacheSize:           newConf.CacheSize,
		RefuseAny:           newConf.RefuseAny,
		UseDNS64:            newConf.UseDNS64,
	})
}
