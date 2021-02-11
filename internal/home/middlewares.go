package home

import (
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"

	"github.com/AdguardTeam/golibs/log"
)

// middlerware is a wrapper function signature.
type middleware func(http.Handler) http.Handler

// withMiddlewares consequently wraps h with all the middlewares.
func withMiddlewares(h http.Handler, middlewares ...middleware) (wrapped http.Handler) {
	wrapped = h

	for _, mw := range middlewares {
		wrapped = mw(wrapped)
	}

	return wrapped
}

// RequestBodySizeLimit is maximum request body length in bytes.
const RequestBodySizeLimit = 64 * 1024

// limitRequestBody wraps underlying handler h, making it's request's body Read
// method limited.
func limitRequestBody(h http.Handler) (limited http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		var bodySizeLimit int64 = RequestBodySizeLimit
		if u := r.URL; u.Path == "/control/access/set" {
			// An exception for a poorly designed API.  Remove once
			// the new, better API is up.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/2666.
			bodySizeLimit *= 4
		}

		r.Body, err = aghio.LimitReadCloser(r.Body, bodySizeLimit)
		if err != nil {
			log.Error("limitRequestBody: %s", err)

			return
		}

		h.ServeHTTP(w, r)
	})
}

// wrapIndexBeta returns handler that deals with new client.
func (web *Web) wrapIndexBeta(http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := Context.mux.Handler(r)
		switch pattern {
		case "/":
			web.handlerBeta.ServeHTTP(w, r)
		case "/install.html":
			web.installerBeta.ServeHTTP(w, r)
		default:
			h.ServeHTTP(w, r)
		}
	})
}
