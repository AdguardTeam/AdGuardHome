package websvc

import (
	"net/http"

	"github.com/AdguardTeam/golibs/timeutil"
)

// All Settings Handlers

// TODO(a.garipov): !! Write tests!

// RespGetV1SettingsAll describes the response of the GET /api/v1/settings/all
// HTTP API.
type RespGetV1SettingsAll struct {
	// TODO(a.garipov): Add more as we go.

	DNS  *httpAPIDNSSettings  `json:"dns"`
	HTTP *httpAPIHTTPSettings `json:"http"`
}

// handleGetSettingsAll is the handler for the GET /api/v1/settings/all HTTP
// API.
func (svc *Service) handleGetSettingsAll(w http.ResponseWriter, r *http.Request) {
	dnsSvc := svc.confMgr.DNS()
	dnsConf := dnsSvc.Config()

	httpConf := svc.Config()

	writeJSONResponse(w, r, &RespGetV1SettingsAll{
		DNS: &httpAPIDNSSettings{
			Addresses:        dnsConf.Addresses,
			BootstrapServers: dnsConf.BootstrapServers,
			UpstreamServers:  dnsConf.UpstreamServers,
			UpstreamTimeout:  timeutil.Duration{Duration: dnsConf.UpstreamTimeout},
		},
		HTTP: &httpAPIHTTPSettings{
			Addresses:       httpConf.Addresses,
			SecureAddresses: httpConf.SecureAddresses,
		},
	})
}
