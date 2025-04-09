package aghuser

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/AdguardTeam/golibs/errors"
)

// DB is an interface that defines methods for interacting with user
// information.  All methods must be safe for concurrent use.
//
// TODO(s.chzhen):  Use this.
//
// TODO(s.chzhen):  Consider updating methods to return a clone.
type DB interface {
	// All retrieves all users from the database, sorted by login.
	//
	// TODO(s.chzhen):  Consider function signature change to reflect the
	// in-memory implementation, as it currently always returns nil for error.
	All(ctx context.Context) (users []*User, err error)

	// ByLogin retrieves a user by their login.  u must not be modified.
	//
	// TODO(s.chzhen):  Remove this once user sessions support [UserID].
	ByLogin(ctx context.Context, login Login) (u *User, err error)

	// ByUUID retrieves a user by their unique identifier.  u must not be
	// modified.
	//
	// TODO(s.chzhen):  Use this.
	ByUUID(ctx context.Context, id UserID) (u *User, err error)

	// Create adds a new user to the database.  If the credentials already
	// exist, it returns the [errors.ErrDuplicated] error.  It also can return
	// an error from the cryptographic randomness reader.  u must not be
	// modified.
	Create(ctx context.Context, u *User) (err error)
}

// DefaultDB is the default in-memory implementation of the [DB] interface.
type DefaultDB struct {
	// mu protects all properties below.
	mu *sync.Mutex

	// loginToUserID maps a web user login to their UserID.  The values must not
	// be empty.
	//
	// TODO(s.chzhen):  Remove this once user sessions support [UserID].
	loginToUserID map[Login]UserID

	// userIDToUser maps a UserID to a web user.  The values must not be nil.
	// It must be synchronized with loginToUserID, meaning all UserIDs stored in
	// loginToUserID must also be stored in this map.
	userIDToUser map[UserID]*User
}

// NewDefaultDB returns the new properly initialized *DefaultDB.
func NewDefaultDB() (db *DefaultDB) {
	return &DefaultDB{
		mu:            &sync.Mutex{},
		loginToUserID: map[Login]UserID{},
		userIDToUser:  map[UserID]*User{},
	}
}

// type check
var _ DB = (*DefaultDB)(nil)

// All implements the [DB] interface for *DefaultDB.
func (db *DefaultDB) All(ctx context.Context) (users []*User, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if len(db.userIDToUser) == 0 {
		return nil, nil
	}

	users = slices.SortedStableFunc(
		maps.Values(db.userIDToUser),
		func(a, b *User) (res int) {
			// TODO(s.chzhen):  Consider adding a custom comparer.
			return cmp.Compare(a.Login, b.Login)
		},
	)

	return users, nil
}

// ByLogin implements the [DB] interface for *DefaultDB.
func (db *DefaultDB) ByLogin(ctx context.Context, login Login) (u *User, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	id, ok := db.loginToUserID[login]
	if !ok {
		return nil, nil
	}

	u, ok = db.userIDToUser[id]
	if !ok {
		// Should not happen.
		panic(fmt.Errorf("no web user present with login %q", login))
	}

	return u, nil
}

// ByUUID implements the [DB] interface for *DefaultDB.
func (db *DefaultDB) ByUUID(ctx context.Context, id UserID) (u *User, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	u, ok := db.userIDToUser[id]
	if !ok {
		return nil, nil
	}

	return u, nil
}

// Create implements the [DB] interface for *DefaultDB.
func (db *DefaultDB) Create(ctx context.Context, u *User) (err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if u.ID == (UserID{}) {
		return fmt.Errorf("userid: %w", errors.ErrEmptyValue)
	}

	_, ok := db.userIDToUser[u.ID]
	if ok {
		return fmt.Errorf("userid: %w", errors.ErrDuplicated)
	}

	_, ok = db.loginToUserID[u.Login]
	if ok {
		return fmt.Errorf("login: %w", errors.ErrDuplicated)
	}

	db.userIDToUser[u.ID] = u
	db.loginToUserID[u.Login] = u.ID

	return nil
}
