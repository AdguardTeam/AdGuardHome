package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/textproto"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TODO(s.chzhen): !! Add more tests.
func TestAuth_ServeHTTP_first_run(t *testing.T) {
	storeGlobals(t)

	globalContext.firstRun = true

	mux := http.NewServeMux()
	globalContext.mux = mux

	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	web, err := initWeb(ctx, options{}, nil, nil, logger, nil, false)
	require.NoError(t, err)

	globalContext.web = web

	testCases := []struct {
		url    string
		method string
		code   int
	}{{
		url:    "/",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/apple/doh.mobileconfig",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/apple/dot.mobileconfig",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/i18n/change_language",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/i18n/current_language",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/install/check_config",
		method: http.MethodPost,
		code:   http.StatusBadRequest,
	}, {
		url:    "/control/install/configure",
		method: http.MethodPost,
		code:   http.StatusBadRequest,
	}, {
		url:    "/control/install/get_addresses",
		method: http.MethodGet,
		code:   http.StatusOK,
	}, {
		url:    "/control/login",
		method: http.MethodPost,
		code:   http.StatusFound,
	}, {
		url:    "/control/logout",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/profile",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/profile/update",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/status",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/update",
		method: http.MethodGet,
		code:   http.StatusFound,
	}, {
		url:    "/control/version.json",
		method: http.MethodGet,
		code:   http.StatusFound,
	}}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.url, nil)

			h, pattern := mux.Handler(r)
			require.NotEmpty(t, pattern)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			assert.Equal(t, tc.code, w.Code)
		})
	}
}

