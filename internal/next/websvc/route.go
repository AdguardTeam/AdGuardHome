package websvc

import (
	"log/slog"
	"net/http"

	"github.com/AdguardTeam/golibs/netutil/httputil"
)

// Path pattern constants.
const (
	PathPatternFrontend       = "/"
	PathPatternHealthCheck    = "/health-check"
	PathPatternV1SettingsAll  = "/api/v1/settings/all"
	PathPatternV1SettingsDNS  = "/api/v1/settings/dns"
	PathPatternV1SettingsHTTP = "/api/v1/settings/http"
	PathPatternV1SystemInfo   = "/api/v1/system/info"
)

// Route pattern constants.
const (
	routePatternFrontend            = http.MethodGet + " " + PathPatternFrontend
	routePatternGetV1SettingsAll    = http.MethodGet + " " + PathPatternV1SettingsAll
	routePatternGetV1SystemInfo     = http.MethodGet + " " + PathPatternV1SystemInfo
	routePatternHealthCheck         = http.MethodGet + " " + PathPatternHealthCheck
	routePatternPatchV1SettingsDNS  = http.MethodPatch + " " + PathPatternV1SettingsDNS
	routePatternPatchV1SettingsHTTP = http.MethodPatch + " " + PathPatternV1SettingsHTTP
)

// route registers all necessary handlers in mux.
func (svc *Service) route(mux *http.ServeMux) {
	routes := []struct {
		handler http.Handler
		pattern string
		isJSON  bool
	}{{
		handler: httputil.HealthCheckHandler,
		pattern: routePatternHealthCheck,
		isJSON:  false,
	}, {
		handler: http.FileServer(http.FS(svc.frontend)),
		pattern: routePatternFrontend,
		isJSON:  false,
	}, {
		handler: http.HandlerFunc(svc.handleGetSettingsAll),
		pattern: routePatternGetV1SettingsAll,
		isJSON:  true,
	}, {
		handler: http.HandlerFunc(svc.handlePatchSettingsDNS),
		pattern: routePatternPatchV1SettingsDNS,
		isJSON:  true,
	}, {
		handler: http.HandlerFunc(svc.handlePatchSettingsHTTP),
		pattern: routePatternPatchV1SettingsHTTP,
		isJSON:  true,
	}, {
		handler: http.HandlerFunc(svc.handleGetV1SystemInfo),
		pattern: routePatternGetV1SystemInfo,
		isJSON:  true,
	}}

	logMw := httputil.NewLogMiddleware(svc.logger, slog.LevelDebug)
	for _, r := range routes {
		var hdlr http.Handler
		if r.isJSON {
			hdlr = jsonMw(r.handler)
		} else {
			hdlr = r.handler
		}

		mux.Handle(r.pattern, logMw.Wrap(hdlr))
	}
}
