//go:build darwin || freebsd || linux || openbsd

package aghos

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func notifyReconfigureSignal(c chan<- os.Signal) {
	signal.Notify(c, unix.SIGHUP)
}

func notifyShutdownSignal(c chan<- os.Signal) {
	signal.Notify(c, unix.SIGINT, unix.SIGQUIT, unix.SIGTERM)
}

func isReconfigureSignal(sig os.Signal) (ok bool) {
	return sig == unix.SIGHUP
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

func sendShutdownSignal(_ chan<- os.Signal) {
	// On Unix we are already notified by the system.
}
