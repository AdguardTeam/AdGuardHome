//go:build !(windows || linux)
// +build !windows,!linux

package aghnet

func defaultHostsPaths() (paths []string) {
	return []string{"etc/hosts"}
}
