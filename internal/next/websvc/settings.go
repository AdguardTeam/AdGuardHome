package websvc

import (
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// All Settings Handlers

// RespGetV1SettingsAll describes the response of the GET /api/v1/settings/all
// HTTP API.
type RespGetV1SettingsAll struct {
	// TODO(a.garipov): Add more as we go.

	DNS  *HTTPAPIDNSSettings  `json:"dns"`
	HTTP *HTTPAPIHTTPSettings `json:"http"`
}

// handleGetSettingsAll is the handler for the GET /api/v1/settings/all HTTP
// API.
func (svc *Service) handleGetSettingsAll(w http.ResponseWriter, r *http.Request) {
	dnsSvc := svc.confMgr.DNS()
	dnsConf := dnsSvc.Config()

	webSvc := svc.confMgr.Web()
	httpConf := webSvc.Config()

	// TODO(a.garipov): Add all currently supported parameters.
	aghhttp.WriteJSONResponseOK(w, r, &RespGetV1SettingsAll{
		DNS: &HTTPAPIDNSSettings{
			Addresses:           dnsConf.Addresses,
			BootstrapServers:    dnsConf.BootstrapServers,
			UpstreamServers:     dnsConf.UpstreamServers,
			DNS64Prefixes:       dnsConf.DNS64Prefixes,
			UpstreamTimeout:     aghhttp.JSONDuration(dnsConf.UpstreamTimeout),
			BootstrapPreferIPv6: dnsConf.BootstrapPreferIPv6,
			UseDNS64:            dnsConf.UseDNS64,
		},
		HTTP: &HTTPAPIHTTPSettings{
			Addresses:       httpConf.Addresses,
			SecureAddresses: httpConf.SecureAddresses,
			Timeout:         aghhttp.JSONDuration(httpConf.Timeout),
			ForceHTTPS:      httpConf.ForceHTTPS,
		},
	})
}
