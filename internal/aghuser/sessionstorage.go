package aghuser

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"
)

// SessionStorage is an interface that defines methods for handling web user
// sessions.  All methods must be safe for concurrent use.
//
// TODO(s.chzhen):  Add DeleteAll method.
type SessionStorage interface {
	// New creates a new session for the web user.
	New(ctx context.Context, u *User) (s *Session, err error)

	// FindByToken returns the stored session for the web user based on the session
	// token.
	//
	// TODO(s.chzhen):  Consider function signature change to reflect the
	// in-memory implementation, as it currently always returns nil for error.
	FindByToken(ctx context.Context, t SessionToken) (s *Session, err error)

	// DeleteByToken removes a stored web user session by the provided token.
	DeleteByToken(ctx context.Context, t SessionToken) (err error)

	// Close releases the web user sessions database resources.
	Close() (err error)
}

// DefaultSessionStorageConfig represents the web user session storage
// configuration structure.
type DefaultSessionStorageConfig struct {
	// Logger is used for logging the operation of the session storage.  It must
	// not be nil.
	Logger *slog.Logger

	// Clock is used to get the current time.  It must not be nil.
	Clock timeutil.Clock

	// UserDB contains the web user information such as ID, login, and password.
	// It must not be nil.
	UserDB DB

	// DBPath is the path to the database file where session data is stored.  It
	// must not be empty.
	DBPath string

	// SessionTTL is the default Time-To-Live duration for web user sessions.
	// It specifies how long a session should last and is a required field.
	SessionTTL time.Duration
}

// DefaultSessionStorage is the default bbolt database implementation of the
// [SessionStorage] interface.
type DefaultSessionStorage struct {
	// db is an instance of the bbolt database where web user sessions are
	// stored by [SessionToken] in the [bucketNameSessions] bucket.
	db *bbolt.DB

	// logger is used for logging the operation of the session storage.
	logger *slog.Logger

	// mu protects sessions.
	mu *sync.Mutex

	// clock is used to get the current time.
	clock timeutil.Clock

	// userDB contains the web user information such as ID, login, and password.
	userDB DB

	// sessions maps a session token to a web user session.
	sessions map[SessionToken]*Session

	// sessionTTL is the default Time-To-Live value for web user sessions.
	sessionTTL time.Duration
}

// NewDefaultSessionStorage returns the new properly initialized
// *DefaultSessionStorage.
func NewDefaultSessionStorage(
	ctx context.Context,
	conf *DefaultSessionStorageConfig,
) (ds *DefaultSessionStorage, err error) {
	ds = &DefaultSessionStorage{
		clock:      conf.Clock,
		userDB:     conf.UserDB,
		logger:     conf.Logger,
		mu:         &sync.Mutex{},
		sessions:   map[SessionToken]*Session{},
		sessionTTL: conf.SessionTTL,
	}

	dbFilename := conf.DBPath
	// TODO(s.chzhen):  Pass logger with options.
	ds.db, err = bbolt.Open(dbFilename, aghos.DefaultPermFile, nil)
	if err != nil {
		ds.logger.ErrorContext(ctx, "opening db %q: %w", dbFilename, err)
		if errors.Is(err, berrors.ErrInvalid) {
			const s = "AdGuard Home cannot be initialized due to an incompatible file system.\n" +
				"Please read the explanation here: https://adguard-dns.io/kb/adguard-home/getting-started/#limitations"
			slogutil.PrintLines(ctx, ds.logger, slog.LevelError, "", s)
		}

		return nil, err
	}

	err = ds.loadSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading sessions: %w", err)
	}

	return ds, nil
}

// loadSessions loads web user sessions from the bbolt database.
func (ds *DefaultSessionStorage) loadSessions(ctx context.Context) (err error) {
	tx, err := ds.db.Begin(true)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	needRollback := true
	defer func() {
		if needRollback {
			err = errors.WithDeferred(err, tx.Rollback())
		}
	}()

	bkt := tx.Bucket([]byte(bboltBucketSessions))
	if bkt == nil {
		return nil
	}

	removed, err := ds.processSessions(ctx, bkt)
	if err != nil {
		return fmt.Errorf("processing sessions: %w", err)
	}

	if removed == 0 {
		ds.logger.DebugContext(ctx, "loading sessions from db", "stored", len(ds.sessions))

		return nil
	}

	needRollback = false
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	ds.logger.DebugContext(
		ctx,
		"loading sessions from db",
		"stored", len(ds.sessions),
		"removed", removed,
	)

	return nil
}

