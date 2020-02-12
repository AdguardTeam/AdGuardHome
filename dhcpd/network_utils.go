package dhcpd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AdguardTeam/golibs/file"

	"github.com/AdguardTeam/golibs/log"
)

// GetValidNetInterfaces returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is a ppp interface or the one that doesn't allow broadcasts
func GetValidNetInterfaces() ([]net.Interface, error) {
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

// Check if network interface has a static IP configured
// Supports: Raspbian.
func HasStaticIP(ifaceName string) (bool, error) {
	if runtime.GOOS == "linux" {
		body, err := ioutil.ReadFile("/etc/dhcpcd.conf")
		if err != nil {
			return false, err
		}

		return hasStaticIPDhcpcdConf(string(body), ifaceName), nil
	}

	if runtime.GOOS == "darwin" {
		return hasStaticIPDarwin(ifaceName)
	}

	return false, fmt.Errorf("Cannot check if IP is static: not supported on %s", runtime.GOOS)
}

// Get IP address with netmask
func GetFullIP(ifaceName string) string {
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

// Set a static IP for network interface
// Supports: Raspbian.
func SetStaticIP(ifaceName string) error {
	ip := GetFullIP(ifaceName)
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
func hasStaticIPDhcpcdConf(dhcpConf, ifaceName string) bool {
	lines := strings.Split(dhcpConf, "\n")
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
