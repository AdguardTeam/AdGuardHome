package home

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

const cookieTTL = 365 * 24 // in hours
const expireTime = 30 * 24 // in hours

// Auth - global object
type Auth struct {
	db       *bbolt.DB
	sessions map[string]uint32 // session -> expiration time (in seconds)
	lock     sync.Mutex
	users    []User
}

// User object
type User struct {
	Name         string `yaml:"name"`
	PasswordHash string `yaml:"password"` // bcrypt hash
}

// InitAuth - create a global object
func InitAuth(dbFilename string, users []User) *Auth {
	a := Auth{}
	a.sessions = make(map[string]uint32)
	rand.Seed(time.Now().UTC().Unix())
	var err error
	a.db, err = bbolt.Open(dbFilename, 0644, nil)
	if err != nil {
		log.Error("Auth: bbolt.Open: %s", err)
		return nil
	}
	a.loadSessions()
	a.users = users
	log.Debug("Auth: initialized.  users:%d  sessions:%d", len(a.users), len(a.sessions))
	return &a
}

// Close - close module
func (a *Auth) Close() {
	_ = a.db.Close()
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

	bkt := tx.Bucket([]byte("sessions"))
	if bkt == nil {
		return
	}

	removed := 0
	now := uint32(time.Now().UTC().Unix())
	forEach := func(k, v []byte) error {
		i := binary.BigEndian.Uint32(v)
		if i <= now {
			err = bkt.Delete(k)
			if err != nil {
				log.Error("Auth: bbolt.Delete: %s", err)
			} else {
				removed++
			}
			return nil
		}
		a.sessions[hex.EncodeToString(k)] = i
		return nil
	}
	_ = bkt.ForEach(forEach)
	if removed != 0 {
		_ = tx.Commit()
	}
	log.Debug("Auth: loaded %d sessions from DB (removed %d expired)", len(a.sessions), removed)
}

// store session data in file
func (a *Auth) storeSession(data []byte, expire uint32) {
	a.lock.Lock()
	a.sessions[hex.EncodeToString(data)] = expire
	a.lock.Unlock()

	tx, err := a.db.Begin(true)
	if err != nil {
		log.Error("Auth: bbolt.Begin: %s", err)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bkt, err := tx.CreateBucketIfNotExists([]byte("sessions"))
	if err != nil {
		log.Error("Auth: bbolt.CreateBucketIfNotExists: %s", err)
		return
	}
	var val []byte
	val = make([]byte, 4)
	binary.BigEndian.PutUint32(val, expire)
	err = bkt.Put(data, val)
	if err != nil {
		log.Error("Auth: bbolt.Put: %s", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Error("Auth: bbolt.Commit: %s", err)
		return
	}

	log.Debug("Auth: stored session in DB")
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

	bkt := tx.Bucket([]byte("sessions"))
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

// CheckSession - check if session is valid
// Return 0 if OK;  -1 if session doesn't exist;  1 if session has expired
func (a *Auth) CheckSession(sess string) int {
	now := uint32(time.Now().UTC().Unix())
	update := false

	a.lock.Lock()
	expire, ok := a.sessions[sess]
	if !ok {
		a.lock.Unlock()
		return -1
	}
	if expire <= now {
		delete(a.sessions, sess)
		key, _ := hex.DecodeString(sess)
		a.removeSession(key)
		a.lock.Unlock()
		return 1
	}

	newExpire := now + expireTime*60*60
	if expire/(24*60*60) != newExpire/(24*60*60) {
		// update expiration time once a day
		update = true
		a.sessions[sess] = newExpire
	}

	a.lock.Unlock()

	if update {
		key, _ := hex.DecodeString(sess)
		a.storeSession(key, expire)
	}

	return 0
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

func getSession(u *User) []byte {
	d := []byte(fmt.Sprintf("%d%s%s", rand.Uint32(), u.Name, u.PasswordHash))
	hash := sha256.Sum256(d)
	return hash[:]
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	req := loginJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	u := config.auth.UserFind(req.Name, req.Password)
	if len(u.Name) == 0 {
		time.Sleep(1 * time.Second)
		httpError(w, http.StatusBadRequest, "invalid login or password")
		return
	}

	sess := getSession(&u)

	now := time.Now().UTC()
	expire := now.Add(cookieTTL * time.Hour)
	expstr := expire.Format(time.RFC1123)
	expstr = expstr[:len(expstr)-len("UTC")] // "UTC" -> "GMT"
	expstr += "GMT"

	expireSess := uint32(now.Unix()) + expireTime*60*60
	config.auth.storeSession(sess, expireSess)

	s := fmt.Sprintf("session=%s; Path=/; HttpOnly; Expires=%s", hex.EncodeToString(sess), expstr)
	w.Header().Set("Set-Cookie", s)

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	returnOK(w)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := r.Header.Get("Cookie")
	sess := parseCookie(cookie)

	config.auth.RemoveSession(sess)

	w.Header().Set("Location", "/login.html")

	s := fmt.Sprintf("session=; Path=/; HttpOnly; Expires=Thu, 01 Jan 1970 00:00:00 GMT")
	w.Header().Set("Set-Cookie", s)

	w.WriteHeader(http.StatusFound)
}

// RegisterAuthHandlers - register handlers
func RegisterAuthHandlers() {
	http.Handle("/control/login", postInstallHandler(ensureHandler("POST", handleLogin)))
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
		if kv[0] == "session" {
			return kv[1]
		}
	}
	return ""
}

func optionalAuth(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/login.html" {
			// redirect to dashboard if already authenticated
			authRequired := config.auth != nil && config.auth.AuthRequired()
			cookie, err := r.Cookie("session")
			if authRequired && err == nil {
				r := config.auth.CheckSession(cookie.Value)
				if r == 0 {
					w.Header().Set("Location", "/")
					w.WriteHeader(http.StatusFound)
					return
				} else if r < 0 {
					log.Debug("Auth: invalid cookie value: %s", cookie)
				}
			}

		} else if r.URL.Path == "/favicon.png" ||
			strings.HasPrefix(r.URL.Path, "/login.") {
			// process as usual

		} else if config.auth != nil && config.auth.AuthRequired() {
			// redirect to login page if not authenticated
			ok := false
			cookie, err := r.Cookie("session")
			if err == nil {
				r := config.auth.CheckSession(cookie.Value)
				if r == 0 {

					ok = true
				} else if r < 0 {
					log.Debug("Auth: invalid cookie value: %s", cookie)
				}
			} else {
				// there's no Cookie, check Basic authentication
				user, pass, ok2 := r.BasicAuth()
				if ok2 {
					u := config.auth.UserFind(user, pass)
					if len(u.Name) != 0 {
						ok = true
					}
				}
			}
			if !ok {
				w.Header().Set("Location", "/login.html")
				w.WriteHeader(http.StatusFound)
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
func (a *Auth) UserFind(login string, password string) User {
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

// GetUsers - get users
func (a *Auth) GetUsers() []User {
	a.lock.Lock()
	users := a.users
	a.lock.Unlock()
	return users
}

// AuthRequired - if authentication is required
func (a *Auth) AuthRequired() bool {
	a.lock.Lock()
	r := (len(a.users) != 0)
	a.lock.Unlock()
	return r
}
