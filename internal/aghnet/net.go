// Package aghnet contains some utilities for networking.
package aghnet

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
)

// ErrNoStaticIPInfo is returned by IfaceHasStaticIP when no information about
// the IP being static is available.
const ErrNoStaticIPInfo errors.Error = "no information about static ip"

// IfaceHasStaticIP checks if interface is configured to have static IP address.
// If it can't give a definitive answer, it returns false and an error for which
// errors.Is(err, ErrNoStaticIPInfo) is true.
func IfaceHasStaticIP(ifaceName string) (has bool, err error) {
	return ifaceHasStaticIP(ifaceName)
}

// IfaceSetStaticIP sets static IP address for network interface.
func IfaceSetStaticIP(ifaceName string) (err error) {
	return ifaceSetStaticIP(ifaceName)
}

// GatewayIP returns IP address of interface's gateway.
func GatewayIP(ifaceName string) net.IP {
	cmd := exec.Command("ip", "route", "show", "dev", ifaceName)
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	d, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return nil
	}

	fields := strings.Fields(string(d))
	// The meaningful "ip route" command output should contain the word
	// "default" at first field and default gateway IP address at third
	// field.
	if len(fields) < 3 || fields[0] != "default" {
		return nil
	}

	return net.ParseIP(fields[2])
}

// CanBindPort checks if we can bind to the given port.
func CanBindPort(port int) (can bool, err error) {
	var addr *net.TCPAddr
	addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false, err
	}

	var listener *net.TCPListener
	listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return false, err
	}
	_ = listener.Close()
	return true, nil
}

// CanBindPrivilegedPorts checks if current process can bind to privileged
// ports.
func CanBindPrivilegedPorts() (can bool, err error) {
	return canBindPrivilegedPorts()
}

// NetInterface represents an entry of network interfaces map.
type NetInterface struct {
	// Addresses are the network interface addresses.
	Addresses []net.IP `json:"ip_addresses,omitempty"`
	// Subnets are the IP networks for this network interface.
	Subnets      []*net.IPNet     `json:"-"`
	Name         string           `json:"name"`
	HardwareAddr net.HardwareAddr `json:"hardware_address"`
	Flags        net.Flags        `json:"flags"`
	MTU          int              `json:"mtu"`
}

// MarshalJSON implements the json.Marshaler interface for NetInterface.
func (iface NetInterface) MarshalJSON() ([]byte, error) {
	type netInterface NetInterface
	return json.Marshal(&struct {
		HardwareAddr string `json:"hardware_address"`
		Flags        string `json:"flags"`
		netInterface
	}{
		HardwareAddr: iface.HardwareAddr.String(),
		Flags:        iface.Flags.String(),
		netInterface: netInterface(iface),
	})
}

// GetValidNetInterfacesForWeb returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func GetValidNetInterfacesForWeb() ([]*NetInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("couldn't get interfaces: %w", err)
	}
	if len(ifaces) == 0 {
		return nil, errors.Error("couldn't find any legible interface")
	}

	var netInterfaces []*NetInterface

	for _, iface := range ifaces {
		var addrs []net.Addr
		addrs, err = iface.Addrs()
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
	ln, err := net.Listen("tcp", netutil.JoinHostPort(host.String(), port))
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
	ln, err := net.ListenPacket("udp", netutil.JoinHostPort(host.String(), port))
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

// SplitHost is a wrapper for net.SplitHostPort for the cases when the hostport
// does not necessarily contain a port.
func SplitHost(hostport string) (host string, err error) {
	host, _, err = net.SplitHostPort(hostport)
	if err != nil {
		// Check for the missing port error.  If it is that error, just
		// use the host as is.
		//
		// See the source code for net.SplitHostPort.
		const missingPort = "missing port in address"

		addrErr := &net.AddrError{}
		if !errors.As(err, &addrErr) || addrErr.Err != missingPort {
			return "", err
		}

		host = hostport
	}

	return host, nil
}

// CollectAllIfacesAddrs returns the slice of all network interfaces IP
// addresses without port number.
func CollectAllIfacesAddrs() (addrs []string, err error) {
	var ifaces []net.Interface
	ifaces, err = net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("getting network interfaces: %w", err)
	}

	for _, iface := range ifaces {
		var ifaceAddrs []net.Addr
		ifaceAddrs, err = iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("getting addresses for %q: %w", iface.Name, err)
		}

		for _, addr := range ifaceAddrs {
			cidr := addr.String()
			var ip net.IP
			ip, _, err = net.ParseCIDR(cidr)
			if err != nil {
				return nil, fmt.Errorf("parsing cidr: %w", err)
			}

			addrs = append(addrs, ip.String())
		}
	}

	return addrs, nil
}
