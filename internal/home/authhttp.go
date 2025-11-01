package home

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// cookieTTL is the time-to-live of the session cookie.
const cookieTTL = 365 * timeutil.Day

// sessionCookieName is the name of the session cookie.
const sessionCookieName = "agh_session"

// loginJSON is the JSON structure for authentication.
type loginJSON struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// realIP extracts the real IP address of the client from an HTTP request using
// the known HTTP headers.
//
// TODO(a.garipov): Currently, this is basically a copy of a similar function in
// module dnsproxy.  This should really become a part of module golibs and be
// replaced both here and there.  Or be replaced in both places by
// a well-maintained third-party module.
//
// TODO(a.garipov): Support header Forwarded from RFC 7329.
func realIP(r *http.Request) (ip netip.Addr, err error) {
	proxyHeaders := []string{
		httphdr.CFConnectingIP,
		httphdr.TrueClientIP,
		httphdr.XRealIP,
	}

	for _, h := range proxyHeaders {
		v := r.Header.Get(h)
		ip, err = netip.ParseAddr(v)
		if err == nil {
			return ip, nil
		}
	}

	// If none of the above yielded any results, get the leftmost IP address
	// from the X-Forwarded-For header.
	s := r.Header.Get(httphdr.XForwardedFor)
	ipStr, _, _ := strings.Cut(s, ",")
	ip, err = netip.ParseAddr(ipStr)
	if err == nil {
		return ip, nil
	}

	// When everything else fails, just return the remote address as understood
	// by the stdlib.
	ipStr, err = netutil.SplitHost(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting ip from client addr: %w", err)
	}

	return netip.ParseAddr(ipStr)
}

// writeErrorWithIP is like [aghhttp.Error], but includes the remote IP address
// when it writes to the log.
func writeErrorWithIP(
	r *http.Request,
	w http.ResponseWriter,
	code int,
	remoteIP string,
	format string,
	args ...any,
) {
	text := fmt.Sprintf(format, args...)
	log.Error("%s %s %s: from ip %s: %s", r.Method, r.Host, r.URL, remoteIP, text)
	http.Error(w, text, code)
}

// handleLogin is the handler for the POST /control/login HTTP API.
func (web *webAPI) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := loginJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	var remoteIP string
	// The real IP address of the client [realIP] cannot be used here without
	// taking trusted proxies into account due to security issues:
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2799.
	if remoteIP, err = netutil.SplitHost(r.RemoteAddr); err != nil {
		writeErrorWithIP(
			r,
			w,
			http.StatusBadRequest,
			r.RemoteAddr,
			"auth: getting remote address: %s",
			err,
		)

		return
	}

	if rateLimiter := web.auth.rateLimiter; rateLimiter != nil {
		if left := rateLimiter.check(remoteIP); left > 0 {
			w.Header().Set(httphdr.RetryAfter, strconv.Itoa(int(left.Seconds())))
			writeErrorWithIP(
				r,
				w,
				http.StatusTooManyRequests,
				remoteIP,
				"auth: blocked for %s",
				left,
			)

			return
		}
	}

	ip, err := realIP(r)
	if err != nil {
		web.logger.ErrorContext(
			ctx,
			"getting real ip",
			"remote_ip", remoteIP,
			slogutil.KeyError, err,
		)
	}

	cookie, err := newCookie(ctx, web.auth, req, remoteIP)
	if err != nil {
		logIP := remoteIP
		if web.auth.trustedProxies.Contains(ip.Unmap()) {
			logIP = ip.String()
		}

		writeErrorWithIP(r, w, http.StatusForbidden, logIP, "%s", err)

		return
	}

	web.logger.InfoContext(ctx, "successful login", "user", req.Name, "ip", ip)

	http.SetCookie(w, cookie)

	h := w.Header()
	h.Set(httphdr.CacheControl, "no-store, no-cache, must-revalidate, proxy-revalidate")
	h.Set(httphdr.Pragma, "no-cache")
	h.Set(httphdr.Expires, "0")

	aghhttp.OK(ctx, web.logger, w)
}

