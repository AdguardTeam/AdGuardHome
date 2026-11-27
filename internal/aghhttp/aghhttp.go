// Package aghhttp provides some common methods to work with HTTP.
package aghhttp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// RegisterFunc is the function that sets the handler to handle the URL for the
// method.
//
// TODO(e.burkov, a.garipov):  Get rid of it.
type RegisterFunc func(method, url string, handler http.HandlerFunc)

// OK writes "OK\n" to the response.  l and w must not be nil.
func OK(ctx context.Context, l *slog.Logger, w http.ResponseWriter) {
	if _, err := io.WriteString(w, "OK\n"); err != nil {
		l.WarnContext(ctx, "writing ok body", slogutil.KeyError, err)
	}
}

// ErrorAndLog writes a formatted HTTP error response and logs it at
// [slog.LevelError] level.  l, r, and w must not be nil.
func ErrorAndLog(
	ctx context.Context,
	l *slog.Logger,
	r *http.Request,
	w http.ResponseWriter,
	code int,
	format string,
	args ...any,
) {
	text := fmt.Sprintf(format, args...)
	l.WarnContext(
		ctx,
		"http error",
		"host", r.Host,
		"method", r.Method,
		"raddr", r.RemoteAddr,
		"request_uri", r.RequestURI,
		"status", code,
		slogutil.KeyError, text,
	)

	http.Error(w, text, code)
}

// UserAgent returns the ID of the service as a User-Agent string.  It can also
// be used as the value of the Server HTTP header.
func UserAgent() (ua string) {
	return fmt.Sprintf("AdGuardHome/%s", version.Version())
}
