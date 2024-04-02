package home

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
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

// newCookie creates a new authentication cookie.
func (a *Auth) newCookie(req loginJSON, addr string) (c *http.Cookie, err error) {
	rateLimiter := a.rateLimiter
	u, ok := a.findUser(req.Name, req.Password)
	if !ok {
		if rateLimiter != nil {
			rateLimiter.inc(addr)
		}

		return nil, errors.Error("invalid username or password")
	}

	if rateLimiter != nil {
		rateLimiter.remove(addr)
	}

	sess, err := newSessionToken()
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	now := time.Now().UTC()

	a.addSession(sess, &session{
		userName: u.Name,
		expire:   uint32(now.Unix()) + a.sessionTTL,
	})

	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    hex.EncodeToString(sess),
		Path:     "/",
		Expires:  now.Add(cookieTTL),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}, nil
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
func handleLogin(w http.ResponseWriter, r *http.Request) {
	req := loginJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	var remoteIP string
	// realIP cannot be used here without taking TrustedProxies into account due
	// to security issues.
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

	if rateLimiter := Context.auth.rateLimiter; rateLimiter != nil {
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
		log.Error("auth: getting real ip from request with remote ip %s: %s", remoteIP, err)
	}

	cookie, err := Context.auth.newCookie(req, remoteIP)
	if err != nil {
		logIP := remoteIP
		if Context.auth.trustedProxies.Contains(ip.Unmap()) {
			logIP = ip.String()
		}

		writeErrorWithIP(r, w, http.StatusForbidden, logIP, "%s", err)

		return
	}

	log.Info("auth: user %q successfully logged in from ip %s", req.Name, ip)

	http.SetCookie(w, cookie)

	h := w.Header()
	h.Set(httphdr.CacheControl, "no-store, no-cache, must-revalidate, proxy-revalidate")
	h.Set(httphdr.Pragma, "no-cache")
	h.Set(httphdr.Expires, "0")

	aghhttp.OK(w)
}

// handleLogout is the handler for the GET /control/logout HTTP API.
func handleLogout(w http.ResponseWriter, r *http.Request) {
	respHdr := w.Header()
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		// The only error that is returned from r.Cookie is [http.ErrNoCookie].
		// The user is already logged out.
		respHdr.Set(httphdr.Location, "/login.html")
		w.WriteHeader(http.StatusFound)

		return
	}

	Context.auth.removeSession(c.Value)

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

// RegisterAuthHandlers - register handlers
func RegisterAuthHandlers() {
	Context.mux.Handle("/control/login", postInstallHandler(ensureHandler(http.MethodPost, handleLogin)))
	httpRegister(http.MethodGet, "/control/logout", handleLogout)
}

// optionalAuthThird returns true if a user should authenticate first.
func optionalAuthThird(w http.ResponseWriter, r *http.Request) (mustAuth bool) {
	pref := fmt.Sprintf("auth: raddr %s", r.RemoteAddr)

	if glProcessCookie(r) {
		log.Debug("%s: authentication is handled by gl-inet submodule", pref)

		return false
	}

	// redirect to login page if not authenticated
	isAuthenticated := false
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		// The only error that is returned from r.Cookie is [http.ErrNoCookie].
		// Check Basic authentication.
		user, pass, hasBasic := r.BasicAuth()
		if hasBasic {
			_, isAuthenticated = Context.auth.findUser(user, pass)
			if !isAuthenticated {
				log.Info("%s: invalid basic authorization value", pref)
			}
		}
	} else {
		res := Context.auth.checkSession(cookie.Value)
		isAuthenticated = res == checkSessionOK
		if !isAuthenticated {
			log.Debug("%s: invalid cookie value: %q", pref, cookie)
		}
	}

	if isAuthenticated {
		return false
	}

	if p := r.URL.Path; p == "/" || p == "/index.html" {
		if glProcessRedirect(w, r) {
			log.Debug("%s: redirected to login page by gl-inet submodule", pref)
		} else {
			log.Debug("%s: redirected to login page", pref)
			http.Redirect(w, r, "login.html", http.StatusFound)
		}
	} else {
		log.Debug("%s: responded with forbidden to %s %s", pref, r.Method, p)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Forbidden"))
	}

	return true
}

// TODO(a.garipov): Use [http.Handler] consistently everywhere throughout the
// project.
func optionalAuth(
	h func(http.ResponseWriter, *http.Request),
) (wrapped func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		authRequired := Context.auth != nil && Context.auth.authRequired()
		if p == "/login.html" {
			cookie, err := r.Cookie(sessionCookieName)
			if authRequired && err == nil {
				// Redirect to the dashboard if already authenticated.
				res := Context.auth.checkSession(cookie.Value)
				if res == checkSessionOK {
					http.Redirect(w, r, "", http.StatusFound)

					return
				}

				log.Debug("auth: raddr %s: invalid cookie value: %q", r.RemoteAddr, cookie)
			}
		} else if isPublicResource(p) {
			// Process as usual, no additional auth requirements.
		} else if authRequired {
			if optionalAuthThird(w, r) {
				return
			}
		}

		h(w, r)
	}
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

	return isAsset || isLogin
}

// authHandler is a helper structure that implements [http.Handler].
type authHandler struct {
	handler http.Handler
}

// ServeHTTP implements the [http.Handler] interface for *authHandler.
func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	optionalAuth(a.handler.ServeHTTP)(w, r)
}

// optionalAuthHandler returns a authentication handler.
func optionalAuthHandler(handler http.Handler) http.Handler {
	return &authHandler{handler}
}
