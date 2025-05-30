package home

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/josharian/native"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// testSessionStorage is the mock implementation of the [aghuser.SessionStorage]
// interface.
type testSessionStorage struct {
	onNew         func(ctx context.Context, u *aghuser.User) (s *aghuser.Session, err error)
	onFindByToken func(
		ctx context.Context,
		t aghuser.SessionToken,
	) (s *aghuser.Session, err error)
	onDeleteByToken func(ctx context.Context, t aghuser.SessionToken) (err error)
	onClose         func() (err error)
}

// type check
var _ aghuser.SessionStorage = (*testSessionStorage)(nil)

// newTestSessionStorage returns a new *testSessionStorage all methods of which
// panic.
func newTestSessionStorage() (ts *testSessionStorage) {
	return &testSessionStorage{
		onNew: func(_ context.Context, u *aghuser.User) (_ *aghuser.Session, _ error) {
			panic(fmt.Errorf("unexpected call to testSessionStorage.New(%v)", u))
		},
		onFindByToken: func(
			_ context.Context,
			t aghuser.SessionToken,
		) (_ *aghuser.Session, err error) {
			panic(fmt.Errorf("unexpected call to testSessionStorage.FindByToken(%v)", t))
		},
		onDeleteByToken: func(_ context.Context, t aghuser.SessionToken) (_ error) {
			panic(fmt.Errorf("unexpected call to testSessionStorage.DeleteByToken(%v)", t))
		},
		onClose: func() (_ error) {
			panic("unexpected call to testSessionStorage.Close")
		},
	}
}

// New implements the [aghuser.SessionStorage] interface for
// *testSessionStorage.
func (ts *testSessionStorage) New(
	ctx context.Context,
	u *aghuser.User,
) (s *aghuser.Session, err error) {
	return ts.onNew(ctx, u)
}

// FindByToken implements the [aghuser.SessionStorage] interface for
// *testSessionStorage.
func (ts *testSessionStorage) FindByToken(
	ctx context.Context,
	t aghuser.SessionToken,
) (s *aghuser.Session, err error) {
	return ts.onFindByToken(ctx, t)
}

// DeleteByToken implements the [aghuser.SessionStorage] interface for
// *testSessionStorage.
func (ts *testSessionStorage) DeleteByToken(
	ctx context.Context,
	t aghuser.SessionToken,
) (err error) {
	return ts.onDeleteByToken(ctx, t)
}

// Close implements the [aghuser.SessionStorage] interface for
// *testSessionStorage.
func (ts *testSessionStorage) Close() (err error) {
	return ts.onClose()
}

// testUsersDB is the mock implementation of the [aghuser.DB] interface.
type testUsersDB struct {
	onAll     func(ctx context.Context) (users []*aghuser.User, err error)
	onByLogin func(ctx context.Context, login aghuser.Login) (u *aghuser.User, err error)
	onByUUID  func(ctx context.Context, id aghuser.UserID) (u *aghuser.User, err error)
	onCreate  func(ctx context.Context, u *aghuser.User) (err error)
}

// newTestUsersDB returns a new *testUsersDB all methods of which panic.
func newTestUsersDB() (ts *testUsersDB) {
	return &testUsersDB{
		onAll: func(_ context.Context) (_ []*aghuser.User, _ error) {
			panic("unexpected call to testUsersDB.All")
		},
		onByLogin: func(_ context.Context, l aghuser.Login) (_ *aghuser.User, _ error) {
			panic(fmt.Errorf("unexpected call to testUsersDB.ByLogin(%v)", l))
		},
		onByUUID: func(_ context.Context, id aghuser.UserID) (_ *aghuser.User, _ error) {
			panic(fmt.Errorf("unexpected call to testUsersDB.ByUUID(%v)", id))
		},
		onCreate: func(_ context.Context, u *aghuser.User) (_ error) {
			panic(fmt.Errorf("unexpected call to testUsersDB.Create(%v)", u))
		},
	}
}

