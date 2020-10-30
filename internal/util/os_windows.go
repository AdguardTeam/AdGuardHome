package util

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// Set user-specified limit of how many fd's we can use
func SetRlimit(val uint) {
}

func HaveAdminRights() (bool, error) {
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

func SendProcessSignal(pid int, sig syscall.Signal) error {
	return fmt.Errorf("not supported on Windows")
}
