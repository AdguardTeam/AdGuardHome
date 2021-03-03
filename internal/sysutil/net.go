package sysutil

import (
	"net"
	"os/exec"
	"strings"

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
