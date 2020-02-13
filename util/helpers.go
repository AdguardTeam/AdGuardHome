package util

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

// ---------------------
// general helpers
// ---------------------

// fileExists returns TRUE if file exists
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	if err != nil {
		return false
	}
	return true
}

// runCommand runs shell command
func RunCommand(command string, arguments ...string) (int, string, error) {
	cmd := exec.Command(command, arguments...)
	out, err := cmd.Output()
	if err != nil {
		return 1, "", fmt.Errorf("exec.Command(%s) failed: %s", command, err)
	}

	return cmd.ProcessState.ExitCode(), string(out), nil
}

// ---------------------
// debug logging helpers
// ---------------------
func FuncName() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}

// SplitNext - split string by a byte and return the first chunk
// Whitespace is trimmed
func SplitNext(str *string, splitBy byte) string {
	i := strings.IndexByte(*str, splitBy)
	s := ""
	if i != -1 {
		s = (*str)[0:i]
		*str = (*str)[i+1:]
	} else {
		s = *str
		*str = ""
	}
	return strings.TrimSpace(s)
}
