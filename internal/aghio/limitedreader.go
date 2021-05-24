// Package aghio contains extensions for io package's types and methods
package aghio

import (
	"fmt"
	"io"
)

// LimitReachedError records the limit and the operation that caused it.
type LimitReachedError struct {
	Limit int64
}

// Error implements error interface for LimitReachedError.
//
// TODO(a.garipov): Think about error string format.
func (lre *LimitReachedError) Error() string {
	return fmt.Sprintf("attempted to read more than %d bytes", lre.Limit)
}

// limitedReader is a wrapper for io.Reader with limited reader and dealing with
// errors package.
type limitedReader struct {
	r     io.Reader
	limit int64
	n     int64
}

// Read implements Reader interface.
func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.n == 0 {
		return 0, &LimitReachedError{
			Limit: lr.limit,
		}
	}

	if int64(len(p)) > lr.n {
		p = p[0:lr.n]
	}

	n, err = lr.r.Read(p)
	lr.n -= int64(n)

	return n, err
}

// LimitReader wraps Reader to make it's Reader stop with ErrLimitReached after
// n bytes read.
func LimitReader(r io.Reader, n int64) (limited io.Reader, err error) {
	if n < 0 {
		return nil, fmt.Errorf("aghio: invalid n in LimitReader: %d", n)
	}

	return &limitedReader{
		r:     r,
		limit: n,
		n:     n,
	}, nil
}
