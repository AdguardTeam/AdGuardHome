// Package aghnet contains some utilities for networking.
package aghnet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/osutil"
)

// DialContextFunc is the semantic alias for dialing functions, such as
// [http.Transport.DialContext].
type DialContextFunc = func(ctx context.Context, network, addr string) (conn net.Conn, err error)

// Variables and functions to substitute in tests.
var (
	// aghosRunCommand is the function to run shell commands.
	aghosRunCommand = aghos.RunCommand

	// netInterfaces is the function to get the available network interfaces.
	netInterfaceAddrs = net.InterfaceAddrs

	// rootDirFS is the filesystem pointing to the root directory.
	rootDirFS = osutil.RootDirFS()
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
func GatewayIP(ifaceName string) (ip netip.Addr) {
	code, out, err := aghosRunCommand("ip", "route", "show", "dev", ifaceName)
	if err != nil {
		log.Debug("%s", err)

		return netip.Addr{}
	} else if code != 0 {
		log.Debug("fetching gateway ip: unexpected exit code: %d", code)

		return netip.Addr{}
	}

	fields := bytes.Fields(out)
	// The meaningful "ip route" command output should contain the word
	// "default" at first field and default gateway IP address at third field.
	if len(fields) < 3 || string(fields[0]) != "default" {
		return netip.Addr{}
	}

	ip, err = netip.ParseAddr(string(fields[2]))
	if err != nil {
		return netip.Addr{}
	}

	return ip
}

// CanBindPrivilegedPorts checks if current process can bind to privileged
// ports.
func CanBindPrivilegedPorts() (can bool, err error) {
	return canBindPrivilegedPorts()
}

// NetInterface represents an entry of network interfaces map.
type NetInterface struct {
	// Addresses are the network interface addresses.
	Addresses []netip.Addr `json:"ip_addresses,omitempty"`
	// Subnets are the IP networks for this network interface.
	Subnets      []netip.Prefix   `json:"-"`
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

func NetInterfaceFrom(iface *net.Interface) (niface *NetInterface, err error) {
	niface = &NetInterface{
		Name:         iface.Name,
		HardwareAddr: iface.HardwareAddr,
		Flags:        iface.Flags,
		MTU:          iface.MTU,
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
	}

	// Collect network interface addresses.
	for _, addr := range addrs {
		n, ok := addr.(*net.IPNet)
		if !ok {
			// Should be *net.IPNet, this is weird.
			return nil, fmt.Errorf("expected %[2]s to be %[1]T, got %[2]T", n, addr)
		} else if ip4 := n.IP.To4(); ip4 != nil {
			n.IP = ip4
		}

		ip, ok := netip.AddrFromSlice(n.IP)
		if !ok {
			return nil, fmt.Errorf("bad address %s", n.IP)
		}

		ip = ip.Unmap()
		if ip.IsLinkLocalUnicast() {
			// Ignore link-local IPv4.
			if ip.Is4() {
				continue
			}

			ip = ip.WithZone(iface.Name)
		}

		ones, _ := n.Mask.Size()
		p := netip.PrefixFrom(ip, ones)

		niface.Addresses = append(niface.Addresses, ip)
		niface.Subnets = append(niface.Subnets, p)
	}

	return niface, nil
}

// GetValidNetInterfacesForWeb returns interfaces that are eligible for DNS and
// WEB only we do not return link-local addresses here.
//
// TODO(e.burkov):  Can't properly test the function since it's nontrivial to
// substitute net.Interface.Addrs and the net.InterfaceAddrs can't be used.
func GetValidNetInterfacesForWeb() (nifaces []*NetInterface, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("getting interfaces: %w", err)
	} else if len(ifaces) == 0 {
		return nil, errors.Error("no legible interfaces")
	}

	for i := range ifaces {
		var niface *NetInterface
		niface, err = NetInterfaceFrom(&ifaces[i])
		if err != nil {
			return nil, err
		} else if len(niface.Addresses) != 0 {
			// Discard interfaces with no addresses.
			nifaces = append(nifaces, niface)
		}
	}

	return nifaces, nil
}

// InterfaceByIP returns the name of the interface bound to ip.
//
// TODO(a.garipov, e.burkov): This function is technically incorrect, since one
// IP address can be shared by multiple interfaces in some configurations.
//
// TODO(e.burkov):  See TODO on GetValidNetInterfacesForWeb.
func InterfaceByIP(ip netip.Addr) (ifaceName string) {
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

// GetSubnet returns the subnet corresponding to the interface of zero prefix if
// the search fails.
//
// TODO(e.burkov):  See TODO on GetValidNetInterfacesForWeb.
func GetSubnet(ifaceName string) (p netip.Prefix) {
	netIfaces, err := GetValidNetInterfacesForWeb()
	if err != nil {
		log.Error("Could not get network interfaces info: %v", err)

		return p
	}

	for _, netIface := range netIfaces {
		if netIface.Name == ifaceName && len(netIface.Subnets) > 0 {
			return netIface.Subnets[0]
		}
	}

	return p
}

// CheckPort checks if the port is available for binding.  network is expected
// to be one of "udp" and "tcp".
func CheckPort(network string, ipp netip.AddrPort) (err error) {
	var c io.Closer
	addr := ipp.String()
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
func CollectAllIfacesAddrs() (addrs []netip.Addr, err error) {
	var ifaceAddrs []net.Addr
	ifaceAddrs, err = netInterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("getting interfaces addresses: %w", err)
	}

	for _, addr := range ifaceAddrs {
		var p netip.Prefix
		p, err = netip.ParsePrefix(addr.String())
		if err != nil {
			// Don't wrap the error since it's informative enough as is.
			return nil, err
		}

		addrs = append(addrs, p.Addr())
	}

	return addrs, nil
}

// ParseAddrPort parses an [netip.AddrPort] from s, which should be either a
// valid IP, optionally with port, or a valid URL with plain IP address.  The
// defaultPort is used if s doesn't contain port number.
func ParseAddrPort(s string, defaultPort uint16) (ipp netip.AddrPort, err error) {
	u, err := url.Parse(s)
	if err == nil && u.Host != "" {
		s = u.Host
	}

	ipp, err = netip.ParseAddrPort(s)
	if err != nil {
		ip, parseErr := netip.ParseAddr(s)
		if parseErr != nil {
			return ipp, errors.Join(err, parseErr)
		}

		return netip.AddrPortFrom(ip, defaultPort), nil
	}

	return ipp, nil
}

// ParseSubnet parses s either as a CIDR prefix itself, or as an IP address,
// returning the corresponding single-IP CIDR prefix.
//
// TODO(e.burkov):  Taken from dnsproxy, move to golibs.
func ParseSubnet(s string) (p netip.Prefix, err error) {
	if strings.Contains(s, "/") {
		p, err = netip.ParsePrefix(s)
		if err != nil {
			return netip.Prefix{}, err
		}
	} else {
		var ip netip.Addr
		ip, err = netip.ParseAddr(s)
		if err != nil {
			return netip.Prefix{}, err
		}

		p = netip.PrefixFrom(ip, ip.BitLen())
	}

	return p, nil
}

// ParseBootstraps returns the slice of upstream resolvers parsed from addrs.
// It additionally returns the closers for each resolver, that should be closed
// after use.
func ParseBootstraps(
	addrs []string,
	opts *upstream.Options,
) (boots []*upstream.UpstreamResolver, err error) {
	boots = make([]*upstream.UpstreamResolver, 0, len(boots))
	for i, b := range addrs {
		var r *upstream.UpstreamResolver
		r, err = upstream.NewUpstreamResolver(b, opts)
		if err != nil {
			return nil, fmt.Errorf("bootstrap at index %d: %w", i, err)
		}

		boots = append(boots, r)
	}

	return boots, nil
}

// BroadcastFromPref calculates the broadcast IP address for p.
func BroadcastFromPref(p netip.Prefix) (bc netip.Addr) {
	bc = p.Addr().Unmap()
	if !bc.IsValid() {
		return netip.Addr{}
	}

	maskLen, addrLen := p.Bits(), bc.BitLen()
	if maskLen == addrLen {
		return bc
	}

	ipBytes := bc.AsSlice()
	for i := maskLen; i < addrLen; i++ {
		ipBytes[i/8] |= 1 << (7 - (i % 8))
	}
	bc, _ = netip.AddrFromSlice(ipBytes)

	return bc
}
