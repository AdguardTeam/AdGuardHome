// Package util contains various utilities.
//
// TODO(a.garipov): Such packages are widely considered an antipattern.  Remove
// this when we refactor our project structure.
package util

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

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
