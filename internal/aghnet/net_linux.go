// +build linux

package aghnet

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/golibs/file"
)

// maxConfigFileSize is the maximum length of interfaces configuration file.
const maxConfigFileSize = 1024 * 1024

func ifaceHasStaticIP(ifaceName string) (has bool, err error) {
	// TODO(a.garipov): Currently, this function returns the first
	// definitive result.  So if /etc/dhcpcd.conf has a static IP while
	// /etc/network/interfaces doesn't, it will return true.  Perhaps this
	// is not the most desirable behavior.

	for _, check := range []struct {
		checker  func(io.Reader, string) (bool, error)
		filePath string
	}{{
		checker:  dhcpcdStaticConfig,
		filePath: "/etc/dhcpcd.conf",
	}, {
		checker:  ifacesStaticConfig,
		filePath: "/etc/network/interfaces",
	}} {
		var f *os.File
		f, err = os.Open(check.filePath)
		if err != nil {
			// ErrNotExist can happen here if there is no such file.
			// This is normal, as not every system uses those files.
			if errors.Is(err, os.ErrNotExist) {
				err = nil

				continue
			}

			return false, err
		}
		defer f.Close()

		var fileReadCloser io.ReadCloser
		fileReadCloser, err = aghio.LimitReadCloser(f, maxConfigFileSize)
		if err != nil {
			return false, err
		}
		defer fileReadCloser.Close()

		has, err = check.checker(fileReadCloser, ifaceName)
		if err != nil {
			return false, err
		}

		return has, nil
	}

	return false, ErrNoStaticIPInfo
}

// dhcpcdStaticConfig checks if interface is configured by /etc/dhcpcd.conf to
// have a static IP.
func dhcpcdStaticConfig(r io.Reader, ifaceName string) (has bool, err error) {
	s := bufio.NewScanner(r)
	var withinInterfaceCtx bool

	for s.Scan() {
		line := strings.TrimSpace(s.Text())

		if withinInterfaceCtx && len(line) == 0 {
			// An empty line resets our state.
			withinInterfaceCtx = false
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)

		if withinInterfaceCtx {
			if len(fields) >= 2 && fields[0] == "static" && strings.HasPrefix(fields[1], "ip_address=") {
				return true, nil
			}
			if len(fields) > 0 && fields[0] == "interface" {
				// Another interface found.
				withinInterfaceCtx = false
			}
			continue
		}

		if len(fields) == 2 && fields[0] == "interface" && fields[1] == ifaceName {
			// The interface found.
			withinInterfaceCtx = true
		}
	}

	return false, s.Err()
}

// ifacesStaticConfig checks if interface is configured by
// /etc/network/interfaces to have a static IP.
func ifacesStaticConfig(r io.Reader, ifaceName string) (has bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		// Man page interfaces(5) declares that interface definition
		// should consist of the key word "iface" followed by interface
		// name, and method at fourth field.
		if len(fields) >= 4 && fields[0] == "iface" && fields[1] == ifaceName && fields[3] == "static" {
			return true, nil
		}
	}
	return false, s.Err()
}

func ifaceSetStaticIP(ifaceName string) (err error) {
	ipNet := GetSubnet(ifaceName)
	if ipNet.IP == nil {
		return errors.New("can't get IP address")
	}

	gatewayIP := GatewayIP(ifaceName)
	add := updateStaticIPdhcpcdConf(ifaceName, ipNet.String(), gatewayIP, ipNet.IP)

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

// updateStaticIPdhcpcdConf sets static IP address for the interface by writing
// into dhcpd.conf.
func updateStaticIPdhcpcdConf(ifaceName, ip string, gatewayIP, dnsIP net.IP) string {
	var body []byte

	add := fmt.Sprintf("\ninterface %s\nstatic ip_address=%s\n",
		ifaceName, ip)
	body = append(body, []byte(add)...)

	if gatewayIP != nil {
		add = fmt.Sprintf("static routers=%s\n",
			gatewayIP)
		body = append(body, []byte(add)...)
	}

	add = fmt.Sprintf("static domain_name_servers=%s\n\n",
		dnsIP)
	body = append(body, []byte(add)...)

	return string(body)
}
