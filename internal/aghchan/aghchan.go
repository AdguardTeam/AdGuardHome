// Package aghchan contains channel utilities.
package aghchan

import (
	"fmt"
	"time"
)

// Receive returns an error if it cannot receive a value form c before timeout
// runs out.
func Receive[T any](c <-chan T, timeout time.Duration) (v T, ok bool, err error) {
	var zero T
	timeoutCh := time.After(timeout)
	select {
	case <-timeoutCh:
		// TODO(a.garipov): Consider implementing [errors.Aser] for
		// os.ErrTimeout.
		return zero, false, fmt.Errorf("did not receive after %s", timeout)
	case v, ok = <-c:
		return v, ok, nil
	}
}

// MustReceive panics if it cannot receive a value form c before timeout runs
// out.
func MustReceive[T any](c <-chan T, timeout time.Duration) (v T, ok bool) {
	v, ok, err := Receive(c, timeout)
	if err != nil {
		panic(err)
	}

	return v, ok
}
