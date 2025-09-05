package home

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
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

	globalContext.mux = http.NewServeMux()

	tlsMgr, err := newTLSManager(testutil.ContextWithTimeout(t, testTimeout), &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
	})
	require.NoError(t, err)

	web, err := initWeb(
		testutil.ContextWithTimeout(t, testTimeout),
		options{},
		nil,
		nil,
		testLogger,
		tlsMgr,
		auth,
		agh.EmptyConfigModifier{},
		false,
	)
	require.NoError(t, err)

	globalContext.web = web

	mux := auth.middleware().Wrap(globalContext.mux)

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
