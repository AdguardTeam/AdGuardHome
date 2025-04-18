package aghuser_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/faketime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addSession is a helper function that saves and returns a session for a newly
// generated [aghuser.User] by login.
func addSession(
	tb testing.TB,
	ctx context.Context,
	ds aghuser.SessionStorage,
	login aghuser.Login,
) (s *aghuser.Session) {
	tb.Helper()

	s, err := ds.New(ctx, &aghuser.User{
		ID:    aghuser.MustNewUserID(),
		Login: login,
	})
	require.NoError(tb, err)
	require.NotNil(tb, s)

	var got *aghuser.Session
	got, err = ds.FindByToken(ctx, s.Token)
	require.NoError(tb, err)
	require.NotNil(tb, got)

	assert.Equal(tb, login, got.UserLogin)

	return s
}

func TestDefaultSessionStorage(t *testing.T) {
	const (
		userLoginFirst  aghuser.Login = "user_one"
		userLoginSecond aghuser.Login = "user_two"
	)

	var (
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		logger = slogutil.NewDiscardLogger()
	)

	const (
		sessionTTL = time.Minute
		timeStep   = time.Second
	)

	// Set up a mock clock to test expired sessions. Each call to [clock.Now]
	// will return the [date] incremented by [timeStep].
	date := time.Now()
	clock := &faketime.Clock{
		OnNow: func() (now time.Time) {
			date = date.Add(timeStep)

			return date
		},
	}

	dbFile, err := os.CreateTemp(t.TempDir(), "sessions.db")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, dbFile.Close)

	userDB := aghuser.NewDefaultDB()

	err = userDB.Create(ctx, &aghuser.User{
		Login: userLoginFirst,
		ID:    aghuser.MustNewUserID(),
	})
	require.NoError(t, err)

	err = userDB.Create(ctx, &aghuser.User{
		Login: userLoginSecond,
		ID:    aghuser.MustNewUserID(),
	})
	require.NoError(t, err)

	var (
		ds *aghuser.DefaultSessionStorage

		sessionFirst  *aghuser.Session
		sessionSecond *aghuser.Session
	)

	require.True(t, t.Run("prepare_session_storage", func(t *testing.T) {
		ds, err = aghuser.NewDefaultSessionStorage(ctx, &aghuser.DefaultSessionStorageConfig{
			Clock:      clock,
			UserDB:     userDB,
			Logger:     logger,
			DBPath:     dbFile.Name(),
			SessionTTL: sessionTTL,
		})
		require.NoError(t, err)

		sessionFirst = addSession(t, ctx, ds, userLoginFirst)

		// Advance time to ensure the first session expires before creating the
		// second session.
		date = date.Add(time.Hour)

		sessionSecond = addSession(t, ctx, ds, userLoginSecond)

		err = ds.Close()
		require.NoError(t, err)
	}))

	require.True(t, t.Run("load_sessions", func(t *testing.T) {
		ds, err = aghuser.NewDefaultSessionStorage(ctx, &aghuser.DefaultSessionStorageConfig{
			Clock:      clock,
			UserDB:     userDB,
			Logger:     logger,
			DBPath:     dbFile.Name(),
			SessionTTL: sessionTTL,
		})
		require.NoError(t, err)

		var got *aghuser.Session
		got, err = ds.FindByToken(ctx, sessionFirst.Token)
		require.NoError(t, err)

		assert.Nil(t, got)

		got, err = ds.FindByToken(ctx, sessionSecond.Token)
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, userLoginSecond, got.UserLogin)

		err = ds.DeleteByToken(ctx, sessionSecond.Token)
		require.NoError(t, err)

		got, err = ds.FindByToken(ctx, sessionSecond.Token)
		require.NoError(t, err)

		assert.Nil(t, got)
	}))

	require.True(t, t.Run("expired_session", func(t *testing.T) {
		testutil.CleanupAndRequireSuccess(t, ds.Close)

		sessionFirst = addSession(t, ctx, ds, userLoginFirst)

		date = date.Add(time.Hour)

		var got *aghuser.Session
		got, err = ds.FindByToken(ctx, sessionFirst.Token)
		require.NoError(t, err)

		assert.Nil(t, got)
	}))
}
