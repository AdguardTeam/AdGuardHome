package dhcpd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/AdguardTeam/golibs/file"

	"github.com/AdguardTeam/golibs/log"
)

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

	return false, fmt.Errorf("cannot check if IP is static: not supported on %s", runtime.GOOS)
}

// Set a static IP for the specified network interface
func SetStaticIP(ifaceName string) error {
	if runtime.GOOS == "linux" {
		return setStaticIPDhcpdConf(ifaceName)
	}

	if runtime.GOOS == "darwin" {
		return setStaticIPDarwin(ifaceName)
	}

	return fmt.Errorf("cannot set static IP on %s", runtime.GOOS)
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

// setStaticIPDhcpdConf - updates /etc/dhcpd.conf and sets the current IP address to be static
func setStaticIPDhcpdConf(ifaceName string) error {
	ip := util.GetSubnet(ifaceName)
	if len(ip) == 0 {
		return errors.New("can't get IP address")
	}

	ip4, _, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	gatewayIP := getGatewayIP(ifaceName)
	add := updateStaticIPDhcpcdConf(ifaceName, ip, gatewayIP, ip4.String())

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

// updates dhcpd.conf content -- sets static IP address there
// for dhcpcd.conf
func updateStaticIPDhcpcdConf(ifaceName, ip, gatewayIP, dnsIP string) string {
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

// Check if network interface has a static IP configured
// Supports: MacOS.
func hasStaticIPDarwin(ifaceName string) (bool, error) {
	portInfo, err := getCurrentHardwarePortInfo(ifaceName)
	if err != nil {
		return false, err
	}

	return portInfo.static, nil
}

// setStaticIPDarwin - uses networksetup util to set the current IP address to be static
// Additionally it configures the current DNS servers as well
func setStaticIPDarwin(ifaceName string) error {
	portInfo, err := getCurrentHardwarePortInfo(ifaceName)
	if err != nil {
		return err
	}

	if portInfo.static {
		return errors.New("IP address is already static")
	}

	dnsAddrs, err := getEtcResolvConfServers()
	if err != nil {
		return err
	}

	args := make([]string, 0)
	args = append(args, "-setdnsservers", portInfo.name)
	args = append(args, dnsAddrs...)

	// Setting DNS servers is necessary when configuring a static IP
	code, _, err := util.RunCommand("networksetup", args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("failed to set DNS servers, code=%d", code)
	}

	// Actually configures hardware port to have static IP
	code, _, err = util.RunCommand("networksetup", "-setmanual",
		portInfo.name, portInfo.ip, portInfo.subnet, portInfo.gatewayIP)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("failed to set DNS servers, code=%d", code)
	}

	return nil
}

// getCurrentHardwarePortInfo gets information the specified network interface
func getCurrentHardwarePortInfo(ifaceName string) (hardwarePortInfo, error) {
	// First of all we should find hardware port name
	m := getNetworkSetupHardwareReports()
	hardwarePort, ok := m[ifaceName]
	if !ok {
		return hardwarePortInfo{}, fmt.Errorf("could not find hardware port for %s", ifaceName)
	}

	return getHardwarePortInfo(hardwarePort)
}

// getNetworkSetupHardwareReports parses the output of the `networksetup -listallhardwareports` command
// it returns a map where the key is the interface name, and the value is the "hardware port"
// returns nil if it fails to parse the output
func getNetworkSetupHardwareReports() map[string]string {
	_, out, err := util.RunCommand("networksetup", "-listallhardwareports")
	if err != nil {
		return nil
	}

	re, err := regexp.Compile("Hardware Port: (.*?)\nDevice: (.*?)\n")
	if err != nil {
		return nil
	}

	m := make(map[string]string, 0)

	matches := re.FindAllStringSubmatch(out, -1)
	for i := range matches {
		port := matches[i][1]
		device := matches[i][2]
		m[device] = port
	}

	return m
}

// hardwarePortInfo - information obtained using MacOS networksetup
// about the current state of the internet connection
type hardwarePortInfo struct {
	name      string
	ip        string
	subnet    string
	gatewayIP string
	static    bool
}

func getHardwarePortInfo(hardwarePort string) (hardwarePortInfo, error) {
	h := hardwarePortInfo{}

	_, out, err := util.RunCommand("networksetup", "-getinfo", hardwarePort)
	if err != nil {
		return h, err
	}

	re := regexp.MustCompile("IP address: (.*?)\nSubnet mask: (.*?)\nRouter: (.*?)\n")

	match := re.FindStringSubmatch(out)
	if len(match) == 0 {
		return h, errors.New("could not find hardware port info")
	}

	h.name = hardwarePort
	h.ip = match[1]
	h.subnet = match[2]
	h.gatewayIP = match[3]

	if strings.Index(out, "Manual Configuration") == 0 {
		h.static = true
	}

	return h, nil
}

// Gets a list of nameservers currently configured in the /etc/resolv.conf
func getEtcResolvConfServers() ([]string, error) {
	body, err := ioutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile("nameserver ([a-zA-Z0-9.:]+)")

	matches := re.FindAllStringSubmatch(string(body), -1)
	if len(matches) == 0 {
		return nil, errors.New("found no DNS servers in /etc/resolv.conf")
	}

	addrs := make([]string, 0)
	for i := range matches {
		addrs = append(addrs, matches[i][1])
	}

	return addrs, nil
}
