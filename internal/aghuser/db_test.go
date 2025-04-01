package aghuser_test

import (
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

func TestDB(t *testing.T) {
	db := aghuser.NewDefaultDB()

	const (
		userWithIDPassRaw = "user_with_id_password"
		userNoIDPassRaw   = "user_no_id_password"
	)

	userWithIDPassHash, err := bcrypt.GenerateFromPassword(
		[]byte(userWithIDPassRaw),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)

	userNoIDPassHash, err := bcrypt.GenerateFromPassword(
		[]byte(userNoIDPassRaw),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)

	userWithIDPass := aghuser.NewDefaultPassword(string(userWithIDPassHash))
	userNoIDPass := aghuser.NewDefaultPassword(string(userNoIDPassHash))

	var (
		userWithID = &aghuser.User{
			ID:       aghuser.MustNewUserID(),
			Login:    "user_with_id",
			Password: userWithIDPass,
		}
		userNoID = &aghuser.User{
			Login:    "user_no_id",
			Password: userNoIDPass,
		}
		userDuplicateLogin = &aghuser.User{
			ID:       aghuser.MustNewUserID(),
			Login:    userWithID.Login,
			Password: userWithIDPass,
		}
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	err = db.Create(ctx, userWithID)
	require.NoError(t, err)

	err = db.Create(ctx, userNoID)
	require.NoError(t, err)

	err = db.Create(ctx, userDuplicateLogin)
	assert.ErrorIs(t, err, aghuser.ErrDuplicateCredentials)

	got, err := db.ByUUID(ctx, userWithID.ID)
	require.NoError(t, err)

	assert.Equal(t, userWithID, got)
	assert.True(t, got.Password.Authenticate(ctx, userWithIDPassRaw))

	got, err = db.ByLogin(ctx, userNoID.Login)
	require.NoError(t, err)

	assert.Equal(t, userNoID, got)
	assert.True(t, got.Password.Authenticate(ctx, userNoIDPassRaw))

	users, err := db.All(ctx)
	require.NoError(t, err)

	assert.Len(t, users, 2)
	assert.Equal(t, []*aghuser.User{userNoID, userWithID}, users)
}
