// Package aghnet contains some utilities for networking.
package aghnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/golibs/log"
)

// ErrNoStaticIPInfo is returned by IfaceHasStaticIP when no information about
// the IP being static is available.
const ErrNoStaticIPInfo agherr.Error = "no information about static ip"

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

// GetValidNetInterfacesForWeb returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func GetValidNetInterfacesForWeb() ([]*NetInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("couldn't get interfaces: %w", err)
	}
	if len(ifaces) == 0 {
		return nil, errors.New("couldn't find any legible interface")
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

// TODO(e.burkov): Inspect the charToHex, ipParseARPA6, ipReverse and
// UnreverseAddr and maybe refactor it.

// charToHex converts character to a hexadecimal.
func charToHex(n byte) int8 {
	if n >= '0' && n <= '9' {
		return int8(n) - '0'
	} else if (n|0x20) >= 'a' && (n|0x20) <= 'f' {
		return (int8(n) | 0x20) - 'a' + 10
	}
	return -1
}

// ipParseARPA6 parse IPv6 reverse address
func ipParseARPA6(s string) (ip6 net.IP) {
	if len(s) != 63 {
		return nil
	}

	ip6 = make(net.IP, 16)

	for i := 0; i != 64; i += 4 {
		// parse "0.1."
		n := charToHex(s[i])
		n2 := charToHex(s[i+2])
		if s[i+1] != '.' || (i != 60 && s[i+3] != '.') ||
			n < 0 || n2 < 0 {
			return nil
		}

		ip6[16-i/4-1] = byte(n2<<4) | byte(n&0x0f)
	}
	return ip6
}

// ipReverse inverts byte order of ip.
func ipReverse(ip net.IP) (rev net.IP) {
	ipLen := len(ip)
	rev = make(net.IP, ipLen)
	for i, b := range ip {
		rev[ipLen-i-1] = b
	}

	return rev
}

// ARPA addresses' suffixes.
const (
	arpaV4Suffix = ".in-addr.arpa"
	arpaV6Suffix = ".ip6.arpa"
)

// UnreverseAddr tries to convert reversed ARPA to a normal IP address.
func UnreverseAddr(arpa string) (unreversed net.IP) {
	// Unify the input data.
	arpa = strings.TrimSuffix(arpa, ".")
	arpa = strings.ToLower(arpa)

	if strings.HasSuffix(arpa, arpaV4Suffix) {
		ip := strings.TrimSuffix(arpa, arpaV4Suffix)
		ip4 := net.ParseIP(ip).To4()
		if ip4 == nil {
			return nil
		}

		return ipReverse(ip4)

	} else if strings.HasSuffix(arpa, arpaV6Suffix) {
		ip := strings.TrimSuffix(arpa, arpaV6Suffix)
		return ipParseARPA6(ip)
	}

	// The suffix unrecognizable.
	return nil
}

// The length of extreme cases of arpa formatted addresses.
//
// The example of IPv4 with maximum length:
//
//   49.91.20.104.in-addr.arpa
//
// The example of IPv6 with maximum length:
//
//   1.3.b.5.4.1.8.6.0.0.0.0.0.0.0.0.0.0.0.0.0.1.0.0.0.0.7.4.6.0.6.2.ip6.arpa
//
const (
	arpaV4MaxLen = len("000.000.000.000") + len(arpaV4Suffix)
	arpaV6MaxLen = len("0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0") +
		len(arpaV6Suffix)
)

// ReverseAddr returns the ARPA hostname of the ip suitable for reverse DNS
// (PTR) record lookups.  This is the modified version of ReverseAddr from
// github.com/miekg/dns package with no error among returned values.
func ReverseAddr(ip net.IP) (arpa string) {
	var strLen int
	var suffix string
	// Don't handle errors in implementations since strings.WriteString
	// never returns non-nil errors.
	var writeByte func(val byte)
	b := &strings.Builder{}
	if ip4 := ip.To4(); ip4 != nil {
		strLen, suffix = arpaV4MaxLen, arpaV4Suffix[1:]
		ip = ip4
		writeByte = func(val byte) {
			_, _ = b.WriteString(strconv.Itoa(int(val)))
			_, _ = b.WriteRune('.')
		}

	} else if ip6 := ip.To16(); ip6 != nil {
		strLen, suffix = arpaV6MaxLen, arpaV6Suffix[1:]
		ip = ip6
		writeByte = func(val byte) {
			lByte, rByte := val&0xF, val>>4

			_, _ = b.WriteString(strconv.FormatUint(uint64(lByte), 16))
			_, _ = b.WriteRune('.')
			_, _ = b.WriteString(strconv.FormatUint(uint64(rByte), 16))
			_, _ = b.WriteRune('.')
		}

	} else {
		return ""
	}

	b.Grow(strLen)
	for i := len(ip) - 1; i >= 0; i-- {
		writeByte(ip[i])
	}
	_, _ = b.WriteString(suffix)

	return b.String()
}
