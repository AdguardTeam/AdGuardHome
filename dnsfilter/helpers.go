package dnsfilter

import (
	"strings"
	"sync/atomic"
)

func isValidRule(rule string) bool {
	if len(rule) < 4 {
		return false
	}
	if rule[0] == '!' {
		return false
	}
	if rule[0] == '#' {
		return false
	}
	if strings.HasPrefix(rule, "[Adblock") {
		return false
	}

	masks := []string{
		"##",
		"#@#",
		"#$#",
		"#@$#",
		"$$",
		"$@$",
		"#%#",
		"#@%#",
	}
	for _, mask := range masks {
		if strings.Contains(rule, mask) {
			return false
		}
	}

	return true
}

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
