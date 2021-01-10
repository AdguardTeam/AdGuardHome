package util

import (
	"fmt"
	"net"
)

// CanBindPort - checks if we can bind to this port or not
func CanBindPort(port int) (bool, error) {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return false, err
	}
	_ = l.Close()
	return true, nil
}
