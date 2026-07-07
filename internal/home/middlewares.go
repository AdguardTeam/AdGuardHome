package home

import (
	"io"
	"net/http"

	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/c2h5oh/datasize"
)

const (
	// defaultReqBodySzLim is the default maximum request body size.
	defaultReqBodySzLim datasize.ByteSize = 64 * datasize.KB

	// largerReqBodySzLim is the maximum request body size for APIs expecting
	// larger requests.
	largerReqBodySzLim datasize.ByteSize = 4 * datasize.MB
)

// expectsLargerRequests shows if this request should use a larger body size
// limit.  These are exceptions for poorly designed current APIs as well as APIs
// that are designed to expect large files and requests.  Remove once the new,
// better APIs are up.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2666 and
// https://github.com/AdguardTeam/AdGuardHome/issues/2675.
func expectsLargerRequests(r *http.Request) (ok bool) {
	if r.Method != http.MethodPost {
		return false
	}

	switch r.URL.Path {
	case "/control/access/set", "/control/filtering/set_rules":
		return true
	default:
		return false
	}
}

// limitRequestBody wraps underlying handler h, making it's request's body Read
// method limited.
func limitRequestBody(h http.Handler) (limited http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		szLim := defaultReqBodySzLim
		if expectsLargerRequests(r) {
			szLim = largerReqBodySzLim
		}

		reader := ioutil.LimitReader(r.Body, szLim.Bytes())

		// HTTP handlers aren't supposed to call r.Body.Close(), so just
		// replace the body in a clone.
		rr := r.Clone(r.Context())
		rr.Body = io.NopCloser(reader)

		h.ServeHTTP(w, rr)
	})
}
