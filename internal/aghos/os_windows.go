//go:build windows
// +build windows

package aghos

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/windows"
)

func setRlimit(val uint64) (err error) {
	return Unsupported("setrlimit")
}

func haveAdminRights() (bool, error) {
	var token windows.Token
	h := windows.CurrentProcess()
	err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token)
	if err != nil {
		return false, err
	}

	info := make([]byte, 4)
	var returnedLen uint32
	err = windows.GetTokenInformation(token, windows.TokenElevation, &info[0], uint32(len(info)), &returnedLen)
	token.Close()
	if err != nil {
		return false, err
	}
	if info[0] == 0 {
		return false, nil
	}
	return true, nil
}

func isOpenWrt() (ok bool) {
	return false
}

func notifyShutdownSignal(c chan<- os.Signal) {
	// syscall.SIGTERM is processed automatically.  See go doc os/signal,
	// section Windows.
	signal.Notify(c, os.Interrupt)
}

func isShutdownSignal(sig os.Signal) (ok bool) {
	switch sig {
	case
		os.Interrupt,
		syscall.SIGTERM:
		return true
	default:
		return false
	}
}
