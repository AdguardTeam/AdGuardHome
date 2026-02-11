package home

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestWeb_HandleGetProfile(t *testing.T) {
	storeGlobals(t)

	const (
		testTTL = 60

		glTokenFileSuffix = "test"

		userName     = "name"
		userPassword = "password"

		path = "/control/profile"
	)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	tempDir := t.TempDir()
	glFilePrefix = tempDir + "/gl_token_"
	glTokenFile := glFilePrefix + glTokenFileSuffix

	glFileData := make([]byte, 4)
	binary.NativeEndian.PutUint32(glFileData, uint32(time.Now().Unix()+testTTL))

	err = os.WriteFile(glTokenFile, glFileData, 0o644)
	require.NoError(t, err)

	sessionsDB := filepath.Join(tempDir, "sessions.db")

	user := &webUser{
		Name:         userName,
		PasswordHash: string(passwordHash),
	}

	auth, err := newAuth(testutil.ContextWithTimeout(t, testTimeout), &authConfig{
		baseLogger:     testLogger,
		rateLimiter:    emptyRateLimiter{},
		trustedProxies: nil,
		dbFilename:     sessionsDB,
		users:          nil,
		sessionTTL:     testTTL * time.Second,
		isGLiNet:       false,
	})
	require.NoError(t, err)

	t.Cleanup(func() { auth.close(testutil.ContextWithTimeout(t, testTimeout)) })

	baseMux := http.NewServeMux()

	tlsMgr, err := newTLSManager(testutil.ContextWithTimeout(t, testTimeout), &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{
		tlsManager: tlsMgr,
		auth:       auth,
		mux:        baseMux,
	})
	require.NoError(t, err)

	globalContext.web = web

	mux := auth.middleware().Wrap(baseMux)

	require.True(t, t.Run("userless", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)

		web.handleGetProfile(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	}))

	require.True(t, t.Run("add_user", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		err = auth.addUser(ctx, user, userPassword)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)

		web.handleGetProfile(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, path, nil)

		loginCookie := generateAuthCookie(t, mux, userName, userPassword)
		r.AddCookie(loginCookie)

		web.handleGetProfile(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}))
}

func TestWeb_HandlePutProfile(t *testing.T) {
	storeGlobals(t)

	mw := &webMw{}
	mux := http.NewServeMux()
	httpReg := aghhttp.NewDefaultRegistrar(mux, mw.wrap)

	isConfigChanged := false
	confModifier := &aghtest.ConfigModifier{
		OnApply: func(_ context.Context) { isConfigChanged = true },
	}

	web := newTestWeb(t, &webConfig{
		mux:            mux,
		configModifier: confModifier,
		httpReg:        httpReg,
	})

	globalContext.web = web
	mw.set(web)

	var (
		dataValid = errors.Must(json.Marshal(&profileJSON{
			Language: "en",
			Theme:    "auto",
		}))

		dataInvalidLang = errors.Must(json.Marshal(&profileJSON{
			Language: "invalid_lang",
			Theme:    "auto",
		}))

		dataInvalidTheme = errors.Must(json.Marshal(&profileJSON{
			Language: "en",
			Theme:    "invalid_theme",
		}))
	)

	testCases := []struct {
		req      *http.Request
		name     string
		wantBody string
		wantCode int
	}{{
		req:      newProfileUpdateRequest(http.MethodPut, dataValid, true),
		name:     "basic",
		wantBody: "OK\n",
		wantCode: http.StatusOK,
	}, {
		req:      newProfileUpdateRequest(http.MethodGet, dataValid, true),
		name:     "invalid_method",
		wantBody: "only method PUT is allowed\n",
		wantCode: http.StatusMethodNotAllowed,
	}, {
		req:      newProfileUpdateRequest(http.MethodPut, dataValid, false),
		name:     "invalid_content_type",
		wantBody: "only content-type application/json is allowed\n",
		wantCode: http.StatusUnsupportedMediaType,
	}, {
		req:      newProfileUpdateRequest(http.MethodPut, nil, false),
		name:     "empty_body",
		wantBody: "reading req: EOF\n",
		wantCode: http.StatusBadRequest,
	}, {
		req:      newProfileUpdateRequest(http.MethodPut, dataInvalidLang, true),
		name:     "invalid_language",
		wantBody: `unknown language: "invalid_lang"` + "\n",
		wantCode: http.StatusBadRequest,
	}, {
		req:  newProfileUpdateRequest(http.MethodPut, dataInvalidTheme, true),
		name: "invalid_theme",
		wantBody: `reading req: invalid theme "invalid_theme", ` +
			`supported: "auto", "dark", "light"` + "\n",
		wantCode: http.StatusBadRequest,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, tc.req)
			assert.Equal(t, tc.wantCode, w.Code)
			assert.Equal(t, tc.wantBody, w.Body.String())
		})
	}

	require.True(t, t.Run("single_config_update", func(t *testing.T) {
		isConfigChanged = false
		config.Language = ""
		config.Theme = ""

		w := httptest.NewRecorder()

		mux.ServeHTTP(w, newProfileUpdateRequest(http.MethodPut, dataValid, true))
		require.Equal(t, http.StatusOK, w.Code)

		assert.True(t, isConfigChanged)

		isConfigChanged = false

		mux.ServeHTTP(w, newProfileUpdateRequest(http.MethodPut, dataValid, true))
		require.Equal(t, http.StatusOK, w.Code)

		assert.False(t, isConfigChanged)
	}))
}

// newProfileUpdateRequest builds an *http.Request for the profile update
// endpoint.  If body is non-nil, it is used as the request body.  If setCT is
// true, the Content-Type header is set to application/json.
func newProfileUpdateRequest(method string, body []byte, setCT bool) (req *http.Request) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}

	req = httptest.NewRequest(method, "/control/profile/update", r)
	if setCT {
		req.Header.Set(httphdr.ContentType, aghhttp.HdrValApplicationJSON)
	}

	return req
}
