package home

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/timeutil"
)

// glFilePrefix is the prefix of the filepath where the authentication token is
// stored.  Note that it is variable so it can be edited in tests.
//
// TODO(s.chzhen):  Make it a constant.
var glFilePrefix = "/tmp/gl_token_"

const (
	// glTokenTimeout is the TTL (Time To Live) of the authentication token.
	glTokenTimeout = 3600 * time.Second

	// glCookieName is the name of the cookie that stores the authentication
	// token.
	glCookieName = "Admin-Token"
)

// MaxFileSize is a maximum file length in bytes.
const MaxFileSize = 1024 * 1024

// authMiddlewareGLiNetConfig is the configuration structure for the GLiNet
// authentication middleware.
type authMiddlewareGLiNetConfig struct {
	// logger is used for logging the operation of the middleware.  It must not
	// be nil.
	//
	// TODO(s.chzhen):  Use logger from the context.
	logger *slog.Logger

	// clock is used to get the current time.  It must not be nil.
	clock timeutil.Clock

	// tokenFilePrefix is the prefix of the filepath where the authentication
	// token is stored.  It must not be empty.
	tokenFilePrefix string

	// ttl is the TTL (Time To Live) of the authentication token.  It must be
	// greater than zero.
	ttl time.Duration

	// maxTokenSize is the maximum size of the file containing the
	// authentication token.  It must be greater than zero.
	maxTokenSize uint
}

// authMiddlewareGLiNet is the GLiNet authentication middleware.  It checks if
// the request is authenticated using a cookie.
type authMiddlewareGLiNet struct {
	logger          *slog.Logger
	clock           timeutil.Clock
	tokenFilePrefix string
	ttl             time.Duration
	maxTokenSize    uint
}

// newAuthMiddlewareGLiNet returns the new properly initialized
// *authMiddlewareGLiNet.
func newAuthMiddlewareGLiNet(c *authMiddlewareGLiNetConfig) (mw *authMiddlewareGLiNet) {
	return &authMiddlewareGLiNet{
		logger:          c.logger,
		clock:           c.clock,
		tokenFilePrefix: c.tokenFilePrefix,
		ttl:             c.ttl,
		maxTokenSize:    c.maxTokenSize,
	}
}

// type check
var _ httputil.Middleware = (*authMiddlewareGLiNet)(nil)

// Wrap implements the [httputil.Middleware] interface for
// *authMiddlewareGLiNet.
func (mw *authMiddlewareGLiNet) Wrap(h http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		path := r.URL.Path
		if isPublicResource(path) {
			h.ServeHTTP(w, r)

			return
		}

		if mw.isAuthenticated(ctx, r) {
			h.ServeHTTP(w, r)

			return
		}

		if path == "/" || path == "/index.html" {
			host := r.Host

			if h, _, err := net.SplitHostPort(r.Host); err == nil {
				host = h
			}

			u := &url.URL{
				Scheme: urlutil.SchemeHTTP,
				Host:   host,
			}

			http.Redirect(w, r, u.String(), http.StatusFound)

			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}

// isAuthenticated returns true if the request is authenticated using a cookie.
//
// TODO(s.chzhen):  Use the request's path.
func (mw *authMiddlewareGLiNet) isAuthenticated(ctx context.Context, r *http.Request) (ok bool) {
	c, err := r.Cookie(glCookieName)
	if err == http.ErrNoCookie {
		mw.logger.ErrorContext(ctx, "no authentication cookie", slogutil.KeyError, err)

		return false
	}

	return mw.checkToken(ctx, c.Value)
}

// checkToken verifies the validity of an authentication token.  It retrieves
// the time stored in a file named after the token and checks if the token has
// expired based on that time.
func (mw *authMiddlewareGLiNet) checkToken(ctx context.Context, token string) (ok bool) {
	tokenFile := mw.tokenFilePrefix + token
	tokenDate := mw.tokenDate(ctx, tokenFile)
	now := mw.clock.Now()
	if now.Before(tokenDate.Add(mw.ttl)) {
		return true
	}

	mw.logger.DebugContext(ctx, "authentication token has expired")

	return false
}

// tokenDate returns the time stored in the authentication token file.  If there
// is an error, it logs the error and returns the zero time.
func (mw *authMiddlewareGLiNet) tokenDate(ctx context.Context, tokenFile string) (t time.Time) {
	f, err := os.Open(tokenFile)
	if err != nil {
		mw.logger.ErrorContext(ctx, "opening token file", slogutil.KeyError, err)

		return time.Time{}
	}

	defer slogutil.CloseAndLog(ctx, mw.logger, f, slog.LevelError)

	// Create a 4-byte long buffer to store Unix time as a uint32, since GL.iNet
	// routers use it as part of an authentication mechanism.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/1853.
	data := make([]byte, 4)
	_, err = io.ReadFull(f, data)
	if err != nil {
		mw.logger.ErrorContext(ctx, "reading token file", slogutil.KeyError, err)

		return time.Time{}
	}

	return time.Unix(int64(binary.NativeEndian.Uint32(data)), 0)
}
