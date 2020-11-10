// Package agherr contains the extended error type, and the function for
// wrapping several errors.
package agherr

import (
	"fmt"
	"strings"
)

// Error is the constant error type.
type Error string

// Error implements the error interface for Error.
func (err Error) Error() (msg string) {
	return string(err)
}

// manyError is an error containing several wrapped errors.  It is created to be
// a simpler version of the API provided by github.com/joomcode/errorx.
type manyError struct {
	message    string
	underlying []error
}

// Many wraps several errors and returns a single error.
func Many(message string, underlying ...error) error {
	err := &manyError{
		message:    message,
		underlying: underlying,
	}

	return err
}

// Error implements the error interface for *manyError.
func (e *manyError) Error() string {
	switch len(e.underlying) {
	case 0:
		return e.message
	case 1:
		return fmt.Sprintf("%s: %s", e.message, e.underlying[0])
	default:
		b := &strings.Builder{}

		// Ignore errors, since strings.(*Buffer).Write never returns
		// errors.
		_, _ = fmt.Fprintf(b, "%s: %s (hidden: %s", e.message, e.underlying[0], e.underlying[1])
		for _, u := range e.underlying[2:] {
			// See comment above.
			_, _ = fmt.Fprintf(b, ", %s", u)
		}

		// See comment above.
		_, _ = b.WriteString(")")

		return b.String()
	}
}

// Unwrap implements the hidden errors.wrapper interface for *manyError.
func (e *manyError) Unwrap() error {
	if len(e.underlying) == 0 {
		return nil
	}

	return e.underlying[0]
}
