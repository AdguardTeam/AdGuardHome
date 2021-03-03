package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

// NetInterface represents an entry of network interfaces map.
type NetInterface struct {
	MTU          int              `json:"mtu"`
	Name         string           `json:"name"`
	HardwareAddr net.HardwareAddr `json:"hardware_address"`
	Flags        net.Flags        `json:"flags"`
	// Array with the network interface addresses.
	Addresses []net.IP `json:"ip_addresses,omitempty"`
	// Array with IP networks for this network interface.
	Subnets []*net.IPNet `json:"-"`
}

// MarshalJSON implements the json.Marshaler interface for *NetInterface.
func (iface *NetInterface) MarshalJSON() ([]byte, error) {
	type netInterface NetInterface
	return json.Marshal(&struct {
		HardwareAddr string `json:"hardware_address"`
		Flags        string `json:"flags"`
		*netInterface
	}{
		HardwareAddr: iface.HardwareAddr.String(),
		Flags:        iface.Flags.String(),
		netInterface: (*netInterface)(iface),
	})
}

// GetValidNetInterfaces returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is a ppp interface or the one that doesn't allow broadcasts
func GetValidNetInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("couldn't get list of interfaces: %w", err)
	}

	netIfaces := []net.Interface{}

	netIfaces = append(netIfaces, ifaces...)

	return netIfaces, nil
}

// GetValidNetInterfacesForWeb returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func GetValidNetInterfacesForWeb() ([]*NetInterface, error) {
	ifaces, err := GetValidNetInterfaces()
	if err != nil {
		return nil, fmt.Errorf("couldn't get interfaces: %w", err)
	}
	if len(ifaces) == 0 {
		return nil, errors.New("couldn't find any legible interface")
	}

	var netInterfaces []*NetInterface

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
		}

		netIface := &NetInterface{
			MTU:          iface.MTU,
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr,
			Flags:        iface.Flags,
		}

		// Collect network interface addresses.
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				// Should be net.IPNet, this is weird.
				return nil, fmt.Errorf("got iface.Addrs() element %s that is not net.IPNet, it is %T", addr, addr)
			}
			// Ignore link-local.
			if ipNet.IP.IsLinkLocalUnicast() {
				continue
			}
			netIface.Addresses = append(netIface.Addresses, ipNet.IP)
			netIface.Subnets = append(netIface.Subnets, ipNet)
		}

		// Discard interfaces with no addresses.
		if len(netIface.Addresses) != 0 {
			netInterfaces = append(netInterfaces, netIface)
		}
	}

	return netInterfaces, nil
}

// GetInterfaceByIP returns the name of interface containing provided ip.
func GetInterfaceByIP(ip net.IP) string {
	ifaces, err := GetValidNetInterfacesForWeb()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		for _, addr := range iface.Addresses {
			if ip.Equal(addr) {
				return iface.Name
			}
		}
	}

	return ""
}

// GetSubnet returns pointer to net.IPNet for the specified interface or nil if
// the search fails.
func GetSubnet(ifaceName string) *net.IPNet {
	netIfaces, err := GetValidNetInterfacesForWeb()
	if err != nil {
		log.Error("Could not get network interfaces info: %v", err)
		return nil
	}

	for _, netIface := range netIfaces {
		if netIface.Name == ifaceName && len(netIface.Subnets) > 0 {
			return netIface.Subnets[0]
		}
	}

	return nil
}

// CheckPortAvailable - check if TCP port is available
func CheckPortAvailable(host net.IP, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host.String(), strconv.Itoa(port)))
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
func CheckPacketPortAvailable(host net.IP, port int) error {
	ln, err := net.ListenPacket("udp", net.JoinHostPort(host.String(), strconv.Itoa(port)))
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
