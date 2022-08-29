//go:build darwin || freebsd || linux || openbsd
// +build darwin freebsd linux openbsd

package aghos

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func notifyShutdownSignal(c chan<- os.Signal) {
	signal.Notify(c, unix.SIGINT, unix.SIGQUIT, unix.SIGTERM)
}

func isShutdownSignal(sig os.Signal) (ok bool) {
	switch sig {
	case
		unix.SIGINT,
		unix.SIGQUIT,
		unix.SIGTERM:
		return true
	default:
		return false
	}
}
