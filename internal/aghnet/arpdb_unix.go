//go:build !windows
// +build !windows

package aghnet

import (
	"io"
)

// rcArpA runs "arp -a".
func rcArpA() (r io.Reader, err error) {
	return runCmd("arp", "-a")
}
