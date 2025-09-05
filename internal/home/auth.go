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

// sessionsDBName is the name of the file where session data is stored.
const sessionsDBName = "sessions.db"

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

// authConfig is the configuration structure for [auth].
type authConfig struct {
	// baseLogger is used for creating other loggers.  It must not be nil.
	baseLogger *slog.Logger

	// rateLimiter manages the rate limiting for login attempts.  It must not be
	// nil.
	rateLimiter loginRateLimiter

	// trustedProxies is a set of subnets considered as trusted.
	trustedProxies netutil.SubnetSet

	// dbFilename is the name of the file where session data is stored.  It must
	// not be empty.
	dbFilename string

	// users contains web user information from the configuration file.
	users []webUser

	// sessionTTL is the TTL (Time To Live) for web user sessions.
	sessionTTL time.Duration

	// isGLiNet indicates whether GLiNet mode is enabled.
	isGLiNet bool
}

// auth stores web user information and handles authentication.
type auth struct {
	// logger is used to log the operation of the auth module.
	logger *slog.Logger

	// rateLimiter manages rate limiting for login attempts.
	rateLimiter loginRateLimiter

	// trustedProxies is a set of subnets considered trusted.
	trustedProxies netutil.SubnetSet

	// sessions stores web users' sessions.
	sessions aghuser.SessionStorage

	// users stores user credentials.
	users aghuser.DB

	// isGLiNet indicates whether GLiNet mode is enabled.
	isGLiNet bool

	// isUserless indicates that there are no users defined in the configuration
	// file.
	isUserless bool
}

// newAuth returns the new properly initialized *auth.
func newAuth(ctx context.Context, conf *authConfig) (a *auth, err error) {
	userDB := aghuser.NewDefaultDB()
	for i, u := range conf.users {
		err = userDB.Create(ctx, u.toUser())
		if err != nil {
			return nil, fmt.Errorf("users: at index %d: %w", i, err)
		}
	}

	s, err := aghuser.NewDefaultSessionStorage(ctx, &aghuser.DefaultSessionStorageConfig{
		Logger:     conf.baseLogger.With(slogutil.KeyPrefix, "session_storage"),
		Clock:      timeutil.SystemClock{},
		UserDB:     userDB,
		DBPath:     conf.dbFilename,
		SessionTTL: conf.sessionTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating session storage: %w", err)
	}

	return &auth{
		logger:         conf.baseLogger.With(slogutil.KeyPrefix, "auth"),
		rateLimiter:    conf.rateLimiter,
		trustedProxies: conf.trustedProxies,
		sessions:       s,
		users:          userDB,
		isGLiNet:       conf.isGLiNet,
		isUserless:     len(conf.users) == 0,
	}, nil
}

// middleware returns authentication middleware.
func (a *auth) middleware() (mw httputil.Middleware) {
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
func (a *auth) usersList(ctx context.Context) (webUsers []webUser) {
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
func (a *auth) addUser(ctx context.Context, u *webUser, password string) (err error) {
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

	a.isUserless = false

	a.logger.DebugContext(ctx, "added user", "login", u.Name)

	return nil
}

// close closes the authentication database.
func (a *auth) close(ctx context.Context) {
	err := a.sessions.Close()
	if err != nil {
		a.logger.ErrorContext(ctx, "closing session storage", slogutil.KeyError, err)
	}
}
