//go:build windows

package dhcpd

import (
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

// jsonError is a generic JSON error response.
//
// TODO(a.garipov): Merge together with the implementations in .../home and
// other packages after refactoring the web handler registering.
type jsonError struct {
	// Message is the error message, an opaque string.
	Message string `json:"message"`
}

// notImplemented is a handler that replies to any request with an HTTP 501 Not
// Implemented status and a JSON error with the provided message msg.
//
// TODO(a.garipov): Either take the logger from the server after we've
// refactored logging or make this not a method of *Server.
func (s *server) notImplemented(w http.ResponseWriter, r *http.Request) {
	aghhttp.WriteJSONResponse(w, r, http.StatusNotImplemented, &jsonError{
		Message: aghos.Unsupported("dhcp").Error(),
	})
}

// registerHandlers sets the handlers for DHCP HTTP API that always respond with
// an HTTP 501, since DHCP server doesn't work on Windows yet.
//
// TODO(a.garipov): This needs refactoring.  We shouldn't even try and
// initialize a DHCP server on Windows, but there are currently too many
// interconnected parts--such as HTTP handlers and frontend--to make that work
// properly.
func (s *server) registerHandlers() {
	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/status", s.notImplemented)
	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/interfaces", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/set_config", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/find_active_dhcp", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/add_static_lease", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/remove_static_lease", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/update_static_lease", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/reset", s.notImplemented)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/reset_leases", s.notImplemented)
}
