package home

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/c2h5oh/datasize"
)

const (
	// defaultReqBodySzLim is the default maximum request body size.
	defaultReqBodySzLim datasize.ByteSize = 64 * datasize.KB

	// largerReqBodySzLim is the maximum request body size for APIs expecting
	// larger requests.
	largerReqBodySzLim datasize.ByteSize = 4 * datasize.MB
)

// recoveryResponseWriter tracks whether the response has already been
// committed.  It also supports unwrapping by [http.ResponseController].
type recoveryResponseWriter struct {
	http.ResponseWriter

	committed bool
}

// Unwrap returns the underlying response writer.
func (w *recoveryResponseWriter) Unwrap() (rw http.ResponseWriter) {
	return w.ResponseWriter
}

// Write implements the [http.ResponseWriter] interface.
func (w *recoveryResponseWriter) Write(b []byte) (n int, err error) {
	w.committed = true

	return w.ResponseWriter.Write(b)
}

// WriteHeader implements the [http.ResponseWriter] interface.
func (w *recoveryResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)

	if statusCode == http.StatusSwitchingProtocols || statusCode >= 200 {
		w.committed = true
	}
}

// FlushError flushes the response through the underlying response writer.
func (w *recoveryResponseWriter) FlushError() (err error) {
	w.committed = true

	return http.NewResponseController(w.ResponseWriter).Flush()
}

// Hijack implements the [http.Hijacker] interface.
func (w *recoveryResponseWriter) Hijack() (conn net.Conn, rw *bufio.ReadWriter, err error) {
	w.committed = true

	return http.NewResponseController(w.ResponseWriter).Hijack()
}

// preparePanicResponse removes response-specific headers while preserving
// headers set by outer middleware, such as CORS and transport-security headers.
func preparePanicResponse(h http.Header) {
	for _, name := range []string{
		httphdr.ContentDisposition,
		httphdr.ContentEncoding,
		httphdr.ContentLength,
		httphdr.ContentType,
		httphdr.ETag,
		httphdr.Expires,
		httphdr.LastModified,
		httphdr.Location,
		httphdr.RetryAfter,
		httphdr.SetCookie,
		httphdr.Trailer,
		httphdr.TransferEncoding,
		httphdr.WWWAuthenticate,
	} {
		h.Del(name)
	}

	for name := range h {
		if strings.HasPrefix(name, http.TrailerPrefix) {
			h.Del(name)
		}
	}

	h.Set(httphdr.CacheControl, "no-store")
}

// recoverPanics returns a handler that recovers from panics, logs them, and
// responds with a generic error.  l and h must not be nil.
func recoverPanics(l *slog.Logger, h http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &recoveryResponseWriter{ResponseWriter: w}

		defer func() {
			v := recover()
			if v == nil {
				return
			}

			if err, ok := v.(error); ok && err == http.ErrAbortHandler {
				panic(v)
			}

			slogutil.PrintRecovered(r.Context(), l, v)
			if rw.committed {
				panic(http.ErrAbortHandler)
			}

			preparePanicResponse(rw.Header())
			http.Error(
				rw,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
		}()

		h.ServeHTTP(rw, r)
	})
}

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
