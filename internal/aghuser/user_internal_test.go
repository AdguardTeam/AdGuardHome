package aghuser

import "fmt"

// MustNewUserID is a wrapper around [NewUserID] that panics if there is an
// error.  It is currently only used in tests.
func MustNewUserID() (uid UserID) {
	uid, err := NewUserID()
	if err != nil {
		panic(fmt.Errorf("unexpected uuidv7 error: %w", err))
	}

	return uid
}
