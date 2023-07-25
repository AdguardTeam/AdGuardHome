//go:build linux

package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/google/renameio/v2/maybe"
	"golang.org/x/sys/unix"
)

// dhcpсdConf is the name of /etc/dhcpcd.conf file in the root filesystem.
const dhcpcdConf = "etc/dhcpcd.conf"

func canBindPrivilegedPorts() (can bool, err error) {
	res, err := unix.PrctlRetInt(
		unix.PR_CAP_AMBIENT,
		unix.PR_CAP_AMBIENT_IS_SET,
		unix.CAP_NET_BIND_SERVICE,
		0,
		0,
	)
	if err != nil {
		if errors.Is(err, unix.EINVAL) {
			// Older versions of Linux kernel do not support this.  Print a
			// warning and check admin rights.
			log.Info("warning: cannot check capability cap_net_bind_service: %s", err)
		} else {
			return false, err
		}
	}

	// Don't check the error because it's always nil on Linux.
	adm, _ := aghos.HaveAdminRights()

	return res == 1 || adm, nil
}

// dhcpcdStaticConfig checks if interface is configured by /etc/dhcpcd.conf to
// have a static IP.
func (n interfaceName) dhcpcdStaticConfig(r io.Reader) (subsources []string, cont bool, err error) {
	s := bufio.NewScanner(r)
	if !findIfaceLine(s, string(n)) {
		return nil, true, s.Err()
	}

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) >= 2 &&
			fields[0] == "static" &&
			strings.HasPrefix(fields[1], "ip_address=") {
			return nil, false, s.Err()
		}

		if len(fields) > 0 && fields[0] == "interface" {
			// Another interface found.
			break
		}
	}

	return nil, true, s.Err()
}

// ifacesStaticConfig checks if the interface is configured by any file of
// /etc/network/interfaces format to have a static IP.
func (n interfaceName) ifacesStaticConfig(r io.Reader) (sub []string, cont bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// TODO(e.burkov): As man page interfaces(5) says, a line may be
		// extended across multiple lines by making the last character a
		// backslash.  Provide extended lines support.

		fields := strings.Fields(line)
		fieldsNum := len(fields)

		// Man page interfaces(5) declares that interface definition should
		// consist of the key word "iface" followed by interface name, and
		// method at fourth field.
		if fieldsNum >= 4 &&
			fields[0] == "iface" && fields[1] == string(n) && fields[3] == "static" {
			return nil, false, nil
		}

		if fieldsNum >= 2 && fields[0] == "source" {
			sub = append(sub, fields[1])
		}
	}

	return sub, true, s.Err()
}

func ifaceHasStaticIP(ifaceName string) (has bool, err error) {
	// TODO(a.garipov): Currently, this function returns the first definitive
	// result.  So if /etc/dhcpcd.conf has and /etc/network/interfaces has no
	// static IP configuration, it will return true.  Perhaps this is not the
	// most desirable behavior.

	iface := interfaceName(ifaceName)

	for _, pair := range [...]struct {
		aghos.FileWalker
		filename string
	}{{
		FileWalker: iface.dhcpcdStaticConfig,
		filename:   dhcpcdConf,
	}, {
		FileWalker: iface.ifacesStaticConfig,
		filename:   "etc/network/interfaces",
	}} {
		has, err = pair.Walk(rootDirFS, pair.filename)
		if err != nil {
			return false, err
		} else if has {
			return true, nil
		}
	}

	return false, ErrNoStaticIPInfo
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

// ifaceSetStaticIP configures the system to retain its current IP on the
// interface through dhcpcd.conf.
func ifaceSetStaticIP(ifaceName string) (err error) {
	ipNet := GetSubnet(ifaceName)
	if !ipNet.Addr().IsValid() {
		return errors.Error("can't get IP address")
	}

	body, err := os.ReadFile(dhcpcdConf)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	gatewayIP := GatewayIP(ifaceName)
	add := dhcpcdConfIface(ifaceName, ipNet, gatewayIP)

	body = append(body, []byte(add)...)
	err = maybe.WriteFile(dhcpcdConf, body, 0o644)
	if err != nil {
		return fmt.Errorf("writing conf: %w", err)
	}

	return nil
}

// dhcpcdConfIface returns configuration lines for the dhcpdc.conf files that
// configure the interface to have a static IP.
func dhcpcdConfIface(ifaceName string, subnet netip.Prefix, gateway netip.Addr) (conf string) {
	b := &strings.Builder{}
	stringutil.WriteToBuilder(
		b,
		"\n# ",
		ifaceName,
		" added by AdGuard Home.\ninterface ",
		ifaceName,
		"\nstatic ip_address=",
		subnet.String(),
		"\n",
	)

	if gateway != (netip.Addr{}) {
		stringutil.WriteToBuilder(b, "static routers=", gateway.String(), "\n")
	}

	stringutil.WriteToBuilder(b, "static domain_name_servers=", subnet.Addr().String(), "\n\n")

	return b.String()
}
