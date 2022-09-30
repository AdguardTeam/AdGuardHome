//go:build windows

package aghos

// TODO(a.garipov): Think of a way to implement these.  Perhaps by using
// syscall.CreateProcessAsUser or something from the golang.org/x/sys module.

func setGroup(_ string) (err error) {
	return Unsupported("setgid")
}

func setUser(_ string) (err error) {
	return Unsupported("setuid")
}
