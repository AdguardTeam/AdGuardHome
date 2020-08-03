package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

// ContainsString checks if "v" is in the array "arr"
func ContainsString(arr []string, v string) bool {
	for _, i := range arr {
		if i == v {
			return true
		}
	}
	return false
}

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
		return 1, "", fmt.Errorf("exec.Command(%s) failed: %v: %s", command, err, string(out))
	}

	return cmd.ProcessState.ExitCode(), string(out), nil
}

func FuncName() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}

// SplitNext - split string by a byte and return the first chunk
// Skip empty chunks
// Whitespace is trimmed
func SplitNext(str *string, splitBy byte) string {
	i := strings.IndexByte(*str, splitBy)
	s := ""
	if i != -1 {
		s = (*str)[0:i]
		*str = (*str)[i+1:]
		k := 0
		ch := rune(0)
		for k, ch = range *str {
			if byte(ch) != splitBy {
				break
			}
		}
		*str = (*str)[k:]
	} else {
		s = *str
		*str = ""
	}
	return strings.TrimSpace(s)
}

// MinInt - return the minimum value
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsOpenWrt checks if OS is OpenWRT
func IsOpenWrt() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	body, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}

	return strings.Contains(string(body), "OpenWrt")
}

// IsFreeBSD checks if OS is FreeBSD
func IsFreeBSD() bool {
	if runtime.GOOS == "freebsd" {
		return true
	}

	return false
}
