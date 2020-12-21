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
// TODO(a.garipov): Think about error string format.
func (lre *LimitReachedError) Error() string {
	return fmt.Sprintf("attempted to read more than %d bytes", lre.Limit)
}

// limitedReadCloser is a wrapper for io.ReadCloser with limited reader and
// dealing with agherr package.
type limitedReadCloser struct {
	limit int64
	n     int64
	rc    io.ReadCloser
}

// Read implements Reader interface.
func (lrc *limitedReadCloser) Read(p []byte) (n int, err error) {
	if lrc.n == 0 {
		return 0, &LimitReachedError{
			Limit: lrc.limit,
		}
	}
	if int64(len(p)) > lrc.n {
		p = p[0:lrc.n]
	}
	n, err = lrc.rc.Read(p)
	lrc.n -= int64(n)
	return n, err
}

// Close implements Closer interface.
func (lrc *limitedReadCloser) Close() error {
	return lrc.rc.Close()
}

// LimitReadCloser wraps ReadCloser to make it's Reader stop with
// ErrLimitReached after n bytes read.
func LimitReadCloser(rc io.ReadCloser, n int64) (limited io.ReadCloser, err error) {
	if n < 0 {
		return nil, fmt.Errorf("aghio: invalid n in LimitReadCloser: %d", n)
	}
	return &limitedReadCloser{
		limit: n,
		n:     n,
		rc:    rc,
	}, nil
}
