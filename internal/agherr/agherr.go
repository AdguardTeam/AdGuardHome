// Package agherr contains AdGuard Home's error handling helpers.
package agherr

import (
	"fmt"
	"strings"

	"github.com/AdguardTeam/golibs/log"
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
//
// TODO(a.garipov): Add formatting to message.
func Many(message string, underlying ...error) (err error) {
	err = &manyError{
		message:    message,
		underlying: underlying,
	}

	return err
}

// Error implements the error interface for *manyError.
func (e *manyError) Error() (msg string) {
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
func (e *manyError) Unwrap() (err error) {
	if len(e.underlying) == 0 {
		return nil
	}

	return e.underlying[0]
}

// wrapper is a copy of the hidden errors.wrapper interface for tests, linting,
// etc.
type wrapper interface {
	Unwrap() error
}

// Annotate annotates the error with the message, unless the error is nil.  This
// is a helper function to simplify code like this:
//
//   func (f *foo) doStuff(s string) (err error) {
//           defer func() {
//                   if err != nil {
//                           err = fmt.Errorf("bad foo string %q: %w", s, err)
//                   }
//           }()
//
//           // …
//   }
//
// Instead, write:
//
//   func (f *foo) doStuff(s string) (err error) {
//           defer agherr.Annotate("bad foo string %q: %w", &err, s)
//
//           // …
//   }
//
// msg must contain the final ": %w" verb.
//
// TODO(a.garipov): Clearify the function usage.
func Annotate(msg string, errPtr *error, args ...interface{}) {
	if errPtr == nil {
		return
	}

	err := *errPtr
	if err != nil {
		args = append(args, err)

		*errPtr = fmt.Errorf(msg, args...)
	}
}

// LogPanic is a convinient helper function to log a panic in a goroutine.  It
// should not be used where proper error handling is required.
func LogPanic(prefix string) {
	if v := recover(); v != nil {
		if prefix != "" {
			log.Error("%s: recovered from panic: %v", prefix, v)

			return
		}

		log.Error("recovered from panic: %v", v)
	}
}
