package home

import (
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuth_UsersList(t *testing.T) {
	const (
		userName     = "name"
		userPassword = "password"
	)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	require.NoError(t, err)

	sessionsDB := filepath.Join(t.TempDir(), "sessions.db")

	user := webUser{
		Name:         userName,
		PasswordHash: string(passwordHash),
		UserID:       aghuser.MustNewUserID(),
	}

	auth, err := newAuth(testutil.ContextWithTimeout(t, testTimeout), &authConfig{
		baseLogger:     testLogger,
		rateLimiter:    emptyRateLimiter{},
		trustedProxies: nil,
		dbFilename:     sessionsDB,
		users:          nil,
		sessionTTL:     testTimeout,
		isGLiNet:       false,
	})
	require.NoError(t, err)

	t.Cleanup(func() { auth.close(testutil.ContextWithTimeout(t, testTimeout)) })

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	assert.Empty(t, auth.usersList(ctx))

	err = auth.addUser(ctx, &user, userPassword)
	require.NoError(t, err)

	assert.Equal(t, []webUser{user}, auth.usersList(ctx))
}
