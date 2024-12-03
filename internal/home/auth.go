package home

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

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

// Auth is the global authentication object.
type Auth struct {
	trustedProxies netutil.SubnetSet
	db             *bbolt.DB
	rateLimiter    *authRateLimiter
	sessions       map[string]*session
	users          []webUser
	lock           sync.Mutex
	sessionTTL     uint32
}

// webUser represents a user of the Web UI.
//
// TODO(s.chzhen):  Improve naming.
type webUser struct {
	Name         string `yaml:"name"`
	PasswordHash string `yaml:"password"`
}

// InitAuth initializes the global authentication object.
func InitAuth(
	dbFilename string,
	users []webUser,
	sessionTTL uint32,
	rateLimiter *authRateLimiter,
	trustedProxies netutil.SubnetSet,
) (a *Auth) {
	log.Info("Initializing auth module: %s", dbFilename)

	a = &Auth{
		sessionTTL:     sessionTTL,
		rateLimiter:    rateLimiter,
		sessions:       make(map[string]*session),
		users:          users,
		trustedProxies: trustedProxies,
	}
	var err error

	a.db, err = bbolt.Open(dbFilename, aghos.DefaultPermFile, nil)
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

// Close closes the authentication database.
func (a *Auth) Close() {
	_ = a.db.Close()
}

func bucketName() []byte {
	return []byte("sessions-2")
}

// loadSessions loads sessions from the database file and removes expired
// sessions.
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

// addSession adds a new session to the list of sessions and saves it in the
// database file.
func (a *Auth) addSession(data []byte, s *session) {
	name := hex.EncodeToString(data)
	a.lock.Lock()
	a.sessions[name] = s
	a.lock.Unlock()
	if a.storeSession(data, s) {
		log.Debug("auth: created session %s: expire=%d", name, s.expire)
	}
}

// storeSession saves a session in the database file.
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

// removeSessionFromFile removes a stored session from the DB file on disk.
func (a *Auth) removeSessionFromFile(sess []byte) {
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
		a.removeSessionFromFile(key)

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

// removeSession removes the session from the active sessions and the disk.
func (a *Auth) removeSession(sess string) {
	key, _ := hex.DecodeString(sess)
	a.lock.Lock()
	delete(a.sessions, sess)
	a.lock.Unlock()
	a.removeSessionFromFile(key)
}

// addUser adds a new user with the given password.
func (a *Auth) addUser(u *webUser, password string) (err error) {
	if len(password) == 0 {
		return errors.Error("empty password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generating hash: %w", err)
	}

	u.PasswordHash = string(hash)

	a.lock.Lock()
	defer a.lock.Unlock()

	a.users = append(a.users, *u)

	log.Debug("auth: added user with login %q", u.Name)

	return nil
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

// usersList returns a copy of a users list.
func (a *Auth) usersList() (users []webUser) {
	a.lock.Lock()
	defer a.lock.Unlock()

	users = make([]webUser, len(a.users))
	copy(users, a.users)

	return users
}

// authRequired returns true if a authentication is required.
func (a *Auth) authRequired() bool {
	if GLMode {
		return true
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	return len(a.users) != 0
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
