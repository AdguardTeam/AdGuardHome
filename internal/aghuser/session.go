package aghuser

import (
	"crypto/rand"
	"time"
)

// SessionToken is the type for the web user session token.
type SessionToken [16]byte

// NewSessionToken returns a cryptographically secure randomly generated web
// user session token.  If an error occurs during random generation, it will
// cause the program to crash.
func NewSessionToken() (t SessionToken) {
	_, _ = rand.Read(t[:])

	return t
}

// Session represents a web user session.
type Session struct {
	// Expire indicates when the session will expire.
	Expire time.Time

	// UserLogin is the login of the web user associated with the session.
	//
	// TODO(s.chzhen):  Remove this field and associate the user by UserID.
	UserLogin Login

	// Token is the session token.
	Token SessionToken

	// UserID is the identifier of the web user associated with the session.
	UserID UserID
}
