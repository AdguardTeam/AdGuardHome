// Package aghchan contains channel utilities.
package aghchan

import (
	"fmt"
	"time"
)

// MustReceive panics if it cannot receive a value form c before timeout runs
// out.
func MustReceive[T any](c <-chan T, timeout time.Duration) (v T, ok bool) {
	timeoutCh := time.After(timeout)
	select {
	case <-timeoutCh:
		panic(fmt.Errorf("did not receive after %s", timeout))
	case v, ok = <-c:
		return v, ok
	}
}