// type check
var _ aghuser.DB = (*testUsersDB)(nil)

// All implements the [aghuser.DB] interface for *testUsersDB.
func (db *testUsersDB) All(ctx context.Context) (users []*aghuser.User, err error) {
	return db.onAll(ctx)
}

// ByLogin implements the [aghuser.DB] interface for *testUsersDB.
func (db *testUsersDB) ByLogin(
	ctx context.Context,
	login aghuser.Login,
) (u *aghuser.User, err error) {
	return db.onByLogin(ctx, login)
}

// ByUUID implements the [aghuser.DB] interface for *testUsersDB.
func (db *testUsersDB) ByUUID(ctx context.Context, id aghuser.UserID) (u *aghuser.User, err error) {
	return db.onByUUID(ctx, id)
}

// Create implements the [aghuser.DB] interface for *testUsersDB.
func (db *testUsersDB) Create(ctx context.Context, u *aghuser.User) (err error) {
	return db.onCreate(ctx, u)
}

// testAuthHandler is a helper handler used for testing HTTP middleware.
type testAuthHandler struct {
	user   *aghuser.User
	called bool
}

// type check
var _ http.Handler = (*testAuthHandler)(nil)

// ServeHTTP implements the [http.Handler] interface for *testAuthHandler.
func (h *testAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	h.user, _ = webUserFromContext(r.Context())
}

func TestAuthMiddlewareDefault_firstRun(t *testing.T) {
	db := newTestUsersDB()
	db.onAll = func(_ context.Context) (users []*aghuser.User, err error) {
		return nil, nil
	}

	mw := newAuthMiddlewareDefault(&authMiddlewareDefaultConfig{
		logger:   testLogger,
		sessions: &testSessionStorage{},
		users:    db,
	})

	h := &testAuthHandler{}
	wrapped := mw.wrap(h)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, h.called)
}

func TestAuthMiddlewareDefault(t *testing.T) {
	t.Parallel()

	const (
		login aghuser.Login = "user_login"

		passwordRaw = "user_password"
	)

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(passwordRaw),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)

	user := &aghuser.User{
		Login:    login,
		Password: aghuser.NewDefaultPassword(string(passwordHash)),
	}

	var token aghuser.SessionToken
	_, _ = rand.Read(token[:])

	tokenHex := hex.EncodeToString(token[:])

	users := map[aghuser.Login]*aghuser.User{login: user}
	usersDB := newTestUsersDB()
	usersDB.onAll = func(_ context.Context) (us []*aghuser.User, err error) {
		return slices.Collect(maps.Values(users)), nil
	}

	usersDB.onByLogin = func(_ context.Context, login aghuser.Login) (u *aghuser.User, err error) {
		return users[login], nil
	}

	sessions := map[aghuser.SessionToken]*aghuser.Session{
		token: {
			UserLogin: login,
		},
	}
	ts := newTestSessionStorage()
	ts.onFindByToken = func(
		_ context.Context,
		t aghuser.SessionToken,
	) (s *aghuser.Session, err error) {
		return sessions[t], nil
	}

	mw := newAuthMiddlewareDefault(&authMiddlewareDefaultConfig{
		logger:   testLogger,
		sessions: ts,
		users:    usersDB,
	})

	reqCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	reqCookie.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tokenHex})

	reqInvalidCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	reqInvalidCookie.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "invalid_cookie"})

	reqBasicAuth := httptest.NewRequest(http.MethodGet, "/", nil)
	reqBasicAuth.SetBasicAuth(string(login), passwordRaw)

	reqInvalidPassBasicAuth := httptest.NewRequest(http.MethodGet, "/", nil)
	reqInvalidPassBasicAuth.SetBasicAuth(string(login), "invalid_password")

	testCases := []struct {
		req      *http.Request
		wantUser *aghuser.User
		name     string
		wantCode int
	}{{
		req:      httptest.NewRequest(http.MethodGet, "/", nil),
		wantUser: nil,
		name:     "no_auth",
		wantCode: http.StatusUnauthorized,
	}, {
		req:      reqCookie,
		wantUser: user,
		name:     "cookie",
		wantCode: http.StatusOK,
	}, {
		req:      reqBasicAuth,
		wantUser: user,
		name:     "basic_auth",
		wantCode: http.StatusOK,
	}, {
		req:      reqInvalidCookie,
		wantUser: nil,
		name:     "invalid_cookie",
		wantCode: http.StatusUnauthorized,
	}, {
		req:      reqInvalidPassBasicAuth,
		wantUser: nil,
		name:     "invalid_basic_auth",
		wantCode: http.StatusUnauthorized,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := &testAuthHandler{}
			wrapped := mw.wrap(h)

			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, tc.req)

			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, tc.wantUser, h.user)
		})
	}
}

