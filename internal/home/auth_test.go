package home

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func TestAuth(t *testing.T) {
	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()
	fn := filepath.Join(dir, "sessions.db")

	users := []User{
		{Name: "name", PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2"},
	}
	a := InitAuth(fn, nil, 60)
	s := session{}

	user := User{Name: "name"}
	a.UserAdd(&user, "password")

	assert.Equal(t, checkSessionNotFound, a.checkSession("notfound"))
	a.RemoveSession("notfound")

	sess, err := getSession(&users[0])
	assert.Nil(t, err)
	sessStr := hex.EncodeToString(sess)

	now := time.Now().UTC().Unix()
	// check expiration
	s.expire = uint32(now)
	a.addSession(sess, &s)
	assert.Equal(t, checkSessionExpired, a.checkSession(sessStr))

	// add session with TTL = 2 sec
	s = session{}
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.addSession(sess, &s)
	assert.Equal(t, checkSessionOK, a.checkSession(sessStr))

	a.Close()

	// load saved session
	a = InitAuth(fn, users, 60)

	// the session is still alive
	assert.Equal(t, checkSessionOK, a.checkSession(sessStr))
	// reset our expiration time because checkSession() has just updated it
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.storeSession(sess, &s)
	a.Close()

	u := a.UserFind("name", "password")
	assert.NotEmpty(t, u.Name)

	time.Sleep(3 * time.Second)

	// load and remove expired sessions
	a = InitAuth(fn, users, 60)
	assert.Equal(t, checkSessionNotFound, a.checkSession(sessStr))

	a.Close()
}

// implements http.ResponseWriter
type testResponseWriter struct {
	hdr        http.Header
	statusCode int
}

func (w *testResponseWriter) Header() http.Header {
	return w.hdr
}

func (w *testResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func TestAuthHTTP(t *testing.T) {
	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()
	fn := filepath.Join(dir, "sessions.db")

	users := []User{
		{Name: "name", PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2"},
	}
	Context.auth = InitAuth(fn, users, 60)

	handlerCalled := false
	handler := func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	}
	handler2 := optionalAuth(handler)
	w := testResponseWriter{}
	w.hdr = make(http.Header)
	r := http.Request{}
	r.Header = make(http.Header)
	r.Method = http.MethodGet

	// get / - we're redirected to login page
	r.URL = &url.URL{Path: "/"}
	handlerCalled = false
	handler2(&w, &r)
	assert.Equal(t, http.StatusFound, w.statusCode)
	assert.NotEmpty(t, w.hdr.Get("Location"))
	assert.False(t, handlerCalled)

	// go to login page
	loginURL := w.hdr.Get("Location")
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)

	// perform login
	cookie, err := Context.auth.httpCookie(loginJSON{Name: "name", Password: "password"})
	assert.Nil(t, err)
	assert.NotEmpty(t, cookie)

	// get /
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.Header.Set("Cookie", cookie)
	r.URL = &url.URL{Path: "/"}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)
	r.Header.Del("Cookie")

	// get / with basic auth
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.URL = &url.URL{Path: "/"}
	r.SetBasicAuth("name", "password")
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)
	r.Header.Del("Authorization")

	// get login page with a valid cookie - we're redirected to /
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.Header.Set("Cookie", cookie)
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.NotEmpty(t, w.hdr.Get("Location"))
	assert.False(t, handlerCalled)
	r.Header.Del("Cookie")

	// get login page with an invalid cookie
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.Header.Set("Cookie", "bad")
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)
	r.Header.Del("Cookie")

	Context.auth.Close()
}
