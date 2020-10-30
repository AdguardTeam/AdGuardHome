package util

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/AdguardTeam/golibs/log"

	"github.com/joomcode/errorx"
)

// NetInterface represents a list of network interfaces
type NetInterface struct {
	Name         string   // Network interface name
	MTU          int      // MTU
	HardwareAddr string   // Hardware address
	Addresses    []string // Array with the network interface addresses
	Subnets      []string // Array with CIDR addresses of this network interface
	Flags        string   // Network interface flags (up, broadcast, etc)
}

// GetValidNetInterfaces returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is a ppp interface or the one that doesn't allow broadcasts
func GetValidNetInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("couldn't get list of interfaces: %s", err)
	}

	netIfaces := []net.Interface{}

	for i := range ifaces {
		iface := ifaces[i]
		netIfaces = append(netIfaces, iface)
	}

	return netIfaces, nil
}

// GetValidNetInterfacesForWeb returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func GetValidNetInterfacesForWeb() ([]NetInterface, error) {
	ifaces, err := GetValidNetInterfaces()
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't get interfaces")
	}
	if len(ifaces) == 0 {
		return nil, errors.New("couldn't find any legible interface")
	}

	var netInterfaces []NetInterface

	for _, iface := range ifaces {
		addrs, e := iface.Addrs()
		if e != nil {
			return nil, errorx.Decorate(e, "Failed to get addresses for interface %s", iface.Name)
		}

		netIface := NetInterface{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr.String(),
		}

		if iface.Flags != 0 {
			netIface.Flags = iface.Flags.String()
		}

		// Collect network interface addresses
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				// not an IPNet, should not happen
				return nil, fmt.Errorf("got iface.Addrs() element %s that is not net.IPNet, it is %T", addr, addr)
			}
			// ignore link-local
			if ipNet.IP.IsLinkLocalUnicast() {
				continue
			}
			netIface.Addresses = append(netIface.Addresses, ipNet.IP.String())
			netIface.Subnets = append(netIface.Subnets, ipNet.String())
		}

		// Discard interfaces with no addresses
		if len(netIface.Addresses) != 0 {
			netInterfaces = append(netInterfaces, netIface)
		}
	}

	return netInterfaces, nil
}

// GetInterfaceByIP - Get interface name by its IP address.
func GetInterfaceByIP(ip string) string {
	ifaces, err := GetValidNetInterfacesForWeb()
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

// GetSubnet - Get IP address with netmask for the specified interface
// Returns an empty string if it fails to find it
func GetSubnet(ifaceName string) string {
	netIfaces, err := GetValidNetInterfacesForWeb()
	if err != nil {
		log.Error("Could not get network interfaces info: %v", err)
		return ""
	}

	for _, netIface := range netIfaces {
		if netIface.Name == ifaceName && len(netIface.Subnets) > 0 {
			return netIface.Subnets[0]
		}
	}

	return ""
}

// CheckPortAvailable - check if TCP port is available
func CheckPortAvailable(host string, port int) error {
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

// CheckPacketPortAvailable - check if UDP port is available
func CheckPacketPortAvailable(host string, port int) error {
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

// ErrorIsAddrInUse - check if error is "address already in use"
func ErrorIsAddrInUse(err error) bool {
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
