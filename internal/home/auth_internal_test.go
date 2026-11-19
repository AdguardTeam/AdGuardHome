package home

import (
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuth_UsersList(t *testing.T) {
	t.Parallel()

	sessionsDB := filepath.Join(t.TempDir(), "sessions.db")
	auth, err := newAuth(testutil.ContextWithTimeout(t, testTimeout), &authConfig{
		baseLogger:     testLogger,
		rateLimiter:    emptyRateLimiter{},
		trustedProxies: testTrustedProxies,
		dbFilename:     sessionsDB,
		users:          nil,
		sessionTTL:     testTimeout,
		isGLiNet:       false,
	})
	require.NoError(t, err)

	t.Cleanup(func() { auth.close(testutil.ContextWithTimeout(t, testTimeout)) })

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	assert.Empty(t, auth.usersList(ctx))

	err = auth.addUser(ctx, testUsername, testPassword)
	require.NoError(t, err)

	users := auth.usersList(ctx)
	require.Len(t, users, 1)

	user := users[0]
	assert.Equal(t, user.Name, testUsername)
	assert.NoError(
		t,
		bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(testPassword)),
	)
}
