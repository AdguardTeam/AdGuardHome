// Package aghnet contains some utilities for networking.
package aghnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
)

// Variables and functions to substitute in tests.
var (
	// aghosRunCommand is the function to run shell commands.
	aghosRunCommand = aghos.RunCommand

	// netInterfaces is the function to get the available network interfaces.
	netInterfaceAddrs = net.InterfaceAddrs

	// rootDirFS is the filesystem pointing to the root directory.
	rootDirFS = aghos.RootDirFS()
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
//
// TODO(e.burkov):  Investigate if the gateway address may be fetched in another
// way since not every machine has the software installed.
func GatewayIP(ifaceName string) (ip net.IP) {
	code, out, err := aghosRunCommand("ip", "route", "show", "dev", ifaceName)
	if err != nil {
		log.Debug("%s", err)

		return nil
	} else if code != 0 {
		log.Debug("fetching gateway ip: unexpected exit code: %d", code)

		return nil
	}

	fields := bytes.Fields(out)
	// The meaningful "ip route" command output should contain the word
	// "default" at first field and default gateway IP address at third field.
	if len(fields) < 3 || string(fields[0]) != "default" {
		return nil
	}

	return net.ParseIP(string(fields[2]))
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

// MarshalText implements the json.Marshaler interface for NetInterface.
func (iface NetInterface) MarshalText() ([]byte, error) {
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

// GetValidNetInterfacesForWeb returns interfaces that are eligible for DNS and
// WEB only we do not return link-local addresses here.
//
// TODO(e.burkov):  Can't properly test the function since it's nontrivial to
// substitute net.Interface.Addrs and the net.InterfaceAddrs can't be used.
func GetValidNetInterfacesForWeb() (netIfaces []*NetInterface, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("couldn't get interfaces: %w", err)
	} else if len(ifaces) == 0 {
		return nil, errors.Error("couldn't find any legible interface")
	}

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
				return nil, fmt.Errorf("got %s that is not net.IPNet, it is %T", addr, addr)
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
			netIfaces = append(netIfaces, netIface)
		}
	}

	return netIfaces, nil
}

// GetInterfaceByIP returns the name of interface containing provided ip.
//
// TODO(e.burkov):  See TODO on GetValidInterfacesForWeb.
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
//
// TODO(e.burkov):  See TODO on GetValidInterfacesForWeb.
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

// CheckPort checks if the port is available for binding.  network is expected
// to be one of "udp" and "tcp".
func CheckPort(network string, ip net.IP, port int) (err error) {
	var c io.Closer
	addr := netutil.IPPort{IP: ip, Port: port}.String()
	switch network {
	case "tcp":
		c, err = net.Listen(network, addr)
	case "udp":
		c, err = net.ListenPacket(network, addr)
	default:
		return nil
	}

	if err != nil {
		return err
	}

	return closePortChecker(c)
}

// IsAddrInUse checks if err is about unsuccessful address binding.
func IsAddrInUse(err error) (ok bool) {
	var sysErr syscall.Errno
	if !errors.As(err, &sysErr) {
		return false
	}

	return isAddrInUse(sysErr)
}

// CollectAllIfacesAddrs returns the slice of all network interfaces IP
// addresses without port number.
func CollectAllIfacesAddrs() (addrs []string, err error) {
	var ifaceAddrs []net.Addr
	ifaceAddrs, err = netInterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("getting interfaces addresses: %w", err)
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

	return addrs, nil
}

// BroadcastFromIPNet calculates the broadcast IP address for n.
func BroadcastFromIPNet(n *net.IPNet) (dc net.IP) {
	dc = netutil.CloneIP(n.IP)

	mask := n.Mask
	if mask == nil {
		mask = dc.DefaultMask()
	}

	for i, b := range mask {
		dc[i] |= ^b
	}

	return dc
}
