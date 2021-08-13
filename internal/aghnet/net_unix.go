//go:build openbsd || freebsd || linux
// +build openbsd freebsd linux

package aghnet

// interfaceName is a string containing network interface's name.  The name is
// used in file walking methods.
type interfaceName string