func TestAuth_ServeHTTP(t *testing.T) {
	storeGlobals(t)

	const (
		authNone = iota
		authBasic
		authCookie
	)

	const (
		testTTL      = 60
		userName     = "name"
		userPassword = "password"
	)

	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	sessionsDB := filepath.Join(t.TempDir(), "sessions.db")

	users := []webUser{{
		Name:         userName,
		PasswordHash: string(passwordHash),
	}}
	auth := InitAuth(sessionsDB, users, testTTL, nil, nil)
	globalContext.auth = auth

	mux := http.NewServeMux()
	globalContext.mux = mux

	tlsMgr, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:         logger,
		configModified: func() {},
	})
	require.NoError(t, err)

	web, err := initWeb(ctx, options{}, nil, nil, logger, tlsMgr, false)
	require.NoError(t, err)

	globalContext.web = web

	creds, err := json.Marshal(&loginJSON{Name: userName, Password: userPassword})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/control/login", bytes.NewReader(creds))
	r.Header.Set(httphdr.ContentType, aghhttp.HdrValApplicationJSON)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	var loginCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName {
			loginCookie = c
		}
	}
	require.NotNil(t, loginCookie)

	testCases := []struct {
		url        string
		method     string
		authMethod int
		wantCode   int
	}{{
		url:        "/",
		method:     http.MethodGet,
		authMethod: authNone,
		wantCode:   http.StatusFound,
	}, {
		url:        "/control/i18n/change_language",
		method:     http.MethodPost,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/i18n/change_language",
		method:     http.MethodPost,
		authMethod: authBasic,
		wantCode:   http.StatusInternalServerError,
	}, {
		url:        "/control/i18n/change_language",
		method:     http.MethodPost,
		authMethod: authCookie,
		wantCode:   http.StatusInternalServerError,
	}, {
		url:        "/control/i18n/current_language",
		method:     http.MethodGet,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/i18n/current_language",
		method:     http.MethodGet,
		authMethod: authBasic,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/i18n/current_language",
		method:     http.MethodGet,
		authMethod: authCookie,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/logout",
		method:     http.MethodGet,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/logout",
		method:     http.MethodGet,
		authMethod: authBasic,
		wantCode:   http.StatusFound,
	}, {
		url:        "/control/profile",
		method:     http.MethodGet,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/profile",
		method:     http.MethodGet,
		authMethod: authBasic,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/profile",
		method:     http.MethodGet,
		authMethod: authCookie,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/profile/update",
		method:     http.MethodPut,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/profile/update",
		method:     http.MethodPut,
		authMethod: authBasic,
		wantCode:   http.StatusBadRequest,
	}, {
		url:        "/control/profile/update",
		method:     http.MethodPut,
		authMethod: authCookie,
		wantCode:   http.StatusBadRequest,
	}, {
		url:        "/control/status",
		method:     http.MethodGet,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/status",
		method:     http.MethodGet,
		authMethod: authBasic,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/status",
		method:     http.MethodGet,
		authMethod: authCookie,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/update",
		method:     http.MethodPost,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/version.json",
		method:     http.MethodGet,
		authMethod: authNone,
		wantCode:   http.StatusForbidden,
	}, {
		url:        "/control/version.json",
		method:     http.MethodGet,
		authMethod: authBasic,
		wantCode:   http.StatusOK,
	}, {
		url:        "/control/version.json",
		method:     http.MethodGet,
		authMethod: authCookie,
		wantCode:   http.StatusOK,
	}}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			r = httptest.NewRequest(tc.method, tc.url, nil)
			switch tc.authMethod {
			case authNone:
				// Go on.
			case authBasic:
				r.SetBasicAuth(userName, userPassword)
			case authCookie:
				r.AddCookie(loginCookie)
			default:
				panic("unrecognized auth method")
			}

			h, pattern := mux.Handler(r)
			require.NotEmpty(t, pattern)

			w = httptest.NewRecorder()
			h.ServeHTTP(w, r)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}

	t.Run("logout", func(t *testing.T) {
		r = httptest.NewRequest(http.MethodGet, "/control/status", nil)
		r.AddCookie(loginCookie)
		w = httptest.NewRecorder()

		mux.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		r = httptest.NewRequest(http.MethodGet, "/control/logout", nil)
		r.AddCookie(loginCookie)
		w = httptest.NewRecorder()

		mux.ServeHTTP(w, r)
		assert.Equal(t, http.StatusFound, w.Code)

		r = httptest.NewRequest(http.MethodGet, "/control/status", nil)
		r.AddCookie(loginCookie)
		w = httptest.NewRecorder()

		mux.ServeHTTP(w, r)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
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
	globalContext.auth = InitAuth(fn, users, 60, nil, nil)

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
	cookie, err := globalContext.auth.newCookie(loginJSON{Name: "name", Password: "password"}, "")
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

	globalContext.auth.Close()
}

func TestRealIP(t *testing.T) {
	const remoteAddr = "1.2.3.4:5678"

	testCases := []struct {
		name       string
		header     http.Header
		remoteAddr string
		wantErrMsg string
		wantIP     netip.Addr
	}{{
		name:       "success_no_proxy",
		header:     nil,
		remoteAddr: remoteAddr,
		wantErrMsg: "",
		wantIP:     netip.MustParseAddr("1.2.3.4"),
	}, {
		name: "success_proxy",
		header: http.Header{
			textproto.CanonicalMIMEHeaderKey(httphdr.XRealIP): []string{"1.2.3.5"},
		},
		remoteAddr: remoteAddr,
		wantErrMsg: "",
		wantIP:     netip.MustParseAddr("1.2.3.5"),
	}, {
		name: "success_proxy_multiple",
		header: http.Header{
			textproto.CanonicalMIMEHeaderKey(httphdr.XForwardedFor): []string{
				"1.2.3.6, 1.2.3.5",
			},
		},
		remoteAddr: remoteAddr,
		wantErrMsg: "",
		wantIP:     netip.MustParseAddr("1.2.3.6"),
	}, {
		name:       "error_no_proxy",
		header:     nil,
		remoteAddr: "1:::2",
		wantErrMsg: `getting ip from client addr: address 1:::2: ` +
			`too many colons in address`,
		wantIP: netip.Addr{},
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
