package home

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"

	"github.com/joomcode/errorx"
)

type netInterface struct {
	Name         string
	MTU          int
	HardwareAddr string
	Addresses    []string
	Flags        string
}

// getValidNetInterfacesMap returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func getValidNetInterfacesForWeb() ([]netInterface, error) {
	ifaces, err := dhcpd.GetValidNetInterfaces()
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't get interfaces")
	}
	if len(ifaces) == 0 {
		return nil, errors.New("couldn't find any legible interface")
	}

	var netInterfaces []netInterface

	for _, iface := range ifaces {
		addrs, e := iface.Addrs()
		if e != nil {
			return nil, errorx.Decorate(e, "Failed to get addresses for interface %s", iface.Name)
		}

		netIface := netInterface{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr.String(),
		}

		if iface.Flags != 0 {
			netIface.Flags = iface.Flags.String()
		}

		// we don't want link-local addresses in json, so skip them
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				// not an IPNet, should not happen
				return nil, fmt.Errorf("got iface.Addrs() element %s that is not net.IPNet, it is %T", addr, addr)
			}
			// ignore link-local
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			netIface.Addresses = append(netIface.Addresses, ipnet.IP.String())
		}
		if len(netIface.Addresses) != 0 {
			netInterfaces = append(netInterfaces, netIface)
		}
	}

	return netInterfaces, nil
}

// Get interface name by its IP address.
func getInterfaceByIP(ip string) string {
	ifaces, err := getValidNetInterfacesForWeb()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		for _, addr := range iface.Addresses {
			if ip == addr {
				return iface.Name
			}
		}
	}

	return ""
}

// checkPortAvailable is not a cheap test to see if the port is bindable, because it's actually doing the bind momentarily
func checkPortAvailable(host string, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	_ = ln.Close()

	// It seems that net.Listener.Close() doesn't close file descriptors right away.
	// We wait for some time and hope that this fd will be closed.
	time.Sleep(100 * time.Millisecond)
	return nil
}

func checkPacketPortAvailable(host string, port int) error {
	ln, err := net.ListenPacket("udp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	_ = ln.Close()

	// It seems that net.Listener.Close() doesn't close file descriptors right away.
	// We wait for some time and hope that this fd will be closed.
	time.Sleep(100 * time.Millisecond)
	return err
}

// check if error is "address already in use"
func errorIsAddrInUse(err error) bool {
	errOpError, ok := err.(*net.OpError)
	if !ok {
		return false
	}

	errSyscallError, ok := errOpError.Err.(*os.SyscallError)
	if !ok {
		return false
	}

	errErrno, ok := errSyscallError.Err.(syscall.Errno)
	if !ok {
		return false
	}

	if runtime.GOOS == "windows" {
		const WSAEADDRINUSE = 10048
		return errErrno == WSAEADDRINUSE
	}

	return errErrno == syscall.EADDRINUSE
}
