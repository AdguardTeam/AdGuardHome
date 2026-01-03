//go:build linux

package home

import (
	"fmt"
	"net"
	"os"
	"time"
)

// Notifies the service manager that the program is ready to serve
func notifyReady() error {
	return sdNotify("READY=1")
}

// Notifies the service manager that the program is beginning to reload its
// configuration
func notifyReload() error {
	now := time.Now().UnixMicro()
	return sdNotify(fmt.Sprintf("RELOADING=1\nMONOTONIC_USEC=%v", now))
}

// Implements the sd_notify mechanism
//
// Reference: https://www.freedesktop.org/software/systemd/man/latest/sd_notify.html
func sdNotify(message string) error {
	socketPath := os.Getenv("NOTIFY_SOCKET")
	if socketPath == "" {
		return nil
	}
	socketAddr := net.UnixAddr{
		Name: socketPath,
		Net:  "unixgram",
	}

	conn, err := net.DialUnix("unixgram", nil, &socketAddr)
	if err != nil {
		return fmt.Errorf("connecting to %q: %w", socketAddr.String(), err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("sending %q: %w", message, err)
	}

	return nil
}
