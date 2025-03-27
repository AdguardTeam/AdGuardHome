package aghuser

import (
	"context"
	"sync"

	"github.com/AdguardTeam/golibs/errors"
)

const (
	// ErrEmptyDB is returned by [DB.All] when there are no users in the
	// database.
	ErrEmptyDB errors.Error = "empty user database"

	// ErrUserNotFound is returned by [DB.ByLogin] and [DB.ByUUID] when
	// searching for a user that does not exist.
	ErrUserNotFound errors.Error = "user not found"

	// ErrDuplicateCredentials is returned by [DB.Create] when attempting to add
	// a user with a duplicate [User.Login] or [User.ID] to the database.
	ErrDuplicateCredentials errors.Error = "duplicate credentials"
)

// DB is an interface that defines methods for interacting with user
// information.
type DB interface {
	// All retrieves all users from the database.  If there are no users, it
	// returns [ErrEmptyDB].
	All(ctx context.Context) (users []*User, err error)

	// ByLogin retrieves a user by their login.  If no such user exists, it
	// returns the [ErrUserNotFound] error.  u must not be modified.
	//
	// TODO(s.chzhen):  Remove this once user sessions support [UserID].
	ByLogin(ctx context.Context, login Login) (u *User, err error)

	// ByUUID retrieves a user by their unique identifier.  If no such user
	// exists, it returns the [ErrUserNotFound] error.  u must not be modified.
	//
	// TODO(s.chzhen):  Use this.
	ByUUID(ctx context.Context, id UserID) (u *User, err error)

	// Create adds a new user to the database.  If the credentials already
	// exist, it returns the [ErrDuplicateCredentials] error.  u must not be
	// modified.  u must not be nil.
	Create(ctx context.Context, u *User) (err error)
}

// DefaultDB is the default in-memory implementation of the [DB] interface.
type DefaultDB struct {
	// mu protects all properties below.
	mu *sync.Mutex

	// loginToUserID maps a web user login to their UserID.
	//
	// TODO(s.chzhen):  Remove this once user sessions support [UserID].
	loginToUserID map[Login]UserID

	// userIDToUser maps a UserID to a web user.
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

// All implements [DB] interface for *DefaultDB.
func (db *DefaultDB) All(ctx context.Context) (users []*User, err error) {
	if len(db.userIDToUser) == 0 {
		return nil, ErrEmptyDB
	}

	// TODO(s.chzhen):  Consider using [aghalg.SortedMap].
	for _, u := range db.userIDToUser {
		users = append(users, u)
	}

	return users, nil
}

// ByLogin implements [DB] interface for *DefaultDB.
func (db *DefaultDB) ByLogin(ctx context.Context, login Login) (u *User, err error) {
	id, ok := db.loginToUserID[login]
	if !ok {
		return nil, ErrUserNotFound
	}

	return db.userIDToUser[id], nil
}

// ByUUID implements [DB] interface for *DefaultDB.
func (db *DefaultDB) ByUUID(ctx context.Context, id UserID) (u *User, err error) {
	u, ok := db.userIDToUser[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return u, nil
}

// Create implements [DB] interface for *DefaultDB.
func (db *DefaultDB) Create(ctx context.Context, u *User) (err error) {
	// TODO(s.chzhen): !! Use provided [UserID] first.
	uid, err := NewUserID()
	if err != nil {
		// TODO(s.chzhen): !! Mention this error in [DB.Create] documentation.
		return err
	}

	u.ID = uid
	db.userIDToUser[uid] = u
	db.loginToUserID[u.Login] = u.ID

	return nil
}
