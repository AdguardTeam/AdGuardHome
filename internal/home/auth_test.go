package home

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func TestAuth(t *testing.T) {
	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()
	fn := filepath.Join(dir, "sessions.db")

	users := []User{
		User{Name: "name", PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2"},
	}
	a := InitAuth(fn, nil, 60)
	s := session{}

	user := User{Name: "name"}
	a.UserAdd(&user, "password")

	assert.True(t, a.CheckSession("notfound") == -1)
	a.RemoveSession("notfound")

	sess := getSession(&users[0])
	sessStr := hex.EncodeToString(sess)

	now := time.Now().UTC().Unix()
	// check expiration
	s.expire = uint32(now)
	a.addSession(sess, &s)
	assert.True(t, a.CheckSession(sessStr) == 1)

	// add session with TTL = 2 sec
	s = session{}
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.addSession(sess, &s)
	assert.True(t, a.CheckSession(sessStr) == 0)

	a.Close()

	// load saved session
	a = InitAuth(fn, users, 60)

	// the session is still alive
	assert.True(t, a.CheckSession(sessStr) == 0)
	// reset our expiration time because CheckSession() has just updated it
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.storeSession(sess, &s)
	a.Close()

	u := a.UserFind("name", "password")
	assert.True(t, len(u.Name) != 0)

	time.Sleep(3 * time.Second)

	// load and remove expired sessions
	a = InitAuth(fn, users, 60)
	assert.True(t, a.CheckSession(sessStr) == -1)

	a.Close()
	os.Remove(fn)
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
		User{Name: "name", PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2"},
	}
	Context.auth = InitAuth(fn, users, 60)

	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}
	handler2 := optionalAuth(handler)
	w := testResponseWriter{}
	w.hdr = make(http.Header)
	r := http.Request{}
	r.Header = make(http.Header)
	r.Method = "GET"

	// get / - we're redirected to login page
	r.URL = &url.URL{Path: "/"}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, w.statusCode == http.StatusFound)
	assert.True(t, w.hdr.Get("Location") != "")
	assert.True(t, !handlerCalled)

	// go to login page
	loginURL := w.hdr.Get("Location")
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)

	// perform login
	cookie := Context.auth.httpCookie(loginJSON{Name: "name", Password: "password"})
	assert.True(t, cookie != "")

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
	assert.True(t, w.hdr.Get("Location") != "")
	assert.True(t, !handlerCalled)
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
