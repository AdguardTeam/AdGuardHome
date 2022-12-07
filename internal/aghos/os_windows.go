//go:build windows

package aghos

import (
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/sys/windows"
)

func rootDirFS() (fsys fs.FS) {
	// TODO(a.garipov): Use a better way if golang/go#44279 is ever resolved.
	sysDir, err := windows.GetSystemDirectory()
	if err != nil {
		log.Error("aghos: getting root filesystem: %s; using C:", err)

		// Assume that C: is the safe default.
		return os.DirFS("C:")
	}

	return os.DirFS(filepath.VolumeName(sysDir))
}

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

func notifyReconfigureSignal(c chan<- os.Signal) {
	signal.Notify(c, windows.SIGHUP)
}

func notifyShutdownSignal(c chan<- os.Signal) {
	// syscall.SIGTERM is processed automatically.  See go doc os/signal,
	// section Windows.
	signal.Notify(c, os.Interrupt)
}

func isReconfigureSignal(sig os.Signal) (ok bool) {
	return sig == windows.SIGHUP
}

func isShutdownSignal(sig os.Signal) (ok bool) {
	switch sig {
	case os.Interrupt, syscall.SIGTERM:
		return true
	default:
		return false
	}
}
