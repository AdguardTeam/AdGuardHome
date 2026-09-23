package home

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestWebAPI_wrapMux(t *testing.T) {
	storeGlobals(t)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	users := []webUser{{
		Name:         testUsername,
		PasswordHash: string(passwordHash),
	}}

	sessionsDB := filepath.Join(t.TempDir(), "sessions.db")

	initCtx := testutil.ContextWithTimeout(t, testTimeout)

	auth, err := newAuth(initCtx, &authConfig{
		baseLogger:     testLogger,
		rateLimiter:    emptyRateLimiter{},
		trustedProxies: testTrustedProxies,
		dbFilename:     sessionsDB,
		users:          users,
		sessionTTL:     testTTL * time.Second,
	})
	require.NoError(t, err)

	t.Cleanup(func() { auth.close(initCtx) })

	web := newTestWeb(t, &webConfig{
		auth: auth,
	})

	const dohPath = "/dns-query"

	require.True(t, t.Run("no_doh_server", func(t *testing.T) {
		h := web.wrapMux(testLogger)

		ctx := testutil.ContextWithTimeout(t, testTimeout)
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, dohPath, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		// Without the DoH server all requests go through the authentication
		// middleware.
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}))

	var dohCalled bool
	web.setDoHServer(newDoHServer(&doHServerConfig{
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dohCalled = true
		}),
		logger: testLogger,
		routes: []string{http.MethodGet + " " + dohPath},
	}))

	h := web.wrapMux(testLogger)

	require.True(t, t.Run("doh_bypasses_auth", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, dohPath, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, dohCalled)
	}))

	require.True(t, t.Run("other_requests_require_auth", func(t *testing.T) {
		dohCalled = false

		ctx := testutil.ContextWithTimeout(t, testTimeout)
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/control/status", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, dohCalled)
	}))
}
