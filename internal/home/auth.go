package home

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

// cookieTTL is the time-to-live of the session cookie.
const cookieTTL = 365 * timeutil.Day

// sessionCookieName is the name of the session cookie.
const sessionCookieName = "agh_session"

// sessionTokenSize is the length of session token in bytes.
const sessionTokenSize = 16

type session struct {
	userName string
	// expire is the expiration time, in seconds.
	expire uint32
}

func (s *session) serialize() []byte {
	const (
		expireLen = 4
		nameLen   = 2
	)
	data := make([]byte, expireLen+nameLen+len(s.userName))
	binary.BigEndian.PutUint32(data[0:4], s.expire)
	binary.BigEndian.PutUint16(data[4:6], uint16(len(s.userName)))
	copy(data[6:], []byte(s.userName))
	return data
}

func (s *session) deserialize(data []byte) bool {
	if len(data) < 4+2 {
		return false
	}
	s.expire = binary.BigEndian.Uint32(data[0:4])
	nameLen := binary.BigEndian.Uint16(data[4:6])
	data = data[6:]

	if len(data) < int(nameLen) {
		return false
	}
	s.userName = string(data)
	return true
}

// Auth - global object
type Auth struct {
	db          *bbolt.DB
	raleLimiter *authRateLimiter
	sessions    map[string]*session
	users       []webUser
	lock        sync.Mutex
	sessionTTL  uint32
}

// webUser represents a user of the Web UI.
type webUser struct {
	Name         string `yaml:"name"`
	PasswordHash string `yaml:"password"`
}

// InitAuth - create a global object
func InitAuth(dbFilename string, users []webUser, sessionTTL uint32, rateLimiter *authRateLimiter) *Auth {
	log.Info("Initializing auth module: %s", dbFilename)

	a := &Auth{
		sessionTTL:  sessionTTL,
		raleLimiter: rateLimiter,
		sessions:    make(map[string]*session),
		users:       users,
	}
	var err error
	a.db, err = bbolt.Open(dbFilename, 0o644, nil)
	if err != nil {
		log.Error("auth: open DB: %s: %s", dbFilename, err)
		if err.Error() == "invalid argument" {
			log.Error("AdGuard Home cannot be initialized due to an incompatible file system.\nPlease read the explanation here: https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#limitations")
		}

		return nil
	}
	a.loadSessions()
	log.Info("auth: initialized.  users:%d  sessions:%d", len(a.users), len(a.sessions))

	return a
}

// Close - close module
func (a *Auth) Close() {
	_ = a.db.Close()
}

func bucketName() []byte {
	return []byte("sessions-2")
}