func TestAuth_ServeHTTP_firstRun(t *testing.T) {
	storeGlobals(t)

	globalContext.firstRun = true

	mux := http.NewServeMux()
	globalContext.mux = mux

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	web, err := initWeb(ctx, options{}, nil, nil, testLogger, nil, false)
	require.NoError(t, err)

	globalContext.web = web

	testCases := []struct {
		name     string
		path     string
		method   string
		wantCode int
	}{{
		name:     "root",
		path:     "/",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "doh_mobileconfig",
		path:     "/apple/doh.mobileconfig",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "dot_mobileconfig",
		path:     "/apple/dot.mobileconfig",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "change_language",
		path:     "/control/i18n/change_language",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "current_language",
		path:     "/control/i18n/current_language",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "check_config",
		path:     "/control/install/check_config",
		method:   http.MethodPost,
		wantCode: http.StatusBadRequest,
	}, {
		name:     "configure",
		path:     "/control/install/configure",
		method:   http.MethodPost,
		wantCode: http.StatusBadRequest,
	}, {
		name:     "get_addresses",
		path:     "/control/install/get_addresses",
		method:   http.MethodGet,
		wantCode: http.StatusOK,
	}, {
		name:     "login",
		path:     "/control/login",
		method:   http.MethodPost,
		wantCode: http.StatusFound,
	}, {
		name:     "logout",
		path:     "/control/logout",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "profile",
		path:     "/control/profile",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "profile_update",
		path:     "/control/profile/update",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "status",
		path:     "/control/status",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "update",
		path:     "/control/update",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}, {
		name:     "version",
		path:     "/control/version.json",
		method:   http.MethodGet,
		wantCode: http.StatusFound,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)

			h, pattern := mux.Handler(r)
			require.NotEmpty(t, pattern)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}

func TestAuth_ServeHTTP_auth(t *testing.T) {
	storeGlobals(t)

	const (
		testTTL = 60

		glTokenFileSuffix = "test"

		userName     = "name"
		userPassword = "password"
	)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	tempDir := t.TempDir()
	glFilePrefix = tempDir + "/gl_token_"
	glTokenFile := glFilePrefix + glTokenFileSuffix

	glFileData := make([]byte, 4)
	native.Endian.PutUint32(glFileData, uint32(time.Now().Unix()+testTTL))

	err = os.WriteFile(glTokenFile, glFileData, 0o644)
	require.NoError(t, err)

	sessionsDB := filepath.Join(tempDir, "sessions.db")

	users := []webUser{{
		Name:         userName,
		PasswordHash: string(passwordHash),
	}}
	auth := InitAuth(sessionsDB, users, testTTL, nil, nil)
	t.Cleanup(auth.Close)
	globalContext.auth = auth

	mux := http.NewServeMux()
	globalContext.mux = mux

	tlsMgr, err := newTLSManager(testutil.ContextWithTimeout(t, testTimeout), &tlsManagerConfig{
		logger:         testLogger,
		configModified: func() {},
	})
	require.NoError(t, err)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	web, err := initWeb(ctx, options{}, nil, nil, testLogger, tlsMgr, false)
	require.NoError(t, err)

	globalContext.web = web

	loginCookie := generateAuthCookie(t, mux, userName, userPassword)

	testCases := []struct {
		name     string
		path     string
		method   string
		wantCode int
	}{{
		name:     "change_language",
		path:     "/control/i18n/change_language",
		method:   http.MethodPost,
		wantCode: http.StatusInternalServerError,
	}, {
		name:     "current_language",
		path:     "/control/i18n/current_language",
		method:   http.MethodGet,
		wantCode: http.StatusOK,
	}, {
		name:     "profile",
		path:     "/control/profile",
		method:   http.MethodGet,
		wantCode: http.StatusOK,
	}, {
		name:     "profile_update",
		path:     "/control/profile/update",
		method:   http.MethodPut,
		wantCode: http.StatusBadRequest,
	}, {
		name:     "status",
		path:     "/control/status",
		method:   http.MethodGet,
		wantCode: http.StatusOK,
	}, {
		name:     "version",
		path:     "/control/version.json",
		method:   http.MethodGet,
		wantCode: http.StatusOK,
	}}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			assertHandlerStatusCode(t, mux, r, http.StatusForbidden)

			r = httptest.NewRequest(tc.method, tc.path, nil)
			r.SetBasicAuth(userName, userPassword)
			assertHandlerStatusCode(t, mux, r, tc.wantCode)

			r = httptest.NewRequest(tc.method, tc.path, nil)
			r.AddCookie(loginCookie)
			assertHandlerStatusCode(t, mux, r, tc.wantCode)

			GLMode = true
			t.Cleanup(func() { GLMode = false })

			r.AddCookie(&http.Cookie{Name: glCookieName, Value: "test"})
			assertHandlerStatusCode(t, mux, r, tc.wantCode)
		})
	}
}