// processSessions iterates over the sessions bucket and loads or removes
// sessions as needed.
func (ds *DefaultSessionStorage) processSessions(
	ctx context.Context,
	bkt *bbolt.Bucket,
) (removed int, err error) {
	invalidSessions := [][]byte{}

	err = bkt.ForEach(ds.bboltSessionHandler(ctx, &invalidSessions))
	if err != nil {
		return 0, fmt.Errorf("iterating over sessions: %w", err)
	}

	var errs []error
	for _, s := range invalidSessions {
		if err = bkt.Delete(s); err != nil {
			errs = append(errs, err)
		}
	}

	if err = errors.Join(errs...); err != nil {
		return 0, fmt.Errorf("deleting sessions: %w", err)
	}

	return len(invalidSessions), nil
}

// bboltSessionHandler returns a function for [bbolt.Bucket.ForEach] that
// iterates over stored sessions, deserializes them, and logs any errors
// encountered.  The returned error is always nil, as these errors are
// considered non-critical to stop the iteration process.
func (ds *DefaultSessionStorage) bboltSessionHandler(
	ctx context.Context,
	invalidSessions *[][]byte,
) (fn func(k, v []byte) (err error)) {
	now := ds.clock.Now()

	return func(k, v []byte) (err error) {
		s, err := bboltDecode(v)
		if err != nil {
			*invalidSessions = append(*invalidSessions, k)
			ds.logger.DebugContext(ctx, "deserializing session", slogutil.KeyError, err)

			return nil
		}

		if now.After(s.Expire) {
			*invalidSessions = append(*invalidSessions, k)

			return nil
		}

		u, err := ds.userDB.ByLogin(ctx, s.UserLogin)
		if err != nil {
			// Should not happen, as it currently always returns nil for error.
			panic(err)
		}

		if u == nil {
			*invalidSessions = append(*invalidSessions, k)
			ds.logger.DebugContext(ctx, "no saved user by name", "name", s.UserLogin)

			return nil
		}

		t := SessionToken(k)
		s.Token = t
		s.UserID = u.ID
		ds.sessions[t] = s

		return nil
	}
}

// bboltBucketSessions is the name of the bucket storing web user sessions in
// the bbolt database.
const bboltBucketSessions = "sessions-2"

const (
	// bboltSessionExpireLen is the length of the expire field in the binary
	// entry stored in bbolt.
	bboltSessionExpireLen = 4

	// bboltSessionNameLen is the length of the name field in the binary entry
	// stored in bbolt.
	bboltSessionNameLen = 2
)

// bboltDecode deserializes decodes a binary data into a session.
func bboltDecode(data []byte) (s *Session, err error) {
	if len(data) < bboltSessionExpireLen+bboltSessionNameLen {
		return nil, fmt.Errorf("length of the data is less than expected: got %d", len(data))
	}

	expireData := data[:bboltSessionExpireLen]
	nameLenData := data[bboltSessionExpireLen : bboltSessionExpireLen+bboltSessionNameLen]
	nameData := data[bboltSessionExpireLen+bboltSessionNameLen:]

	nameLen := binary.BigEndian.Uint16(nameLenData)
	if len(nameData) != int(nameLen) {
		return nil, fmt.Errorf("login: expected length %d, got %d", nameLen, len(nameData))
	}

	expire := binary.BigEndian.Uint32(expireData)

	return &Session{
		Expire:    time.Unix(int64(expire), 0),
		UserLogin: Login(nameData),
	}, nil
}

