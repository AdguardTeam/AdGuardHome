// Package util contains various utilities.
//
// TODO(a.garipov): Such packages are widely considered an antipattern.  Remove
// this when we refactor our project structure.
package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ContainsString checks if string is in the slice of strings.
func ContainsString(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}

// FileExists returns true if file exists.
func FileExists(fn string) bool {
	_, err := os.Stat(fn)
	return err == nil || !os.IsNotExist(err)
}

// RunCommand runs shell command.
func RunCommand(command string, arguments ...string) (int, string, error) {
	cmd := exec.Command(command, arguments...)
	out, err := cmd.Output()
	if err != nil {
		return 1, "", fmt.Errorf("exec.Command(%s) failed: %v: %s", command, err, string(out))
	}

	return cmd.ProcessState.ExitCode(), string(out), nil
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

// IsOpenWrt returns true if host OS is OpenWrt.
func IsOpenWrt() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	const etcDir = "/etc"

	// TODO(e.burkov): Take care of dealing with fs package after updating
	// Go version to 1.16.
	fileInfos, err := ioutil.ReadDir(etcDir)
	if err != nil {
		return false
	}

	// fNameSubstr is a part of a name of the desired file.
	const fNameSubstr = "release"
	osNameData := []byte("OpenWrt")

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}

		if !strings.Contains(fileInfo.Name(), fNameSubstr) {
			continue
		}

		var body []byte
		body, err = ioutil.ReadFile(filepath.Join(etcDir, fileInfo.Name()))
		if err != nil {
			continue
		}

		if bytes.Contains(body, osNameData) {
			return true
		}
	}

	return false
}
