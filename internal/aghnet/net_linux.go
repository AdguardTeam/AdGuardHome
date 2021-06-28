//go:build linux
// +build linux

package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/google/renameio/maybe"
	"golang.org/x/sys/unix"
)

// recurrentChecker is used to check all the files which may include references
// for other ones.
type recurrentChecker struct {
	// checker is the function to check if r's stream contains the desired
	// attribute.  It must return all the patterns for files which should
	// also be checked and each of them should be valid for filepath.Glob
	// function.
	checker func(r io.Reader, desired string) (patterns []string, has bool, err error)
	// initPath is the path of the first member in the sequence of checked
	// files.
	initPath string
}

// maxCheckedFileSize is the maximum length of the file that recurrentChecker
// may check.
const maxCheckedFileSize = 1024 * 1024

// checkFile tries to open and to check single file located on the sourcePath.
func (rc *recurrentChecker) checkFile(sourcePath, desired string) (
	subsources []string,
	has bool,
	err error,
) {
	var f *os.File
	f, err = os.Open(sourcePath)
	if err != nil {
		return nil, false, err
	}

	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	var r io.Reader
	r, err = aghio.LimitReader(f, maxCheckedFileSize)
	if err != nil {
		return nil, false, err
	}

	subsources, has, err = rc.checker(r, desired)
	if err != nil {
		return nil, false, err
	}

	if has {
		return nil, true, nil
	}

	return subsources, has, nil
}

// handlePatterns parses the patterns and takes care of duplicates.
func (rc *recurrentChecker) handlePatterns(sourcesSet *aghstrings.Set, patterns []string) (
	subsources []string,
	err error,
) {
	subsources = make([]string, 0, len(patterns))
	for _, p := range patterns {
		var matches []string
		matches, err = filepath.Glob(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}

		for _, m := range matches {
			if sourcesSet.Has(m) {
				continue
			}

			sourcesSet.Add(m)
			subsources = append(subsources, m)
		}
	}

	return subsources, nil
}

// check walks through all the files searching for the desired attribute.
func (rc *recurrentChecker) check(desired string) (has bool, err error) {
	var i int
	sources := []string{rc.initPath}

	defer func() {
		if i >= len(sources) {
			return
		}

		err = errors.Annotate(err, "checking %q: %w", sources[i])
	}()

	var patterns, subsources []string
	// The slice of sources is separate from the set of sources to keep the
	// order in which the files are walked.
	for sourcesSet := aghstrings.NewSet(rc.initPath); i < len(sources); i++ {
		patterns, has, err = rc.checkFile(sources[i], desired)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return false, err
		}

		if has {
			return true, nil
		}

		subsources, err = rc.handlePatterns(sourcesSet, patterns)
		if err != nil {
			return false, err
		}

		sources = append(sources, subsources...)
	}

	return false, nil
}

func ifaceHasStaticIP(ifaceName string) (has bool, err error) {
	// TODO(a.garipov): Currently, this function returns the first
	// definitive result.  So if /etc/dhcpcd.conf has a static IP while
	// /etc/network/interfaces doesn't, it will return true.  Perhaps this
	// is not the most desirable behavior.

	for _, rc := range []*recurrentChecker{{
		checker:  dhcpcdStaticConfig,
		initPath: "/etc/dhcpcd.conf",
	}, {
		checker:  ifacesStaticConfig,
		initPath: "/etc/network/interfaces",
	}} {
		has, err = rc.check(ifaceName)
		if err != nil {
			return false, err
		}

		if has {
			return true, nil
		}
	}

	return false, ErrNoStaticIPInfo
}

func canBindPrivilegedPorts() (can bool, err error) {
	cnbs, err := unix.PrctlRetInt(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_IS_SET, unix.CAP_NET_BIND_SERVICE, 0, 0)
	// Don't check the error because it's always nil on Linux.
	adm, _ := aghos.HaveAdminRights()

	return cnbs == 1 || adm, err
}

// findIfaceLine scans s until it finds the line that declares an interface with
// the given name.  If findIfaceLine can't find the line, it returns false.
func findIfaceLine(s *bufio.Scanner, name string) (ok bool) {
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "interface" && fields[1] == name {
			return true
		}
	}

	return false
}

// dhcpcdStaticConfig checks if interface is configured by /etc/dhcpcd.conf to
// have a static IP.
func dhcpcdStaticConfig(r io.Reader, ifaceName string) (subsources []string, has bool, err error) {
	s := bufio.NewScanner(r)
	ifaceFound := findIfaceLine(s, ifaceName)
	if !ifaceFound {
		return nil, false, s.Err()
	}

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) >= 2 &&
			fields[0] == "static" &&
			strings.HasPrefix(fields[1], "ip_address=") {
			return nil, true, s.Err()
		}

		if len(fields) > 0 && fields[0] == "interface" {
			// Another interface found.
			break
		}
	}

	return nil, false, s.Err()
}

// ifacesStaticConfig checks if the interface is configured by any file of
// /etc/network/interfaces format to have a static IP.
func ifacesStaticConfig(r io.Reader, ifaceName string) (subsources []string, has bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if aghstrings.IsCommentOrEmpty(line) {
			continue
		}

		// TODO(e.burkov): As man page interfaces(5) says, a line may be
		// extended across multiple lines by making the last character a
		// backslash.  Provide extended lines and "source-directory"
		// stanzas support.

		fields := strings.Fields(line)
		fieldsNum := len(fields)

		// Man page interfaces(5) declares that interface definition
		// should consist of the key word "iface" followed by interface
		// name, and method at fourth field.
		if fieldsNum >= 4 &&
			fields[0] == "iface" && fields[1] == ifaceName && fields[3] == "static" {
			return nil, true, nil
		}

		if fieldsNum >= 2 && fields[0] == "source" {
			subsources = append(subsources, fields[1])
		}
	}

	return subsources, false, s.Err()
}

// ifaceSetStaticIP configures the system to retain its current IP on the
// interface through dhcpdc.conf.
func ifaceSetStaticIP(ifaceName string) (err error) {
	ipNet := GetSubnet(ifaceName)
	if ipNet.IP == nil {
		return errors.Error("can't get IP address")
	}

	gatewayIP := GatewayIP(ifaceName)
	add := dhcpcdConfIface(ifaceName, ipNet, gatewayIP, ipNet.IP)

	body, err := os.ReadFile("/etc/dhcpcd.conf")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	body = append(body, []byte(add)...)
	err = maybe.WriteFile("/etc/dhcpcd.conf", body, 0o644)
	if err != nil {
		return fmt.Errorf("writing conf: %w", err)
	}

	return nil
}

// dhcpcdConfIface returns configuration lines for the dhcpdc.conf files that
// configure the interface to have a static IP.
func dhcpcdConfIface(ifaceName string, ipNet *net.IPNet, gatewayIP, dnsIP net.IP) (conf string) {
	var body []byte

	add := fmt.Sprintf(
		"\n# %[1]s added by AdGuard Home.\ninterface %[1]s\nstatic ip_address=%s\n",
		ifaceName,
		ipNet)
	body = append(body, []byte(add)...)

	if gatewayIP != nil {
		add = fmt.Sprintf("static routers=%s\n", gatewayIP)
		body = append(body, []byte(add)...)
	}

	add = fmt.Sprintf("static domain_name_servers=%s\n\n", dnsIP)
	body = append(body, []byte(add)...)

	return string(body)
}
