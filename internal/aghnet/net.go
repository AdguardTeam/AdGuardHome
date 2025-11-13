// Package aghnet contains networking utilities.
package aghnet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"syscall"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
)

// DialContextFunc is the semantic alias for dialing functions, such as
// [http.Transport.DialContext].
type DialContextFunc = func(ctx context.Context, network, addr string) (conn net.Conn, err error)

// Variables and functions to substitute in tests.
var (
	// netInterfaceAddrs is the function to get the available network
	// interfaces.
	netInterfaceAddrs = net.InterfaceAddrs

	// rootDirFS is the filesystem pointing to the root directory.
	rootDirFS = osutil.RootDirFS()
)

// ErrNoStaticIPInfo is returned by IfaceHasStaticIP when no information about
// whether the IP is static is available.
const ErrNoStaticIPInfo errors.Error = "no information about static ip"

// IfaceHasStaticIP reports whether the interface has a static IP.  If the
// status is indeterminate, it returns false with an error matching
// [ErrNoStaticIPInfo].  cmdCons must not be nil.
func IfaceHasStaticIP(
	ctx context.Context,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (has bool, err error) {
	return ifaceHasStaticIP(ctx, cmdCons, ifaceName)
}

// IfaceSetStaticIP sets a static IP address for network interface.  l and
// cmdCons must not be nil.
func IfaceSetStaticIP(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (err error) {
	return ifaceSetStaticIP(ctx, l, cmdCons, ifaceName)
}

// GatewayIP returns the gateway IP address for the interface.  l and cmdCons
// must not be nil.
//
// TODO(e.burkov):  Investigate if the gateway address may be fetched in another
// way since not every machine has the software installed.
func GatewayIP(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (ip netip.Addr) {
	stdout := bytes.Buffer{}
	err := executil.Run(
		ctx,
		cmdCons,
		&executil.CommandConfig{
			Path:   "ip",
			Args:   []string{"route", "show", "dev", ifaceName},
			Stdout: &stdout,
		},
	)
	if err != nil {
		if code, ok := executil.ExitCodeFromError(err); ok {
			err = fmt.Errorf("unexpected exit code %d: %w", code, err)
		}

		l.DebugContext(ctx, "fetching gateway ip", slogutil.KeyError, err)

		return netip.Addr{}
	}

	fields := bytes.Fields(stdout.Bytes())
	if len(fields) < 3 || !bytes.Equal(fields[0], []byte("default")) {
		return netip.Addr{}
	}

	if err = ip.UnmarshalText(fields[2]); err != nil {
		return netip.Addr{}
	}

	return ip
}

// CanBindPrivilegedPorts checks if current process can bind to privileged
// ports.  l must not be nil.
func CanBindPrivilegedPorts(ctx context.Context, l *slog.Logger) (can bool, err error) {
	return canBindPrivilegedPorts(ctx, l)
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

// NetInterfaceFrom converts a [net.Interface] to [NetInterface], populating
// name, MAC address, flags, MTU, IP addresses, and subnets.  iface must not be
// nil.
func NetInterfaceFrom(iface *net.Interface) (niface *NetInterface, err error) {
	niface = &NetInterface{
		Name:         iface.Name,
		HardwareAddr: iface.HardwareAddr,
		Flags:        iface.Flags,
		MTU:          iface.MTU,
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("getting addresses for interface %q: %w", iface.Name, err)
	}

	for i, addr := range addrs {
		if err = populateAddrs(addr, niface); err != nil {
			return nil, fmt.Errorf("populating at index %d: %w", i, err)
		}
	}

	return niface, nil
}

// populateAddrs fills *NetInterface IP addresses and subnets.  addr and niface
// must not be nil.
func populateAddrs(addr net.Addr, niface *NetInterface) (err error) {
	n, err := ipNetFromAddr(addr)
	if err != nil {
		return err
	}

	ip, ok := netip.AddrFromSlice(n.IP)
	if !ok {
		return fmt.Errorf("bad address %s", n.IP)
	}

	ip = ip.Unmap()

	// Skip link-local IPv4 addresses
	if isLinkLocalV4(ip) {
		return nil
	}

	if ip.IsLinkLocalUnicast() {
		ip = ip.WithZone(niface.Name)
	}

	ones, _ := n.Mask.Size()
	p := netip.PrefixFrom(ip, ones)

	niface.Addresses = append(niface.Addresses, ip)
	niface.Subnets = append(niface.Subnets, p)

	return nil
}

// ipNetFromAddr converts net.Addr to *net.IPNet and its IP to v4 if necessary.
func ipNetFromAddr(addr net.Addr) (ip *net.IPNet, err error) {
	ipNet, ok := addr.(*net.IPNet)
	if !ok {
		// Should be *net.IPNet, this is weird.
		return nil, fmt.Errorf("bad type for interface net.Addr %T(%[1]v)", ipNet)
	}

	// TODO(f.setrakov): Explore whether this logic can be safely removed.
	if ip4 := ipNet.IP.To4(); ip4 != nil {
		ipNet.IP = ip4
	}

	return ipNet, nil
}

// isLinkLocalV4 checks if ip is link-local unicast IPv4 address.
func isLinkLocalV4(ip netip.Addr) (ok bool) {
	return ip.Is4() && ip.IsLinkLocalUnicast()
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
// the search fails.  l must not be nil.
//
// TODO(e.burkov):  See TODO on GetValidNetInterfacesForWeb.
func GetSubnet(ctx context.Context, l *slog.Logger, ifaceName string) (p netip.Prefix) {
	netIfaces, err := GetValidNetInterfacesForWeb()
	if err != nil {
		l.ErrorContext(ctx, "could not get network interfaces info", slogutil.KeyError, err)

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
