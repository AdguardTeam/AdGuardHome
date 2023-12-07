package websvc

import (
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
)

// Middlewares

// jsonMw sets the content type of the response to application/json.
func jsonMw(h http.Handler) (wrapped http.HandlerFunc) {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httphdr.ContentType, aghhttp.HdrValApplicationJSON)

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}

// logMw logs the queries with level debug.
func logMw(h http.Handler) (wrapped http.HandlerFunc) {
	f := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m, u := r.Method, r.RequestURI

		log.Debug("websvc: %s %s started", m, u)
		defer func() { log.Debug("websvc: %s %s finished in %s", m, u, time.Since(start)) }()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}
