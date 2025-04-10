package aghuser_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestDB(t *testing.T) {
	db := aghuser.NewDefaultDB()

	const (
		userWithIDPassRaw = "user_with_id_password"
		userSecondPassRaw = "user_second_password"
	)

	userWithIDPassHash, err := bcrypt.GenerateFromPassword(
		[]byte(userWithIDPassRaw),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)

	userSecondPassHash, err := bcrypt.GenerateFromPassword(
		[]byte(userSecondPassRaw),
		bcrypt.DefaultCost,
	)
	require.NoError(t, err)

	userWithIDPass := aghuser.NewDefaultPassword(string(userWithIDPassHash))
	userSecondPass := aghuser.NewDefaultPassword(string(userSecondPassHash))

	var (
		userWithID = &aghuser.User{
			ID:       aghuser.MustNewUserID(),
			Login:    "user_with_id",
			Password: userWithIDPass,
		}
		userSecond = &aghuser.User{
			ID:       aghuser.MustNewUserID(),
			Login:    "user_second",
			Password: userSecondPass,
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

	err = db.Create(ctx, userSecond)
	require.NoError(t, err)

	err = db.Create(ctx, userDuplicateLogin)
	assert.ErrorIs(t, err, errors.ErrDuplicated)

	got, err := db.ByUUID(ctx, userWithID.ID)
	require.NoError(t, err)

	assert.Equal(t, userWithID, got)
	assert.True(t, got.Password.Authenticate(ctx, userWithIDPassRaw))

	got, err = db.ByLogin(ctx, userSecond.Login)
	require.NoError(t, err)

	assert.Equal(t, userSecond, got)
	assert.True(t, got.Password.Authenticate(ctx, userSecondPassRaw))

	users, err := db.All(ctx)
	require.NoError(t, err)

	assert.Len(t, users, 2)
	assert.Equal(t, []*aghuser.User{userSecond, userWithID}, users)
}
