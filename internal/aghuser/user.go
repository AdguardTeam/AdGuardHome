// Package aghuser contains types and logic for dealing with AdGuard Home's web
// users.
package aghuser

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserID is the type for the unique IDs of web users.
type UserID uuid.UUID

// NewUserID returns a new web user unique identifier.  Any error returned is an
// error from the cryptographic randomness reader.
func NewUserID() (uid UserID, err error) {
	uuidv7, err := uuid.NewV7()

	return UserID(uuidv7), err
}

// Login is the type for web user logins.
type Login string

// Password is an interface that defines methods for handling web user
// passwords.
type Password interface {
	// Authenticate returns true if the provided password is allowed.
	Authenticate(ctx context.Context, password string) (ok bool)

	// Hash returns a hashed representation of the web user password.
	Hash() (b []byte)
}

// DefaultPassword is the default bcrypt implementation of the [Password]
// interface.
type DefaultPassword struct {
	hash []byte
}

// NewDefaultPassword returns the new properly initialized *DefaultPassword.
func NewDefaultPassword(hash string) (p *DefaultPassword) {
	return &DefaultPassword{
		hash: []byte(hash),
	}
}

// type check
var _ Password = (*DefaultPassword)(nil)

// Authenticate implements [Password] interface for *DefaultPassword.
func (p *DefaultPassword) Authenticate(ctx context.Context, passwd string) (ok bool) {
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(passwd)) == nil
}

// Hash implements [Password] interface for *DefaultPassword.
func (p *DefaultPassword) Hash() (b []byte) {
	return p.hash
}

// User represents a web user.
type User struct {
	ID       UserID
	Login    Login
	Password Password
}

// Authenticate checks the given credentials and returns true if they are valid.
// TODO!! remove
func (u *User) Authenticate(ctx context.Context, login, passwd string) (ok bool) {
	if u.Login != Login(login) {
		return false
	}

	return u.Password.Authenticate(ctx, passwd)
}