// newCookie creates a new authentication cookie.  rateLimiter must not be nil.
func newCookie(
	ctx context.Context,
	auth *auth,
	req loginJSON,
	addr string,
) (c *http.Cookie, err error) {
	user, err := auth.users.ByLogin(ctx, aghuser.Login(req.Name))
	if err != nil {
		// Should not happen.
		panic(err)
	}

	rateLimiter := auth.rateLimiter
	if user == nil {
		rateLimiter.inc(addr)

		return nil, errInvalidLogin
	}

	ok := user.Password.Authenticate(ctx, req.Password)
	if !ok {
		rateLimiter.inc(addr)

		return nil, errInvalidLogin
	}

	rateLimiter.remove(addr)

	sess, err := auth.sessions.New(ctx, user)
	if err != nil {
		return nil, err
	}

	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    hex.EncodeToString(sess.Token[:]),
		Path:     "/",
		Expires:  time.Now().Add(cookieTTL),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

// handleLogout is the handler for the GET /control/logout HTTP API.
func (web *webAPI) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	respHdr := w.Header()
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		// The only error that is returned from r.Cookie is [http.ErrNoCookie].
		// The user is already logged out.
		respHdr.Set(httphdr.Location, "/login.html")
		w.WriteHeader(http.StatusFound)

		return
	}

	t, err := sessionTokenFromHex(c.Value)
	if err != nil {
		web.logger.ErrorContext(ctx, "getting token", slogutil.KeyError, err)

		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	err = web.auth.sessions.DeleteByToken(ctx, t)
	if err != nil {
		web.logger.ErrorContext(ctx, "removing session by token", slogutil.KeyError, err)
	}

	c = &http.Cookie{
		Name:    sessionCookieName,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),

		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	respHdr.Set(httphdr.Location, "/login.html")
	respHdr.Set(httphdr.SetCookie, c.String())
	w.WriteHeader(http.StatusFound)
}

// registerAuthHandlers registers authentication handlers.
func (web *webAPI) registerAuthHandlers() {
	web.conf.mux.Handle(
		http.MethodPost+" "+"/control/login",
		web.postInstallHandler(http.HandlerFunc(web.handleLogin)),
	)
	web.httpReg.Register(http.MethodGet, "/control/logout", web.handleLogout)
}

// isPublicResource returns true if p is a path to a public resource.
func isPublicResource(p string) (ok bool) {
	isAsset, err := path.Match("/assets/*", p)
	if err != nil {
		// The only error that is returned from path.Match is
		// [path.ErrBadPattern].  This is a programmer error.
		panic(fmt.Errorf("bad asset pattern: %w", err))
	}

	isLogin, err := path.Match("/login.*", p)
	if err != nil {
		// Same as above.
		panic(fmt.Errorf("bad login pattern: %w", err))
	}

	// TODO(s.chzhen):  Implement a more strict version.
	if strings.HasPrefix(p, "/dns-query/") {
		return true
	}

	paths := []string{
		"/dns-query",
		"/control/login",
		"/apple/doh.mobileconfig",
		"/apple/dot.mobileconfig",
		"/control/install/get_addresses",
		"/control/install/check_config",
		"/control/install/configure",
		"/install.html",
	}

	return isAsset || isLogin || slices.Contains(paths, p)
}

const (
	// errInvalidLogin is returned when there is an invalid login attempt.
	errInvalidLogin errors.Error = "invalid username or password"
)

// authMiddlewareDefaultConfig is the configuration structure for the default
// authentication middleware.
type authMiddlewareDefaultConfig struct {
	// logger is used for logging the operation of the middleware.  It must not
	// be nil.
	logger *slog.Logger

	// rateLimiter manages the rate limiting for login attempts.
	rateLimiter loginRateLimiter

	// trustedProxies is a set of subnets considered as trusted.
	//
	// TODO(s.chzhen):  Use it not only to pass it to the middleware but also to
	// log the work of the rate limiter.
	trustedProxies netutil.SubnetSet

	// sessions contains web user sessions.  It must not be nil.
	sessions aghuser.SessionStorage

	// users contains web user information.  It must not be nil.
	users aghuser.DB
}

// authMiddlewareDefault is the default authentication middleware.  It searches
// for a web client using an authentication cookie or basic auth credentials and
// passes it with the context.
type authMiddlewareDefault struct {
	logger         *slog.Logger
	rateLimiter    loginRateLimiter
	trustedProxies netutil.SubnetSet
	sessions       aghuser.SessionStorage
	users          aghuser.DB
}

// newAuthMiddlewareDefault returns the new properly initialized
// *authMiddlewareDefault.
func newAuthMiddlewareDefault(c *authMiddlewareDefaultConfig) (mw *authMiddlewareDefault) {
	return &authMiddlewareDefault{
		logger:         c.logger,
		rateLimiter:    c.rateLimiter,
		trustedProxies: c.trustedProxies,
		sessions:       c.sessions,
		users:          c.users,
	}
}

// type check
var _ httputil.Middleware = (*authMiddlewareDefault)(nil)

