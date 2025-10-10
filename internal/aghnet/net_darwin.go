//go:build darwin

package aghnet

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
)

// hardwarePortInfo contains information about the current state of the internet
// connection obtained from macOS networksetup.
type hardwarePortInfo struct {
	name      string
	ip        string
	subnet    string
	gatewayIP string
	static    bool
}

// ifaceHasStaticIP reports whether ifaceName is configured with a static IP.
// cmdCons must not be nil.
func ifaceHasStaticIP(
	ctx context.Context,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (ok bool, err error) {
	portInfo, err := getCurrentHardwarePortInfo(ctx, cmdCons, ifaceName)
	if err != nil {
		return false, err
	}

	return portInfo.static, nil
}

// getCurrentHardwarePortInfo returns information for the specified network
// interface.  cmdCons must not be nil.
func getCurrentHardwarePortInfo(
	ctx context.Context,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (hardwarePortInfo, error) {
	// First, find the hardware port name.
	m := getNetworkSetupHardwareReports(ctx, cmdCons)
	hardwarePort, ok := m[ifaceName]
	if !ok {
		return hardwarePortInfo{}, fmt.Errorf("could not find hardware port for %s", ifaceName)
	}

	return getHardwarePortInfo(ctx, cmdCons, hardwarePort)
}

// hardwareReportsReg is the regular expression matching the lines of
// networksetup command output lines containing the interface information.
var hardwareReportsReg = regexp.MustCompile("Hardware Port: (.*?)\nDevice: (.*?)\n")

// getNetworkSetupHardwareReports returns a map of interface names to hardware
// port names.  It returns nil if parsing fails.  cmdCons must not be nil.
//
// TODO(e.burkov):  There should be more proper approach than parsing the
// command output.  For example, see
// https://developer.apple.com/documentation/systemconfiguration.
func getNetworkSetupHardwareReports(
	ctx context.Context,
	cmdCons executil.CommandConstructor,
) (reports map[string]string) {
	_, out, err := aghos.RunCommand(ctx, cmdCons, "networksetup", "-listallhardwareports")
	if err != nil {
		return nil
	}

	reports = make(map[string]string)

	matches := hardwareReportsReg.FindAllSubmatch(out, -1)
	for _, m := range matches {
		reports[string(m[2])] = string(m[1])
	}

	return reports
}

// hardwarePortReg is the regular expression matching the lines of networksetup
// command output lines containing the port information.
var hardwarePortReg = regexp.MustCompile("IP address: (.*?)\nSubnet mask: (.*?)\nRouter: (.*?)\n")

// getHardwarePortInfo returns IP, subnet, gateway, and static/dynamic status
// for the given hardware port.  cmdCons must not be nil.
func getHardwarePortInfo(
	ctx context.Context,
	cmdCons executil.CommandConstructor,
	hardwarePort string,
) (h hardwarePortInfo, err error) {
	_, out, err := aghos.RunCommand(ctx, cmdCons, "networksetup", "-getinfo", hardwarePort)
	if err != nil {
		return h, err
	}

	match := hardwarePortReg.FindSubmatch(out)
	if len(match) != 4 {
		return h, errors.Error("could not find hardware port info")
	}

	return hardwarePortInfo{
		name:      hardwarePort,
		ip:        string(match[1]),
		subnet:    string(match[2]),
		gatewayIP: string(match[3]),
		static:    bytes.Index(out, []byte("Manual Configuration")) == 0,
	}, nil
}

// ifaceSetStaticIP sets a static IP on ifaceName.  cmdCons must not be nil.
func ifaceSetStaticIP(
	ctx context.Context,
	_ *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (err error) {
	portInfo, err := getCurrentHardwarePortInfo(ctx, cmdCons, ifaceName)
	if err != nil {
		return err
	}

	if portInfo.static {
		return errors.Error("ip address is already static")
	}

	dnsAddrs, err := getEtcResolvConfServers()
	if err != nil {
		return err
	}

	args := append([]string{"-setdnsservers", portInfo.name}, dnsAddrs...)

	// Setting DNS servers is necessary when configuring a static IP
	code, _, err := aghos.RunCommand(ctx, cmdCons, "networksetup", args...)
	if err != nil {
		return err
	} else if code != 0 {
		return fmt.Errorf("failed to set DNS servers, code=%d", code)
	}

	// Actually configures hardware port to have static IP
	code, _, err = aghos.RunCommand(
		ctx,
		cmdCons,
		"networksetup",
		"-setmanual",
		portInfo.name,
		portInfo.ip,
		portInfo.subnet,
		portInfo.gatewayIP,
	)
	if err != nil {
		return err
	} else if code != 0 {
		return fmt.Errorf("failed to set DNS servers, code=%d", code)
	}

	return nil
}

// etcResolvConfReg is the regular expression matching the lines of resolv.conf
// file containing a name server information.
var etcResolvConfReg = regexp.MustCompile("nameserver ([a-zA-Z0-9.:]+)")

// getEtcResolvConfServers returns a list of nameservers configured in
// /etc/resolv.conf.
func getEtcResolvConfServers() (addrs []string, err error) {
	const filename = "etc/resolv.conf"

	_, err = aghos.FileWalker(func(r io.Reader) (_ []string, _ bool, err error) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			matches := etcResolvConfReg.FindAllStringSubmatch(sc.Text(), -1)
			if len(matches) == 0 {
				continue
			}

			for _, m := range matches {
				addrs = append(addrs, m[1])
			}
		}

		return nil, false, sc.Err()
	}).Walk(rootDirFS, filename)
	if err != nil {
		return nil, fmt.Errorf("parsing etc/resolv.conf file: %w", err)
	} else if len(addrs) == 0 {
		return nil, fmt.Errorf("found no dns servers in %s", filename)
	}

	return addrs, nil
}
