package home

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/crypto/bcrypt"
)

// webUser represents a user of the Web UI.
//
// TODO(s.chzhen):  Improve naming.
type webUser struct {
	// Name represents the login name of the web user.
	Name string `yaml:"name"`

	// PasswordHash is the hashed representation of the web user password.
	PasswordHash string `yaml:"password"`

	// UserID is the unique identifier of the web user.
	UserID aghuser.UserID `yaml:"-"`
}

// toUser returns the new properly initialized *aghuser.User using stored
// properties.  It panics if there is an error generating the user ID.
func (wu *webUser) toUser() (u *aghuser.User) {
	uid := wu.UserID
	if uid == (aghuser.UserID{}) {
		uid = aghuser.MustNewUserID()
	}

	return &aghuser.User{
		Password: aghuser.NewDefaultPassword(wu.PasswordHash),
		Login:    aghuser.Login(wu.Name),
		ID:       uid,
	}
}

// authMw is the global authentication object.
//
// TODO(s.chzhen): !! Ratelimiter.  Docs.  Replace [Auth].
type authMw struct {
	logger         *slog.Logger
	rateLimiter    rateLimiterInterface
	trustedProxies netutil.SubnetSet
	sessions       aghuser.SessionStorage
	users          aghuser.DB
	isGLiNet       bool
}

// NewAuthMW initializes the global authentication object.
func NewAuthMW(
	ctx context.Context,
	baseLogger *slog.Logger,
	rateLimiter rateLimiterInterface,
	trustedProxies netutil.SubnetSet,
	dbFilename string,
	users []webUser,
	sessionTTL time.Duration,
	isGLiNet bool,
) (a *authMw, err error) {
	userDB := aghuser.NewDefaultDB()
	for i, u := range users {
		err = userDB.Create(ctx, u.toUser())
		if err != nil {
			return nil, fmt.Errorf("users: at index %d: %w", i, err)
		}
	}

	s, err := aghuser.NewDefaultSessionStorage(ctx, &aghuser.DefaultSessionStorageConfig{
		Logger:     baseLogger.With(slogutil.KeyPrefix, "session_storage"),
		Clock:      timeutil.SystemClock{},
		UserDB:     userDB,
		DBPath:     dbFilename,
		SessionTTL: sessionTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating session storage: %w", err)
	}

	return &authMw{
		logger:         baseLogger.With(slogutil.KeyPrefix, "auth"),
		rateLimiter:    rateLimiter,
		trustedProxies: trustedProxies,
		sessions:       s,
		users:          userDB,
		isGLiNet:       isGLiNet,
	}, nil
}

// TODO(s.chzhen): !! Naming.  Docs.
func (a *authMw) mw() (mw httputil.Middleware) {
	if a.isGLiNet {
		return newAuthMiddlewareGLiNet(&authMiddlewareGLiNetConfig{
			logger:          a.logger,
			clock:           timeutil.SystemClock{},
			tokenFilePrefix: glFilePrefix,
			ttl:             glTokenTimeout,
			maxTokenSize:    MaxFileSize,
		})
	}

	return newAuthMiddlewareDefault(&authMiddlewareDefaultConfig{
		logger:         a.logger,
		rateLimiter:    a.rateLimiter,
		trustedProxies: a.trustedProxies,
		sessions:       a.sessions,
		users:          a.users,
	})
}

// usersList returns a copy of a users list.
func (a *authMw) usersList(ctx context.Context) (webUsers []webUser) {
	users, err := a.users.All(ctx)
	if err != nil {
		// Should not happen.
		panic(err)
	}

	webUsers = make([]webUser, 0, len(users))
	for _, u := range users {
		webUsers = append(webUsers, webUser{
			Name:         string(u.Login),
			PasswordHash: string(u.Password.Hash()),
			UserID:       u.ID,
		})
	}

	return webUsers
}

// addUser adds a new user with the given password.  u must not be nil.
func (a *authMw) addUser(ctx context.Context, u *webUser, password string) (err error) {
	if len(password) == 0 {
		return errors.Error("empty password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generating hash: %w", err)
	}

	u.PasswordHash = string(hash)

	err = a.users.Create(ctx, u.toUser())
	if err != nil {
		// Should not happen.
		panic(err)
	}

	a.logger.DebugContext(ctx, "added user", "login", u.Name)

	return nil
}

// Close closes the authentication database.
func (a *authMw) Close(ctx context.Context) {
	err := a.sessions.Close()
	if err != nil {
		a.logger.ErrorContext(ctx, "closing session storage", slogutil.KeyError, err)
	}
}
