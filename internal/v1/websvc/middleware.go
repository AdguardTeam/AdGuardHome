package websvc

import "net/http"

// Middlewares

// jsonMw sets the content type of the response to application/json.
func jsonMw(h http.Handler) (wrapped http.HandlerFunc) {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(f)
}
