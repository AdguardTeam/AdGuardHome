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
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
)

// networkSetupCmd is the command to configure network settings.
const networkSetupCmd = "networksetup"

// stderrPeekSize is the maximum number of bytes to read from the stderr of a
// failing command.
const stderrPeekSize = 1024

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
// l and cmdCons must not be nil.
func ifaceHasStaticIP(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (ok bool, err error) {
	portInfo, err := currentHardwarePortInfo(ctx, l, cmdCons, ifaceName)
	if err != nil {
		return false, err
	}

	return portInfo.static, nil
}

// currentHardwarePortInfo returns information for the specified network
// interface.  l and cmdCons must not be nil.
func currentHardwarePortInfo(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (i hardwarePortInfo, err error) {
	m := networkSetupHardwareReports(ctx, l, cmdCons)
	hardwarePort, ok := m[ifaceName]
	if !ok {
		return hardwarePortInfo{}, fmt.Errorf("could not find hardware port for %s", ifaceName)
	}

	return newHardwarePortInfo(ctx, cmdCons, hardwarePort)
}

// hardwareReportsReg is the regular expression matching the lines of
// networksetup command output lines containing the interface information.
var hardwareReportsReg = regexp.MustCompile("Hardware Port: (.*?)\nDevice: (.*?)\n")

// networkSetupHardwareReports returns a map of interface names to hardware port
// names.  It returns nil if parsing fails.  l and cmdCons must not be nil.
//
// TODO(e.burkov):  There should be more proper approach than parsing the
// command output.  For example, see
// https://developer.apple.com/documentation/systemconfiguration.
func networkSetupHardwareReports(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
) (reports map[string]string) {
	// TODO(a.garipov):  Consider a version of executil.RunWithPeek with
	// different limits for stderr and stdout.
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := executil.Run(ctx, cmdCons, &executil.CommandConfig{
		Stderr: ioutil.NewTruncatedWriter(stderr, stderrPeekSize),
		Stdout: ioutil.NewTruncatedWriter(stdout, aghos.MaxCmdOutputSize),
		Path:   networkSetupCmd,
		Args:   []string{"-listallhardwareports"},
	})
	if err != nil {
		l.WarnContext(
			ctx,
			"getting network setup",
			"stderr_peek", stderr,
			slogutil.KeyError, err,
		)

		return nil
	}

	reports = make(map[string]string)

	matches := hardwareReportsReg.FindAllSubmatch(stdout.Bytes(), -1)
	for _, m := range matches {
		reports[string(m[2])] = string(m[1])
	}

	return reports
}

// hardwarePortReg is the regular expression matching the lines of networksetup
// command output lines containing the port information.
var hardwarePortReg = regexp.MustCompile("IP address: (.*?)\nSubnet mask: (.*?)\nRouter: (.*?)\n")

// newHardwarePortInfo returns IP, subnet, gateway, and static/dynamic status
// for the given hardware port.  cmdCons must not be nil.
func newHardwarePortInfo(
	ctx context.Context,
	cmdCons executil.CommandConstructor,
	hardwarePort string,
) (h hardwarePortInfo, err error) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err = executil.Run(ctx, cmdCons, &executil.CommandConfig{
		Stderr: ioutil.NewTruncatedWriter(stderr, stderrPeekSize),
		Stdout: ioutil.NewTruncatedWriter(stdout, aghos.MaxCmdOutputSize),
		Path:   networkSetupCmd,
		Args:   []string{"-getinfo", hardwarePort},
	})
	if err != nil {
		return h, fmt.Errorf("%w; stderr peek: %q", err, stderr)
	}

	out := stdout.Bytes()
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

// ifaceSetStaticIP sets a static IP on ifaceName.  l and cmdCons must not be
// nil.
func ifaceSetStaticIP(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (err error) {
	portInfo, err := currentHardwarePortInfo(ctx, l, cmdCons, ifaceName)
	if err != nil {
		return err
	}

	if portInfo.static {
		return errors.Error("ip address is already static")
	}

	dnsAddrs, err := etcResolvConfServers()
	if err != nil {
		return err
	}

	args := append([]string{"-setdnsservers", portInfo.name}, dnsAddrs...)

	// Setting DNS servers is necessary when configuring a static IP.
	err = executil.RunWithPeek(ctx, cmdCons, aghos.MaxCmdOutputSize, networkSetupCmd, args...)
	if err != nil {
		return fmt.Errorf("setting dns service with networksetup: %w", err)
	}

	// Actually configures hardware port to have static IP.
	err = executil.RunWithPeek(
		ctx,
		cmdCons,
		aghos.MaxCmdOutputSize,
		networkSetupCmd,
		"-setmanual",
		portInfo.name,
		portInfo.ip,
		portInfo.subnet,
		portInfo.gatewayIP,
	)
	if err != nil {
		return fmt.Errorf("configuring static ip with networksetup: %w", err)
	}

	return nil
}

// etcResolvConfReg is the regular expression matching the lines of resolv.conf
// file containing a name server information.
var etcResolvConfReg = regexp.MustCompile("nameserver ([a-zA-Z0-9.:]+)")

// etcResolvConfServers returns the nameservers configured in /etc/resolv.conf.
func etcResolvConfServers() (addrs []string, err error) {
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
