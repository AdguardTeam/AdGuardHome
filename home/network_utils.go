package home

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

type netInterface struct {
	Name         string
	MTU          int
	HardwareAddr string
	Addresses    []string
	Flags        string
}

// getValidNetInterfaces returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is a ppp interface or the one that doesn't allow broadcasts
func getValidNetInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get list of interfaces: %s", err)
	}

	netIfaces := []net.Interface{}

	for i := range ifaces {
		if ifaces[i].Flags&net.FlagPointToPoint != 0 {
			// this interface is ppp, we're not interested in this one
			continue
		}

		iface := ifaces[i]
		netIfaces = append(netIfaces, iface)
	}

	return netIfaces, nil
}

// getValidNetInterfacesMap returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func getValidNetInterfacesForWeb() ([]netInterface, error) {
	ifaces, err := getValidNetInterfaces()
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

// Check if network interface has a static IP configured
// Supports: Raspbian.
func hasStaticIP(ifaceName string) (bool, error) {
	if runtime.GOOS == "windows" {
		return false, errors.New("Can't detect static IP: not supported on Windows")
	}

	body, err := ioutil.ReadFile("/etc/dhcpcd.conf")
	if err != nil {
		return false, err
	}

	return hasStaticIPDhcpcdConf(string(body), ifaceName), nil
}

// for dhcpcd.conf
func hasStaticIPDhcpcdConf(data, ifaceName string) bool {
	lines := strings.Split(data, "\n")
	nameLine := fmt.Sprintf("interface %s", ifaceName)
	withinInterfaceCtx := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if withinInterfaceCtx && len(line) == 0 {
			// an empty line resets our state
			withinInterfaceCtx = false
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}
		line = strings.TrimSpace(line)

		if !withinInterfaceCtx {
			if line == nameLine {
				// we found our interface
				withinInterfaceCtx = true
			}

		} else {
			if strings.HasPrefix(line, "interface ") {
				// we found another interface - reset our state
				withinInterfaceCtx = false
				continue
			}
			if strings.HasPrefix(line, "static ip_address=") {
				return true
			}
		}
	}
	return false
}

// Get IP address with netmask
func getFullIP(ifaceName string) string {
	cmd := exec.Command("ip", "-oneline", "-family", "inet", "address", "show", ifaceName)
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	d, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return ""
	}

	fields := strings.Fields(string(d))
	if len(fields) < 4 {
		return ""
	}
	_, _, err = net.ParseCIDR(fields[3])
	if err != nil {
		return ""
	}

	return fields[3]
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

// Get gateway IP address
func getGatewayIP(ifaceName string) string {
	cmd := exec.Command("ip", "route", "show", "dev", ifaceName)
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	d, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return ""
	}

	fields := strings.Fields(string(d))
	if len(fields) < 3 || fields[0] != "default" {
		return ""
	}

	ip := net.ParseIP(fields[2])
	if ip == nil {
		return ""
	}

	return fields[2]
}

// Set a static IP for network interface
// Supports: Raspbian.
func setStaticIP(ifaceName string) error {
	ip := getFullIP(ifaceName)
	if len(ip) == 0 {
		return errors.New("Can't get IP address")
	}

	ip4, _, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	gatewayIP := getGatewayIP(ifaceName)
	add := setStaticIPDhcpcdConf(ifaceName, ip, gatewayIP, ip4.String())

	body, err := ioutil.ReadFile("/etc/dhcpcd.conf")
	if err != nil {
		return err
	}

	body = append(body, []byte(add)...)
	err = file.SafeWrite("/etc/dhcpcd.conf", body)
	if err != nil {
		return err
	}

	return nil
}

// for dhcpcd.conf
func setStaticIPDhcpcdConf(ifaceName, ip, gatewayIP, dnsIP string) string {
	var body []byte

	add := fmt.Sprintf("\ninterface %s\nstatic ip_address=%s\n",
		ifaceName, ip)
	body = append(body, []byte(add)...)

	if len(gatewayIP) != 0 {
		add = fmt.Sprintf("static routers=%s\n",
			gatewayIP)
		body = append(body, []byte(add)...)
	}

	add = fmt.Sprintf("static domain_name_servers=%s\n\n",
		dnsIP)
	body = append(body, []byte(add)...)

	return string(body)
}

// checkPortAvailable is not a cheap test to see if the port is bindable, because it's actually doing the bind momentarily
func checkPortAvailable(host string, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	ln.Close()

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
	ln.Close()

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
