package home

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/AdguardTeam/golibs/netutil/httputil"
)

// doHServer represents a DNS-over-HTTPS server.
type doHServer struct {
	// logger is used for logging operations of the server.  It must not be nil.
	logger *slog.Logger

	// handler handles DoH requests.  It must not be nil.
	handler http.Handler

	// routes is the list of HTTP route patterns for DoH requests.
	routes []string
}

// newDoHServer returns a new properly initialized *doHServer.  logger and
// handler must not be nil.
func newDoHServer(logger *slog.Logger, handler http.Handler, routes []string) (srv *doHServer) {
	return &doHServer{
		logger:  logger,
		handler: handler,
		routes:  slices.Clone(routes),
	}
}

// wrapRoutes returns a handler that serves the DoH routes of srv and passes
// all other requests to h.  h must not be nil.
func (srv *doHServer) wrapRoutes(h http.Handler) (wrapped http.Handler) {
	dohHdlr := httputil.Wrap(srv.handler, httputil.MiddlewareFunc(limitRequestBody))

	logMw := httputil.NewLogMiddleware(srv.logger, slog.LevelDebug)
	dohHdlr = logMw.Wrap(dohHdlr)

	mux := http.NewServeMux()
	for _, route := range srv.routes {
		mux.Handle(route, dohHdlr)
	}

	mux.Handle("/", h)

	return mux
}
