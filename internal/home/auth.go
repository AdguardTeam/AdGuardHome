package home

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/crypto/bcrypt"
)

// sessionsDBName is the name of the file where session data is stored.
const sessionsDBName = "sessions.db"

// authConfig is the configuration structure for [auth].
type authConfig struct {
	// baseLogger is used for creating other loggers.  It must not be nil.
	baseLogger *slog.Logger

	// gliNetTokenRoot is the root where GLiNet tokens are stored.  It must not
	// be nil if isGLiNet is true.
	gliNetTokenRoot *os.Root

	// rateLimiter manages the rate limiting for login attempts.  It must not be
	// nil.
	rateLimiter loginRateLimiter

	// trustedProxies is a set of subnets considered as trusted.
	trustedProxies netutil.SubnetSet

	// dbFilename is the name of the file where session data is stored.  It must
	// not be empty.
	dbFilename string

	// users contains web user information from the configuration file.
	users []*aghuser.User

	// sessionTTL is the TTL (Time To Live) for web user sessions.
	sessionTTL time.Duration

	// isGLiNet indicates whether GLiNet mode is enabled.
	isGLiNet bool
}

// auth stores web user information and handles authentication.
type auth struct {
	// logger is used to log the operation of the auth module.
	logger *slog.Logger

	// gliNetTokenRoot is the root where GLiNet tokens are stored.  It must not
	// be nil if isGLiNet is true.
	gliNetTokenRoot *os.Root

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
		err = userDB.Create(ctx, u)
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
		logger:          conf.baseLogger.With(slogutil.KeyPrefix, "auth"),
		rateLimiter:     conf.rateLimiter,
		trustedProxies:  conf.trustedProxies,
		gliNetTokenRoot: conf.gliNetTokenRoot,
		sessions:        s,
		users:           userDB,
		isGLiNet:        conf.isGLiNet,
		isUserless:      len(conf.users) == 0,
	}, nil
}

// middleware returns authentication middleware.
func (a *auth) middleware() (mw httputil.Middleware) {
	if a.isGLiNet {
		return newAuthMiddlewareGLiNet(&authMiddlewareGLiNetConfig{
			logger:        a.logger,
			clock:         timeutil.SystemClock{},
			tokenFileRoot: a.gliNetTokenRoot,
			ttl:           glTokenTimeout,
			maxTokenSize:  MaxFileSize,
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
func (a *auth) usersList(ctx context.Context) (webUsers []*configmgr.WebUser) {
	users, err := a.users.All(ctx)
	if err != nil {
		// Should not happen.
		panic(err)
	}

	webUsers = make([]*configmgr.WebUser, 0, len(users))
	for _, u := range users {
		webUsers = append(webUsers, &configmgr.WebUser{
			Name:         string(u.Login),
			PasswordHash: string(u.Password.Hash()),
		})
	}

	return webUsers
}

// addUser adds a new user with the given password.
func (a *auth) addUser(ctx context.Context, name, password string) (err error) {
	if len(password) == 0 {
		return errors.Error("empty password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generating hash: %w", err)
	}

	login, err := aghuser.NewLogin(name)
	if err != nil {
		return fmt.Errorf("creating login: %w", err)
	}

	u := &aghuser.User{
		Password: aghuser.NewDefaultPassword(string(hash)),
		Login:    login,
		ID:       aghuser.NewUserID(),
	}

	err = a.users.Create(ctx, u)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	a.isUserless = false

	a.logger.DebugContext(ctx, "added user", "login", name)

	return nil
}

// close closes the authentication database.
func (a *auth) close(ctx context.Context) {
	err := a.sessions.Close()
	if err != nil {
		a.logger.ErrorContext(ctx, "closing session storage", slogutil.KeyError, err)
	}
}
