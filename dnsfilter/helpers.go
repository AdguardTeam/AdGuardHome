package dnsfilter

import (
	"sync/atomic"
)

func updateMax(valuePtr *int64, maxPtr *int64) {
	for {
		current := atomic.LoadInt64(valuePtr)
		max := atomic.LoadInt64(maxPtr)
		if current <= max {
			break
		}
		swapped := atomic.CompareAndSwapInt64(maxPtr, max, current)
		if swapped {
			break
		}
		// swapping failed because value has changed after reading, try again
	}
}
