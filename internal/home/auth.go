package home

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieTTL         = 365 * 24 // in hours
	sessionCookieName = "agh_session"
)

type session struct {
	userName string
	expire   uint32 // expiration time (in seconds)
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
	db         *bbolt.DB
	sessions   map[string]*session
	users      []User
	lock       sync.Mutex
	sessionTTL uint32
}

// User object
type User struct {
	Name         string `yaml:"name"`
	PasswordHash string `yaml:"password"` // bcrypt hash
}

// InitAuth - create a global object
func InitAuth(dbFilename string, users []User, sessionTTL uint32) *Auth {
	log.Info("Initializing auth module: %s", dbFilename)

	a := Auth{}
	a.sessionTTL = sessionTTL
	a.sessions = make(map[string]*session)
	var err error
	a.db, err = bbolt.Open(dbFilename, 0o644, nil)
	if err != nil {
		log.Error("Auth: open DB: %s: %s", dbFilename, err)
		if err.Error() == "invalid argument" {
			log.Error("AdGuard Home cannot be initialized due to an incompatible file system.\nPlease read the explanation here: https://github.com/AdguardTeam/AdGuardHome/internal/wiki/Getting-Started#limitations")
		}
		return nil
	}
	a.loadSessions()
	a.users = users
	log.Info("Auth: initialized.  users:%d  sessions:%d", len(a.users), len(a.sessions))
	return &a
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
		log.Error("Auth: bbolt.Begin: %s", err)
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
				log.Error("Auth: bbolt.Delete: %s", err)
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
	log.Debug("Auth: loaded %d sessions from DB (removed %d expired)", len(a.sessions), removed)
}

// store session data in file
func (a *Auth) addSession(data []byte, s *session) {
	name := hex.EncodeToString(data)
	a.lock.Lock()
	a.sessions[name] = s
	a.lock.Unlock()
	if a.storeSession(data, s) {
		log.Debug("Auth: created session %s: expire=%d", name, s.expire)
	}
}

