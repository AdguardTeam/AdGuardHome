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
	PathPatternV2SettingsAll  = "/api/v2/settings/all"
	PathPatternV2SettingsDNS  = "/api/v2/settings/dns"
	PathPatternV2SettingsHTTP = "/api/v2/settings/http"
	PathPatternV2SystemInfo   = "/api/v2/system/info"
)

// Route pattern constants.
const (
	routePatternFrontend            = http.MethodGet + " " + PathPatternFrontend
	routePatternGetV2SettingsAll    = http.MethodGet + " " + PathPatternV2SettingsAll
	routePatternGetV2SystemInfo     = http.MethodGet + " " + PathPatternV2SystemInfo
	routePatternHealthCheck         = http.MethodGet + " " + PathPatternHealthCheck
	routePatternPatchV2SettingsDNS  = http.MethodPatch + " " + PathPatternV2SettingsDNS
	routePatternPatchV2SettingsHTTP = http.MethodPatch + " " + PathPatternV2SettingsHTTP
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
		pattern: routePatternGetV2SettingsAll,
		isJSON:  true,
	}, {
		handler: http.HandlerFunc(svc.handlePatchSettingsDNS),
		pattern: routePatternPatchV2SettingsDNS,
		isJSON:  true,
	}, {
		handler: http.HandlerFunc(svc.handlePatchSettingsHTTP),
		pattern: routePatternPatchV2SettingsHTTP,
		isJSON:  true,
	}, {
		handler: http.HandlerFunc(svc.handleGetV2SystemInfo),
		pattern: routePatternGetV2SystemInfo,
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
