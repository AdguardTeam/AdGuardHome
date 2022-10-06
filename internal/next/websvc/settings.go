package websvc

import (
	"net/http"
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
	writeJSONOKResponse(w, r, &RespGetV1SettingsAll{
		DNS: &HTTPAPIDNSSettings{
			Addresses:        dnsConf.Addresses,
			BootstrapServers: dnsConf.BootstrapServers,
			UpstreamServers:  dnsConf.UpstreamServers,
			UpstreamTimeout:  JSONDuration(dnsConf.UpstreamTimeout),
		},
		HTTP: &HTTPAPIHTTPSettings{
			Addresses:       httpConf.Addresses,
			SecureAddresses: httpConf.SecureAddresses,
			Timeout:         JSONDuration(httpConf.Timeout),
			ForceHTTPS:      httpConf.ForceHTTPS,
		},
	})
}
