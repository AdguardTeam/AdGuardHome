// Package aghhttp provides some common methods to work with HTTP.
package aghhttp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// RegisterFunc is the function that sets the handler to handle the URL for the
// method.
//
// TODO(e.burkov, a.garipov):  Get rid of it.
type RegisterFunc func(method, url string, handler http.HandlerFunc)

// OK responds with word OK.
func OK(w http.ResponseWriter) {
	if _, err := io.WriteString(w, "OK\n"); err != nil {
		log.Error("couldn't write body: %s", err)
	}
}

// Error writes formatted message to w and also logs it.
//
// TODO(s.chzhen):  Remove it.
func Error(r *http.Request, w http.ResponseWriter, code int, format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	log.Error("%s %s %s: %s", r.Method, r.Host, r.URL, text)
	http.Error(w, text, code)
}

// ErrorAndLog writes formatted message to w and also logs it with the specified
// logging level.
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
	l.ErrorContext(
		ctx,
		"http error",
		"host", r.Host,
		"method", r.Method,
		"raddr", r.RemoteAddr,
		"request_uri", r.RequestURI,
		slogutil.KeyError, text,
	)

	http.Error(w, text, code)
}

// UserAgent returns the ID of the service as a User-Agent string.  It can also
// be used as the value of the Server HTTP header.
func UserAgent() (ua string) {
	return fmt.Sprintf("AdGuardHome/%s", version.Version())
}

// textPlainDeprMsg is the message returned to API users when they try to use
// an API that used to accept "text/plain" but doesn't anymore.
const textPlainDeprMsg = `using this api with the text/plain content-type is deprecated; ` +
	`use application/json`

// WriteTextPlainDeprecated responds to the request with a message about
// deprecation and removal of a plain-text API if the request is made with the
// "text/plain" content-type.
func WriteTextPlainDeprecated(w http.ResponseWriter, r *http.Request) (isPlainText bool) {
	if r.Header.Get(httphdr.ContentType) != HdrValTextPlain {
		return false
	}

	Error(r, w, http.StatusUnsupportedMediaType, textPlainDeprMsg)

	return true
}