// bboltEncode serializes a session properties into a binary data.
func bboltEncode(s *Session) (data []byte) {
	data = make([]byte, bboltSessionExpireLen+bboltSessionNameLen+len(s.UserLogin))

	expireData := data[:bboltSessionExpireLen]
	nameLenData := data[bboltSessionExpireLen : bboltSessionExpireLen+bboltSessionNameLen]
	nameData := data[bboltSessionExpireLen+bboltSessionNameLen:]

	expire := uint32(s.Expire.Unix())
	binary.BigEndian.PutUint32(expireData, expire)
	binary.BigEndian.PutUint16(nameLenData, uint16(len(s.UserLogin)))
	copy(nameData, []byte(s.UserLogin))

	return data
}

// type check
var _ SessionStorage = (*DefaultSessionStorage)(nil)

// New implements the [SessionStorage] interface for *DefaultSessionStorage.
func (ds *DefaultSessionStorage) New(ctx context.Context, u *User) (s *Session, err error) {
	s = &Session{
		Token:     NewSessionToken(),
		UserID:    u.ID,
		UserLogin: u.Login,
		Expire:    ds.clock.Now().Add(ds.sessionTTL),
	}

	err = ds.store(s)
	if err != nil {
		return nil, fmt.Errorf("storing session: %w", err)
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.sessions[s.Token] = s

	return s, nil
}

// store saves a web user session in the bbolt database.
func (ds *DefaultSessionStorage) store(s *Session) (err error) {
	tx, err := ds.db.Begin(true)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	needRollback := true
	defer func() {
		if needRollback {
			err = errors.WithDeferred(err, tx.Rollback())
		}
	}()

	bkt, err := tx.CreateBucketIfNotExists([]byte(bboltBucketSessions))
	if err != nil {
		return fmt.Errorf("creating bucket: %w", err)
	}

	err = bkt.Put(s.Token[:], bboltEncode(s))
	if err != nil {
		return fmt.Errorf("putting data: %w", err)
	}

	needRollback = false
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// FindByToken implements the [SessionStorage] interface for *DefaultSessionStorage.
func (ds *DefaultSessionStorage) FindByToken(ctx context.Context, t SessionToken) (s *Session, err error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	s, ok := ds.sessions[t]
	if !ok {
		return nil, nil
	}

	now := ds.clock.Now()
	if now.After(s.Expire) {
		err = ds.deleteByToken(ctx, t)
		if err != nil {
			return nil, fmt.Errorf("expired session: %w", err)
		}

		return nil, nil
	}

	return s, nil
}

// DeleteByToken implements the [SessionStorage] interface for
// *DefaultSessionStorage.
func (ds *DefaultSessionStorage) DeleteByToken(ctx context.Context, t SessionToken) (err error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Don't wrap the error because it's informative enough as is.
	return ds.deleteByToken(ctx, t)
}

// deleteByToken removes stored session by token.  ds.mu is expected to be
// locked.
func (ds *DefaultSessionStorage) deleteByToken(ctx context.Context, t SessionToken) (err error) {
	err = ds.remove(ctx, t)
	if err != nil {
		ds.logger.ErrorContext(ctx, "deleting session", slogutil.KeyError, err)

		return err
	}

	delete(ds.sessions, t)

	return nil
}

// remove deletes a web user session from the bbolt database.
func (ds *DefaultSessionStorage) remove(ctx context.Context, t SessionToken) (err error) {
	tx, err := ds.db.Begin(true)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	needRollback := true
	defer func() {
		if needRollback {
			err = errors.WithDeferred(err, tx.Rollback())
		}
	}()

	bkt := tx.Bucket([]byte(bboltBucketSessions))
	if bkt == nil {
		return errors.Error("no bucket")
	}

	err = bkt.Delete(t[:])
	if err != nil {
		return fmt.Errorf("removing data: %w", err)
	}

	needRollback = false
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	ds.logger.DebugContext(ctx, "removed session from db")

	return err
}

// Close implements the [SessionStorage] interface for *DefaultSessionStorage.
func (ds *DefaultSessionStorage) Close() (err error) {
	err = ds.db.Close()
	if err != nil {
		return fmt.Errorf("closing db: %w", err)
	}

	return nil
}