// Wrap implements the [httputil.Middleware] interface for
// *authMiddlewareDefault.
func (mw *authMiddlewareDefault) Wrap(h http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !mw.needsAuthentication(ctx) {
			h.ServeHTTP(w, r)

			return
		}

		path := r.URL.Path
		if mw.handleAuthenticatedUser(ctx, w, r, h, path) {
			return
		}

		if mw.handlePublicAccess(w, r, h, path) {
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}

// handleAuthenticatedUser tries to get user from request and processes request
// if user was successfully authenticated.  Returns true if request was handled.
func (mw *authMiddlewareDefault) handleAuthenticatedUser(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	h http.Handler,
	path string,
) (ok bool) {
	u, err := mw.userFromRequest(ctx, r)
	if err != nil {
		mw.logger.ErrorContext(ctx, "retrieving user from request", slogutil.KeyError, err)
	}

	if u == nil {
		return false
	}

	if path == "/login.html" {
		http.Redirect(w, r, "/", http.StatusFound)

		return true
	}

	h.ServeHTTP(w, r.WithContext(withWebUser(ctx, u)))

	return true
}

// handlePublicAccess handles request if user is trying to access public or root
// pages.
func (mw *authMiddlewareDefault) handlePublicAccess(
	w http.ResponseWriter,
	r *http.Request,
	h http.Handler,
	path string,
) (ok bool) {
	if isPublicResource(path) {
		h.ServeHTTP(w, r)

		return true
	}

	if path == "/" || path == "/index.html" {
		http.Redirect(w, r, "login.html", http.StatusFound)

		return true
	}

	return false
}

// needsAuthentication returns true if there are stored web users and requests
// should be authenticated first.
func (mw *authMiddlewareDefault) needsAuthentication(ctx context.Context) (ok bool) {
	users, err := mw.users.All(ctx)
	if err != nil {
		// Should not happen.
		panic(err)
	}

	return len(users) != 0
}

// userFromRequest tries to retrieve a user based on the request.  r must not be
// nil.
func (mw *authMiddlewareDefault) userFromRequest(
	ctx context.Context,
	r *http.Request,
) (u *aghuser.User, err error) {
	defer func() { err = errors.Annotate(err, "getting user from request: %w") }()

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		return mw.userFromCookie(ctx, cookie.Value)
	}

	return mw.userFromRequestBasicAuth(ctx, r)
}

// userFromCookie tries to retrieve a user based on the provided cookie value.
func (mw *authMiddlewareDefault) userFromCookie(
	ctx context.Context,
	val string,
) (u *aghuser.User, err error) {
	t, err := sessionTokenFromHex(val)
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return nil, err
	}

	s, err := mw.sessions.FindByToken(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("searching session by token: %w", err)
	}

	if s == nil {
		return nil, nil
	}

	u, err = mw.users.ByLogin(ctx, s.UserLogin)
	if err != nil {
		return nil, fmt.Errorf("searching user by login %q: %w", s.UserLogin, err)
	}

	return u, nil
}

// sessionTokenFromHex converts a hexadecimal string into a session token.
func sessionTokenFromHex(val string) (token aghuser.SessionToken, err error) {
	sess, err := hex.DecodeString(val)
	if err != nil {
		return token, fmt.Errorf("decoding value: %w", err)
	}

	l := aghuser.SessionTokenLength

	err = validate.Equal("token length", l, len(sess))
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return token, err
	}

	return aghuser.SessionToken(sess), nil
}

// userFromRequestBasicAuth searches for a user using Basic Auth credentials.  r
// must not be nil.
func (mw *authMiddlewareDefault) userFromRequestBasicAuth(
	ctx context.Context,
	r *http.Request,
) (user *aghuser.User, err error) {
	login, pass, ok := r.BasicAuth()
	if !ok {
		return nil, nil
	}

	var remoteIP string
	// The real IP address of the client [realIP] cannot be used here without
	// taking trusted proxies into account due to security issues:
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2799.
	if remoteIP, err = netutil.SplitHost(r.RemoteAddr); err != nil {
		return nil, fmt.Errorf("getting remote address: %w", err)
	}

	rateLimiter := mw.rateLimiter
	if left := rateLimiter.check(remoteIP); left > 0 {
		return nil, fmt.Errorf("login attempt blocked for %s", left)
	}

	defer func() {
		if err != nil {
			rateLimiter.inc(remoteIP)

			return
		}

		rateLimiter.remove(remoteIP)
	}()

	user, _ = mw.users.ByLogin(ctx, aghuser.Login(login))
	if user == nil {
		return nil, errInvalidLogin
	}

	ok = user.Password.Authenticate(ctx, pass)
	if !ok {
		return nil, errInvalidLogin
	}

	return user, nil
}
