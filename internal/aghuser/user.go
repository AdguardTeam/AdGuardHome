// Package aghuser contains types and logic for dealing with AdGuard Home's web
// users.
package aghuser

import (
	"uuid"
)

// UserID is the type for the unique IDs of web users.
type UserID uuid.UUID

// NewUserID returns a new web user unique identifier.
func NewUserID() (uid UserID) {
	return UserID(uuid.NewV7())
}

// User represents a web user.
type User struct {
	// Password stores the password information for the web user.  It must not
	// be nil.
	Password Password

	// Login is the login name of the web user.  It must not be empty.
	Login Login

	// ID is the unique identifier for the web user.  It must not be empty.
	ID UserID
}
