package sysutil

import (
	"net"
	"os/exec"
	"strings"

	"github.com/AdguardTeam/golibs/log"
)

// IfaceHasStaticIP checks if interface is configured to have static IP address.
func IfaceHasStaticIP(ifaceName string) (has bool, err error) {
	return ifaceHasStaticIP(ifaceName)
}

// IfaceSetStaticIP sets static IP address for network interface.
func IfaceSetStaticIP(ifaceName string) (err error) {
	return ifaceSetStaticIP(ifaceName)
}

// GatewayIP returns IP address of interface's gateway.
func GatewayIP(ifaceName string) string {
	cmd := exec.Command("ip", "route", "show", "dev", ifaceName)
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	d, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return ""
	}

	fields := strings.Fields(string(d))
	// The meaningful "ip route" command output should contain the word
	// "default" at first field and default gateway IP address at third
	// field.
	if len(fields) < 3 || fields[0] != "default" {
		return ""
	}

	ip := net.ParseIP(fields[2])
	if ip == nil {
		return ""
	}

	return fields[2]
}
