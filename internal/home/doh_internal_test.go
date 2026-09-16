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

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	auth, err := newAuth(ctx, &authConfig{
		baseLogger:     testLogger,
		rateLimiter:    emptyRateLimiter{},
		trustedProxies: testTrustedProxies,
		dbFilename:     sessionsDB,
		users:          users,
		sessionTTL:     testTTL * time.Second,
	})
	require.NoError(t, err)

	t.Cleanup(func() { auth.close(ctx) })

	web := newTestWeb(t, &webConfig{auth: auth})

	const dohPath = "/dns-query"

	t.Run("no_doh_server", func(t *testing.T) {
		h := web.wrapMux(testLogger)

		r := httptest.NewRequest(http.MethodGet, dohPath, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		// Without the DoH server all requests go through the authentication
		// middleware.
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	var dohCalled bool
	web.setDoHServer(newDoHServer(
		testLogger,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dohCalled = true
		}),
		[]string{http.MethodGet + " " + dohPath},
	))

	h := web.wrapMux(testLogger)

	t.Run("doh_bypasses_auth", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, dohPath, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, dohCalled)
	})

	t.Run("other_requests_require_auth", func(t *testing.T) {
		dohCalled = false

		r := httptest.NewRequest(http.MethodGet, "/control/status", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, dohCalled)
	})
}
