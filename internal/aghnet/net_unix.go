//go:build darwin || freebsd || linux || openbsd

package aghnet

import (
	"io"
	"syscall"

	"github.com/AdguardTeam/golibs/errors"
)

// closePortChecker closes c.  c must be non-nil.
func closePortChecker(c io.Closer) (err error) {
	return c.Close()
}

func isAddrInUse(err syscall.Errno) (ok bool) {
	return errors.Is(err, syscall.EADDRINUSE)
}
