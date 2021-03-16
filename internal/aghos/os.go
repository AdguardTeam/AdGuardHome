// Package aghos contains utilities for functions requiring system calls.
package aghos

import "syscall"

// CanBindPrivilegedPorts checks if current process can bind to privileged
// ports.
func CanBindPrivilegedPorts() (can bool, err error) {
	return canBindPrivilegedPorts()
}

// SetRlimit sets user-specified limit of how many fd's we can use
// https://github.com/AdguardTeam/AdGuardHome/internal/issues/659.
func SetRlimit(val uint) {
	setRlimit(val)
}

// HaveAdminRights checks if the current user has root (administrator) rights.
func HaveAdminRights() (bool, error) {
	return haveAdminRights()
}

// SendProcessSignal sends signal to a process.
func SendProcessSignal(pid int, sig syscall.Signal) error {
	return sendProcessSignal(pid, sig)
}