// generateAuthCookie is a helper function that logs in with the provided
// credentials and returns the resulting authentication cookie.
func generateAuthCookie(t *testing.T, mux *http.ServeMux, name, password string) (ac *http.Cookie) {
	t.Helper()

	creds, err := json.Marshal(&loginJSON{Name: name, Password: password})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/control/login", bytes.NewReader(creds))
	r.Header.Set(httphdr.ContentType, aghhttp.HdrValApplicationJSON)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName {
			return c
		}
	}

	return nil
}

// assertHandlerStatusCode is a helper function that asserts the response status
// code of a HTTP handler.
func assertHandlerStatusCode(t *testing.T, h http.Handler, r *http.Request, wantCode int) {
	t.Helper()

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, wantCode, w.Code)
}

func TestAuth_ServeHTTP_logout(t *testing.T) {
	storeGlobals(t)

	const (
		testTTL = 60

		userName     = "name"
		userPassword = "password"
	)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	sessionsDB := filepath.Join(t.TempDir(), "sessions.db")

	users := []webUser{{
		Name:         userName,
		PasswordHash: string(passwordHash),
	}}
	auth := InitAuth(sessionsDB, users, testTTL, nil, nil)
	t.Cleanup(auth.Close)
	globalContext.auth = auth

	mux := http.NewServeMux()
	globalContext.mux = mux

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	web, err := initWeb(ctx, options{}, nil, nil, testLogger, nil, false)
	require.NoError(t, err)

	globalContext.web = web

	loginCookie := generateAuthCookie(t, mux, userName, userPassword)
	require.NotNil(t, loginCookie)

	r := httptest.NewRequest(http.MethodGet, "/control/profile", nil)
	r.AddCookie(loginCookie)
	assertHandlerStatusCode(t, mux, r, http.StatusOK)

	r = httptest.NewRequest(http.MethodGet, "/control/logout", nil)
	r.AddCookie(loginCookie)
	assertHandlerStatusCode(t, mux, r, http.StatusFound)

	r = httptest.NewRequest(http.MethodGet, "/control/profile", nil)
	r.AddCookie(loginCookie)
	assertHandlerStatusCode(t, mux, r, http.StatusForbidden)
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