// load sessions from file, remove expired sessions
func (a *Auth) loadSessions() {
	tx, err := a.db.Begin(true)
	if err != nil {
		log.Error("auth: bbolt.Begin: %s", err)

		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bkt := tx.Bucket(bucketName())
	if bkt == nil {
		return
	}

	removed := 0

	if tx.Bucket([]byte("sessions")) != nil {
		_ = tx.DeleteBucket([]byte("sessions"))
		removed = 1
	}

	now := uint32(time.Now().UTC().Unix())
	forEach := func(k, v []byte) error {
		s := session{}
		if !s.deserialize(v) || s.expire <= now {
			err = bkt.Delete(k)
			if err != nil {
				log.Error("auth: bbolt.Delete: %s", err)
			} else {
				removed++
			}

			return nil
		}

		a.sessions[hex.EncodeToString(k)] = &s
		return nil
	}
	_ = bkt.ForEach(forEach)
	if removed != 0 {
		err = tx.Commit()
		if err != nil {
			log.Error("bolt.Commit(): %s", err)
		}
	}

	log.Debug("auth: loaded %d sessions from DB (removed %d expired)", len(a.sessions), removed)
}

// store session data in file
func (a *Auth) addSession(data []byte, s *session) {
	name := hex.EncodeToString(data)
	a.lock.Lock()
	a.sessions[name] = s
	a.lock.Unlock()
	if a.storeSession(data, s) {
		log.Debug("auth: created session %s: expire=%d", name, s.expire)
	}
}

// store session data in file
func (a *Auth) storeSession(data []byte, s *session) bool {
	tx, err := a.db.Begin(true)
	if err != nil {
		log.Error("auth: bbolt.Begin: %s", err)

		return false
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bkt, err := tx.CreateBucketIfNotExists(bucketName())
	if err != nil {
		log.Error("auth: bbolt.CreateBucketIfNotExists: %s", err)

		return false
	}

	err = bkt.Put(data, s.serialize())
	if err != nil {
		log.Error("auth: bbolt.Put: %s", err)

		return false
	}

	err = tx.Commit()
	if err != nil {
		log.Error("auth: bbolt.Commit: %s", err)

		return false
	}

	return true
}

// remove session from file
func (a *Auth) removeSession(sess []byte) {
	tx, err := a.db.Begin(true)
	if err != nil {
		log.Error("auth: bbolt.Begin: %s", err)

		return
	}

	defer func() {
		_ = tx.Rollback()
	}()

	bkt := tx.Bucket(bucketName())
	if bkt == nil {
		log.Error("auth: bbolt.Bucket")

		return
	}

	err = bkt.Delete(sess)
	if err != nil {
		log.Error("auth: bbolt.Put: %s", err)

		return
	}

	err = tx.Commit()
	if err != nil {
		log.Error("auth: bbolt.Commit: %s", err)

		return
	}

	log.Debug("auth: removed session from DB")
}

// checkSessionResult is the result of checking a session.
type checkSessionResult int

// checkSessionResult constants.
const (
	checkSessionOK       checkSessionResult = 0
	checkSessionNotFound checkSessionResult = -1
	checkSessionExpired  checkSessionResult = 1
)

// checkSession checks if the session is valid.
func (a *Auth) checkSession(sess string) (res checkSessionResult) {
	now := uint32(time.Now().UTC().Unix())
	update := false

	a.lock.Lock()
	defer a.lock.Unlock()

	s, ok := a.sessions[sess]
	if !ok {
		return checkSessionNotFound
	}

	if s.expire <= now {
		delete(a.sessions, sess)
		key, _ := hex.DecodeString(sess)
		a.removeSession(key)

		return checkSessionExpired
	}

	newExpire := now + a.sessionTTL
	if s.expire/(24*60*60) != newExpire/(24*60*60) {
		// update expiration time once a day
		update = true
		s.expire = newExpire
	}

	if update {
		key, _ := hex.DecodeString(sess)
		if a.storeSession(key, s) {
			log.Debug("auth: updated session %s: expire=%d", sess, s.expire)
		}
	}

	return checkSessionOK
}

// RemoveSession - remove session
func (a *Auth) RemoveSession(sess string) {
	key, _ := hex.DecodeString(sess)
	a.lock.Lock()
	delete(a.sessions, sess)
	a.lock.Unlock()
	a.removeSession(key)
}

type loginJSON struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// newSessionToken returns cryptographically secure randomly generated slice of
// bytes of sessionTokenSize length.
//
// TODO(e.burkov): Think about using byte array instead of byte slice.
func newSessionToken() (data []byte, err error) {
	randData := make([]byte, sessionTokenSize)

	_, err = rand.Read(randData)
	if err != nil {
		return nil, err
	}

	return randData, nil
}

// newCookie creates a new authentication cookie.
func (a *Auth) newCookie(req loginJSON, addr string) (c *http.Cookie, err error) {
	rateLimiter := a.raleLimiter
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
		Name:    sessionCookieName,
		Value:   hex.EncodeToString(sess),
		Path:    "/",
		Expires: now.Add(cookieTTL),

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
func realIP(r *http.Request) (ip net.IP, err error) {
	proxyHeaders := []string{
		httphdr.CFConnectingIP,
		httphdr.TrueClientIP,
		httphdr.XRealIP,
	}

	for _, h := range proxyHeaders {
		v := r.Header.Get(h)
		ip = net.ParseIP(v)
		if ip != nil {
			return ip, nil
		}
	}

	// If none of the above yielded any results, get the leftmost IP address
	// from the X-Forwarded-For header.
	s := r.Header.Get(httphdr.XForwardedFor)
	ipStrs := strings.SplitN(s, ", ", 2)
	ip = net.ParseIP(ipStrs[0])
	if ip != nil {
		return ip, nil
	}

	// When everything else fails, just return the remote address as understood
	// by the stdlib.
	ipStr, err := netutil.SplitHost(r.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("getting ip from client addr: %w", err)
	}

	return net.ParseIP(ipStr), nil
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
	//
	// TODO(e.burkov): Use realIP when the issue will be fixed.
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

	if rateLimiter := Context.auth.raleLimiter; rateLimiter != nil {
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

	cookie, err := Context.auth.newCookie(req, remoteIP)
	if err != nil {
		writeErrorWithIP(r, w, http.StatusForbidden, remoteIP, "%s", err)

		return
	}

	// Use realIP here, since this IP address is only used for logging.
	ip, err := realIP(r)
	if err != nil {
		log.Error("auth: getting real ip from request with remote ip %s: %s", remoteIP, err)
	}

	log.Info("auth: user %q successfully logged in from ip %v", req.Name, ip)

	http.SetCookie(w, cookie)

	h := w.Header()
	h.Set(httphdr.CacheControl, "no-store, no-cache, must-revalidate, proxy-revalidate")
	h.Set(httphdr.Pragma, "no-cache")
	h.Set(httphdr.Expires, "0")

	aghhttp.OK(w)
}

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

	Context.auth.RemoveSession(c.Value)

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

// optionalAuthThird return true if user should authenticate first.
func optionalAuthThird(w http.ResponseWriter, r *http.Request) (mustAuth bool) {
	if glProcessCookie(r) {
		log.Debug("auth: authentication is handled by GL-Inet submodule")

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
				log.Info("auth: invalid Basic Authorization value")
			}
		}
	} else {
		res := Context.auth.checkSession(cookie.Value)
		isAuthenticated = res == checkSessionOK
		if !isAuthenticated {
			log.Debug("auth: invalid cookie value: %s", cookie)
		}
	}

	if isAuthenticated {
		return false
	}

	if p := r.URL.Path; p == "/" || p == "/index.html" {
		if glProcessRedirect(w, r) {
			log.Debug("auth: redirected to login page by GL-Inet submodule")
		} else {
			log.Debug("auth: redirected to login page")
			http.Redirect(w, r, "login.html", http.StatusFound)
		}
	} else {
		log.Debug("auth: responded with forbidden to %s %s", r.Method, p)
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
		authRequired := Context.auth != nil && Context.auth.AuthRequired()
		if p == "/login.html" {
			cookie, err := r.Cookie(sessionCookieName)
			if authRequired && err == nil {
				// Redirect to the dashboard if already authenticated.
				res := Context.auth.checkSession(cookie.Value)
				if res == checkSessionOK {
					http.Redirect(w, r, "", http.StatusFound)

					return
				}

				log.Debug("auth: invalid cookie value: %s", cookie)
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

type authHandler struct {
	handler http.Handler
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	optionalAuth(a.handler.ServeHTTP)(w, r)
}

func optionalAuthHandler(handler http.Handler) http.Handler {
	return &authHandler{handler}
}

// UserAdd - add new user
func (a *Auth) UserAdd(u *webUser, password string) {
	if len(password) == 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("bcrypt.GenerateFromPassword: %s", err)
		return
	}
	u.PasswordHash = string(hash)

	a.lock.Lock()
	a.users = append(a.users, *u)
	a.lock.Unlock()

	log.Debug("auth: added user: %s", u.Name)
}

// findUser returns a user if there is one.
func (a *Auth) findUser(login, password string) (u webUser, ok bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	for _, u = range a.users {
		if u.Name == login &&
			bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil {
			return u, true
		}
	}

	return webUser{}, false
}

// getCurrentUser returns the current user.  It returns an empty User if the
// user is not found.
func (a *Auth) getCurrentUser(r *http.Request) (u webUser) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		// There's no Cookie, check Basic authentication.
		user, pass, ok := r.BasicAuth()
		if ok {
			u, _ = Context.auth.findUser(user, pass)

			return u
		}

		return webUser{}
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	s, ok := a.sessions[cookie.Value]
	if !ok {
		return webUser{}
	}

	for _, u = range a.users {
		if u.Name == s.userName {
			return u
		}
	}

	return webUser{}
}

// GetUsers - get users
func (a *Auth) GetUsers() []webUser {
	a.lock.Lock()
	users := a.users
	a.lock.Unlock()
	return users
}

// AuthRequired - if authentication is required
func (a *Auth) AuthRequired() bool {
	if GLMode {
		return true
	}

	a.lock.Lock()
	r := (len(a.users) != 0)
	a.lock.Unlock()
	return r
}
