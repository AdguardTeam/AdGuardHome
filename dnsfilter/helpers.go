package dnsfilter

import (
	"fmt"
	"os"
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

	// Filter out all sorts of cosmetic rules:
	// https://kb.adguard.com/en/general/how-to-create-your-own-ad-filters#cosmetic-rules
	masks := []string{
		"##",
		"#@#",
		"#?#",
		"#@?#",
		"#$#",
		"#@$#",
		"#?$#",
		"#@?$#",
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
	fmt.Fprint(os.Stderr, buf.String())
}