// store session data in file
func (a *Auth) storeSession(data []byte, s *session) bool {
	tx, err := a.db.Begin(true)
	if err != nil {
		log.Error("Auth: bbolt.Begin: %s", err)
		return false
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bkt, err := tx.CreateBucketIfNotExists(bucketName())
	if err != nil {
		log.Error("Auth: bbolt.CreateBucketIfNotExists: %s", err)
		return false
	}
	err = bkt.Put(data, s.serialize())
	if err != nil {
		log.Error("Auth: bbolt.Put: %s", err)
		return false
	}

	err = tx.Commit()
	if err != nil {
		log.Error("Auth: bbolt.Commit: %s", err)
		return false
	}
	return true
}

// remove session from file
func (a *Auth) removeSession(sess []byte) {
	tx, err := a.db.Begin(true)
	if err != nil {
		log.Error("Auth: bbolt.Begin: %s", err)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bkt := tx.Bucket(bucketName())
	if bkt == nil {
		log.Error("Auth: bbolt.Bucket")
		return
	}
	err = bkt.Delete(sess)
	if err != nil {
		log.Error("Auth: bbolt.Put: %s", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Error("Auth: bbolt.Commit: %s", err)
		return
	}

	log.Debug("Auth: removed session from DB")
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
			log.Debug("Auth: updated session %s: expire=%d", sess, s.expire)
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

func getSession(u *User) ([]byte, error) {
	maxSalt := big.NewInt(math.MaxUint32)
	salt, err := rand.Int(rand.Reader, maxSalt)
	if err != nil {
		return nil, err
	}

	d := []byte(fmt.Sprintf("%s%s%s", salt, u.Name, u.PasswordHash))
	hash := sha256.Sum256(d)
	return hash[:], nil
}

func (a *Auth) httpCookie(req loginJSON) (string, error) {
	u := a.UserFind(req.Name, req.Password)
	if len(u.Name) == 0 {
		return "", nil
	}

	sess, err := getSession(&u)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	expire := now.Add(cookieTTL * time.Hour)
	expstr := expire.Format(time.RFC1123)
	expstr = expstr[:len(expstr)-len("UTC")] // "UTC" -> "GMT"
	expstr += "GMT"

	s := session{}
	s.userName = u.Name
	s.expire = uint32(now.Unix()) + a.sessionTTL
	a.addSession(sess, &s)

	return fmt.Sprintf("%s=%s; Path=/; HttpOnly; Expires=%s",
		sessionCookieName, hex.EncodeToString(sess), expstr), nil
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	req := loginJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	cookie, err := Context.auth.httpCookie(req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "crypto rand reader: %s", err)
		return
	}
	if len(cookie) == 0 {
		log.Info("Auth: invalid user name or password: name=%q", req.Name)
		time.Sleep(1 * time.Second)
		http.Error(w, "invalid user name or password", http.StatusBadRequest)
		return
	}

	w.Header().Set("Set-Cookie", cookie)

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	returnOK(w)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := r.Header.Get("Cookie")
	sess := parseCookie(cookie)

	Context.auth.RemoveSession(sess)

	w.Header().Set("Location", "/login.html")

	s := fmt.Sprintf("%s=; Path=/; HttpOnly; Expires=Thu, 01 Jan 1970 00:00:00 GMT",
		sessionCookieName)
	w.Header().Set("Set-Cookie", s)

	w.WriteHeader(http.StatusFound)
}

// RegisterAuthHandlers - register handlers
func RegisterAuthHandlers() {
	Context.mux.Handle("/control/login", postInstallHandler(ensureHandler("POST", handleLogin)))
	httpRegister("GET", "/control/logout", handleLogout)
}

func parseCookie(cookie string) string {
	pairs := strings.Split(cookie, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if kv[0] == sessionCookieName {
			return kv[1]
		}
	}
	return ""
}

// optionalAuthThird return true if user should authenticate first.
func optionalAuthThird(w http.ResponseWriter, r *http.Request) (authFirst bool) {
	authFirst = false

	// redirect to login page if not authenticated
	ok := false
	cookie, err := r.Cookie(sessionCookieName)

	if glProcessCookie(r) {
		log.Debug("Auth: authentification was handled by GL-Inet submodule")
		ok = true

	} else if err == nil {
		r := Context.auth.checkSession(cookie.Value)
		if r == checkSessionOK {
			ok = true
		} else if r < 0 {
			log.Debug("Auth: invalid cookie value: %s", cookie)
		}
	} else {
		// there's no Cookie, check Basic authentication
		user, pass, ok2 := r.BasicAuth()
		if ok2 {
			u := Context.auth.UserFind(user, pass)
			if len(u.Name) != 0 {
				ok = true
			} else {
				log.Info("Auth: invalid Basic Authorization value")
			}
		}
	}
	if !ok {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			if glProcessRedirect(w, r) {
				log.Debug("Auth: redirected to login page by GL-Inet submodule")
			} else {
				w.Header().Set("Location", "/login.html")
				w.WriteHeader(http.StatusFound)
			}
		} else {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Forbidden"))
		}
		authFirst = true
	}
	return authFirst
}

func optionalAuth(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login.html" {
			// redirect to dashboard if already authenticated
			authRequired := Context.auth != nil && Context.auth.AuthRequired()
			cookie, err := r.Cookie(sessionCookieName)
			if authRequired && err == nil {
				r := Context.auth.checkSession(cookie.Value)
				if r == checkSessionOK {
					w.Header().Set("Location", "/")
					w.WriteHeader(http.StatusFound)

					return
				} else if r == checkSessionNotFound {
					log.Debug("Auth: invalid cookie value: %s", cookie)
				}
			}

		} else if strings.HasPrefix(r.URL.Path, "/assets/") ||
			strings.HasPrefix(r.URL.Path, "/login.") {
			// process as usual
			// no additional auth requirements
		} else if Context.auth != nil && Context.auth.AuthRequired() {
			if optionalAuthThird(w, r) {
				return
			}
		}

		handler(w, r)
	}
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
func (a *Auth) UserAdd(u *User, password string) {
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

	log.Debug("Auth: added user: %s", u.Name)
}

// UserFind - find a user
func (a *Auth) UserFind(login, password string) User {
	a.lock.Lock()
	defer a.lock.Unlock()
	for _, u := range a.users {
		if u.Name == login &&
			bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil {
			return u
		}
	}
	return User{}
}

// getCurrentUser returns the current user.  It returns an empty User if the
// user is not found.
func (a *Auth) getCurrentUser(r *http.Request) User {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		// There's no Cookie, check Basic authentication.
		user, pass, ok := r.BasicAuth()
		if ok {
			return Context.auth.UserFind(user, pass)
		}

		return User{}
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	s, ok := a.sessions[cookie.Value]
	if !ok {
		return User{}
	}

	for _, u := range a.users {
		if u.Name == s.userName {
			return u
		}
	}

	return User{}
}

// GetUsers - get users
func (a *Auth) GetUsers() []User {
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
