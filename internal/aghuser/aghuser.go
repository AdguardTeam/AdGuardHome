package aghuser

import (
	"context"

	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/crypto/bcrypt"
)

// Login is the type for web user logins.
type Login string

// NewLogin returns a web user login.
//
// TODO(s.chzhen): Add more constraints as needed.
func NewLogin(s string) (l Login, err error) {
	if s == "" {
		return "", errors.ErrEmptyValue
	}

	return Login(s), nil
}

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

// Authenticate implements the [Password] interface for *DefaultPassword.
func (p *DefaultPassword) Authenticate(ctx context.Context, passwd string) (ok bool) {
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(passwd)) == nil
}

// Hash implements the [Password] interface for *DefaultPassword.
func (p *DefaultPassword) Hash() (b []byte) {
	return p.hash
}
