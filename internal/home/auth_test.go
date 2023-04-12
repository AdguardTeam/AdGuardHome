package home

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionToken(t *testing.T) {
	// Successful case.
	token, err := newSessionToken()
	require.NoError(t, err)
	assert.Len(t, token, sessionTokenSize)

	// Break the rand.Reader.
	prevReader := rand.Reader
	t.Cleanup(func() { rand.Reader = prevReader })
	rand.Reader = &bytes.Buffer{}

	// Unsuccessful case.
	token, err = newSessionToken()
	require.Error(t, err)
	assert.Empty(t, token)
}

func TestAuth(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "sessions.db")

	users := []webUser{{
		Name:         "name",
		PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2",
	}}
	a := InitAuth(fn, nil, 60, nil)
	s := session{}

	user := webUser{Name: "name"}
	a.UserAdd(&user, "password")

	assert.Equal(t, checkSessionNotFound, a.checkSession("notfound"))
	a.RemoveSession("notfound")

	sess, err := newSessionToken()
	require.NoError(t, err)
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
	a = InitAuth(fn, users, 60, nil)

	// the session is still alive
	assert.Equal(t, checkSessionOK, a.checkSession(sessStr))
	// reset our expiration time because checkSession() has just updated it
	s.expire = uint32(time.Now().UTC().Unix() + 2)
	a.storeSession(sess, &s)
	a.Close()

	u, ok := a.findUser("name", "password")
	assert.True(t, ok)
	assert.NotEmpty(t, u.Name)

	time.Sleep(3 * time.Second)

	// load and remove expired sessions
	a = InitAuth(fn, users, 60, nil)
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
	dir := t.TempDir()
	fn := filepath.Join(dir, "sessions.db")

	users := []webUser{
		{Name: "name", PasswordHash: "$2y$05$..vyzAECIhJPfaQiOK17IukcQnqEgKJHy0iETyYqxn3YXJl8yZuo2"},
	}
	Context.auth = InitAuth(fn, users, 60, nil)

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
	assert.NotEmpty(t, w.hdr.Get(httphdr.Location))
	assert.False(t, handlerCalled)

	// go to login page
	loginURL := w.hdr.Get(httphdr.Location)
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)

	// perform login
	cookie, err := Context.auth.newCookie(loginJSON{Name: "name", Password: "password"}, "")
	require.NoError(t, err)
	require.NotNil(t, cookie)

	// get /
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.Header.Set(httphdr.Cookie, cookie.String())
	r.URL = &url.URL{Path: "/"}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)

	r.Header.Del(httphdr.Cookie)

	// get / with basic auth
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.URL = &url.URL{Path: "/"}
	r.SetBasicAuth("name", "password")
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)
	r.Header.Del(httphdr.Authorization)

	// get login page with a valid cookie - we're redirected to /
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.Header.Set(httphdr.Cookie, cookie.String())
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.NotEmpty(t, w.hdr.Get(httphdr.Location))
	assert.False(t, handlerCalled)
	r.Header.Del(httphdr.Cookie)

	// get login page with an invalid cookie
	handler2 = optionalAuth(handler)
	w.hdr = make(http.Header)
	r.Header.Set(httphdr.Cookie, "bad")
	r.URL = &url.URL{Path: loginURL}
	handlerCalled = false
	handler2(&w, &r)
	assert.True(t, handlerCalled)
	r.Header.Del(httphdr.Cookie)

	Context.auth.Close()
}

func TestRealIP(t *testing.T) {
	const remoteAddr = "1.2.3.4:5678"

	testCases := []struct {
		name       string
		header     http.Header
		remoteAddr string
		wantErrMsg string
		wantIP     net.IP
	}{{
		name:       "success_no_proxy",
		header:     nil,
		remoteAddr: remoteAddr,
		wantErrMsg: "",
		wantIP:     net.IPv4(1, 2, 3, 4),
	}, {
		name: "success_proxy",
		header: http.Header{
			textproto.CanonicalMIMEHeaderKey(httphdr.XRealIP): []string{"1.2.3.5"},
		},
		remoteAddr: remoteAddr,
		wantErrMsg: "",
		wantIP:     net.IPv4(1, 2, 3, 5),
	}, {
		name: "success_proxy_multiple",
		header: http.Header{
			textproto.CanonicalMIMEHeaderKey(httphdr.XForwardedFor): []string{
				"1.2.3.6, 1.2.3.5",
			},
		},
		remoteAddr: remoteAddr,
		wantErrMsg: "",
		wantIP:     net.IPv4(1, 2, 3, 6),
	}, {
		name:       "error_no_proxy",
		header:     nil,
		remoteAddr: "1:::2",
		wantErrMsg: `getting ip from client addr: address 1:::2: ` +
			`too many colons in address`,
		wantIP: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{
				Header:     tc.header,
				RemoteAddr: tc.remoteAddr,
			}

			ip, err := realIP(r)
			assert.Equal(t, tc.wantIP, ip)

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}
