package websvc

import (
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/golibs/timeutil"
)

// All Settings Handlers

// TODO(a.garipov): !! Write tests!

// RespGetV1SettingsAll describes the response of the GET /api/v1/settings/all
// HTTP API.
type RespGetV1SettingsAll struct {
	// TODO(a.garipov): Add more as we go.

	DNS  *respGetV1SettingsAllDNS  `json:"dns"`
	HTTP *respGetV1SettingsAllHTTP `json:"http"`
}

// respGetV1SettingsAllDNS describes the DNS part of the response of the GET
// /api/v1/settings/all HTTP API.
type respGetV1SettingsAllDNS struct {
	// TODO(a.garipov): Add more as we go.

	Addresses        []netip.AddrPort  `json:"addresses"`
	BootstrapServers []string          `json:"bootstrap_servers"`
	UpstreamServers  []string          `json:"upstream_servers"`
	UpstreamTimeout  timeutil.Duration `json:"upstream_timeout"`
}

// respGetV1SettingsAllHTTP describes the HTTP part of the response of the GET
// /api/v1/settings/all HTTP API.
type respGetV1SettingsAllHTTP struct {
	// TODO(a.garipov): Add more as we go.

	Addresses       []netip.AddrPort `json:"addresses"`
	SecureAddresses []netip.AddrPort `json:"secure_addresses"`
}

// handleGetSettingsAll is the handler for the GET /api/v1/settings/all HTTP
// API.
func (svc *Service) handleGetSettingsAll(w http.ResponseWriter, r *http.Request) {
	dnsSvc := svc.confMgr.DNS()
	dnsConf := dnsSvc.Config()

	httpConf := svc.Config()

	writeJSONResponse(w, r, &RespGetV1SettingsAll{
		DNS: &respGetV1SettingsAllDNS{
			Addresses:        dnsConf.Addresses,
			BootstrapServers: dnsConf.BootstrapServers,
			UpstreamServers:  dnsConf.UpstreamServers,
			UpstreamTimeout:  timeutil.Duration{Duration: dnsConf.UpstreamTimeout},
		},
		HTTP: &respGetV1SettingsAllHTTP{
			Addresses:       httpConf.Addresses,
			SecureAddresses: httpConf.SecureAddresses,
		},
	})
}
