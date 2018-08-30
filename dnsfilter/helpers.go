package dnsfilter

import (
	"fmt"
	"path"
	"runtime"
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
		if swapped == true {
			break
		}
		// swapping failed because value has changed after reading, try again
	}
}

//
// helper functions for debugging and testing
//
func _Func() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}

func trace(format string, args ...interface{}) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%s(): ", path.Base(f.Name())))
	text := fmt.Sprintf(format, args...)
	buf.WriteString(text)
	if len(text) == 0 || text[len(text)-1] != '\n' {
		buf.WriteRune('\n')
	}
	fmt.Print(buf.String())
}
