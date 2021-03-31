// +build darwin

package aghnet

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

// hardwarePortInfo - information obtained using MacOS networksetup
// about the current state of the internet connection
type hardwarePortInfo struct {
	name      string
	ip        string
	subnet    string
	gatewayIP string
	static    bool
}

func ifaceHasStaticIP(ifaceName string) (bool, error) {
	portInfo, err := getCurrentHardwarePortInfo(ifaceName)
	if err != nil {
		return false, err
	}

	return portInfo.static, nil
}

// getCurrentHardwarePortInfo gets information for the specified network interface.
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
	_, out, err := aghos.RunCommand("networksetup", "-listallhardwareports")
	if err != nil {
		return nil
	}

	re, err := regexp.Compile("Hardware Port: (.*?)\nDevice: (.*?)\n")
	if err != nil {
		return nil
	}

	m := make(map[string]string)

	matches := re.FindAllStringSubmatch(out, -1)
	for i := range matches {
		port := matches[i][1]
		device := matches[i][2]
		m[device] = port
	}

	return m
}

func getHardwarePortInfo(hardwarePort string) (hardwarePortInfo, error) {
	h := hardwarePortInfo{}

	_, out, err := aghos.RunCommand("networksetup", "-getinfo", hardwarePort)
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

func ifaceSetStaticIP(ifaceName string) (err error) {
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
	code, _, err := aghos.RunCommand("networksetup", args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("failed to set DNS servers, code=%d", code)
	}

	// Actually configures hardware port to have static IP
	code, _, err = aghos.RunCommand("networksetup", "-setmanual",
		portInfo.name, portInfo.ip, portInfo.subnet, portInfo.gatewayIP)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("failed to set DNS servers, code=%d", code)
	}

	return nil
}

// getEtcResolvConfServers returns a list of nameservers configured in
// /etc/resolv.conf.
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
