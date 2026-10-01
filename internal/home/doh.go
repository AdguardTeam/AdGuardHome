package home

import (
	"log/slog"
	"net/http"

	"github.com/AdguardTeam/golibs/netutil/httputil"
)

// doHServerConfig is the configuration for a DNS-over-HTTPS server.
type doHServerConfig struct {
	// handler is the DoH handler that serves the DoH requests.  It must not be
	// nil.
	handler http.Handler

	// logger is the logger used by the DoH server.  It must not be nil.
	logger *slog.Logger

	// routes are the route patterns that the DoH server will handle.  Each
	// route entry must be a valid HTTP route pattern.
	routes []string
}

// doHServer represents a DNS-over-HTTPS server.
type doHServer struct {
	// mux matches the DoH route patterns and serves the matched requests with
	// the DoH handler.  It must not be nil.
	mux *http.ServeMux
}

// newDoHServer returns a new properly initialized *doHServer.  c must be valid.
func newDoHServer(c *doHServerConfig) (srv *doHServer) {
	h := httputil.Wrap(c.handler, httputil.MiddlewareFunc(limitRequestBody))

	logMw := httputil.NewLogMiddleware(c.logger, slog.LevelDebug)
	h = logMw.Wrap(h)

	mux := http.NewServeMux()
	for _, route := range c.routes {
		mux.Handle(route, h)
	}

	return &doHServer{mux: mux}
}

// tryServe serves r with the DoH handler if r matches one of the DoH routes and
// reports whether it did.  r and w must not be nil.
func (srv *doHServer) tryServe(w http.ResponseWriter, r *http.Request) (ok bool) {
	_, pattern := srv.mux.Handler(r)
	if pattern == "" {
		return false
	}

	srv.mux.ServeHTTP(w, r)

	return true
}
