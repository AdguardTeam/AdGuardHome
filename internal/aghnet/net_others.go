// +build !linux,!darwin

package aghnet

import (
	"fmt"
	"runtime"
)

func ifaceHasStaticIP(string) (bool, error) {
	return false, fmt.Errorf("cannot check if IP is static: not supported on %s", runtime.GOOS)
}

func ifaceSetStaticIP(string) error {
	return fmt.Errorf("cannot set static IP on %s", runtime.GOOS)
}
