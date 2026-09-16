package home

import (
	"log/slog"
	"net/http"

	"github.com/AdguardTeam/golibs/netutil/httputil"
)

// doHServer represents a DNS-over-HTTPS server.
type doHServer struct {
	// mux matches the DoH route patterns and serves the matched requests with
	// the DoH handler.  It must not be nil.
	mux *http.ServeMux
}

// newDoHServer returns a new properly initialized *doHServer.  logger and
// handler must not be nil.
func newDoHServer(logger *slog.Logger, handler http.Handler, routes []string) (srv *doHServer) {
	h := httputil.Wrap(handler, httputil.MiddlewareFunc(limitRequestBody))

	logMw := httputil.NewLogMiddleware(logger, slog.LevelDebug)
	h = logMw.Wrap(h)

	mux := http.NewServeMux()
	for _, route := range routes {
		mux.Handle(route, h)
	}

	return &doHServer{mux: mux}
}

// tryServe serves r with the DoH handler if r matches one of the DoH routes and
// reports whether it did.  w must not be nil.
func (srv *doHServer) tryServe(w http.ResponseWriter, r *http.Request) (ok bool) {
	_, pattern := srv.mux.Handler(r)
	if pattern == "" {
		return false
	}

	srv.mux.ServeHTTP(w, r)

	return true
}
