package dhcpsvc

import (
	"fmt"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/netutil"
	"golang.org/x/exp/maps"
)

// Config is the configuration for the DHCP service.
type Config struct {
	// Interfaces stores configurations of DHCP server specific for the network
	// interface identified by its name.
	Interfaces map[string]*InterfaceConfig

	// LocalDomainName is the top-level domain name to use for resolving DHCP
	// clients' hostnames.
	LocalDomainName string

	// TODO(e.burkov):  Add DB path.

	// ICMPTimeout is the timeout for checking another DHCP server's presence.
	ICMPTimeout time.Duration

	// Enabled is the state of the service, whether it is enabled or not.
	Enabled bool
}

// InterfaceConfig is the configuration of a single DHCP interface.
type InterfaceConfig struct {
	// IPv4 is the configuration of DHCP protocol for IPv4.
	IPv4 *IPv4Config

	// IPv6 is the configuration of DHCP protocol for IPv6.
	IPv6 *IPv6Config
}

// Validate returns an error in conf if any.
func (conf *Config) Validate() (err error) {
	switch {
	case conf == nil:
		return errNilConfig
	case !conf.Enabled:
		return nil
	case conf.ICMPTimeout < 0:
		return newMustErr("icmp timeout", "be non-negative", conf.ICMPTimeout)
	}

	err = netutil.ValidateDomainName(conf.LocalDomainName)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	if len(conf.Interfaces) == 0 {
		return errNoInterfaces
	}

	ifaces := maps.Keys(conf.Interfaces)
	slices.Sort(ifaces)

	for _, iface := range ifaces {
		if err = conf.Interfaces[iface].validate(); err != nil {
			return fmt.Errorf("interface %q: %w", iface, err)
		}
	}

	return nil
}

// validate returns an error in ic, if any.
func (ic *InterfaceConfig) validate() (err error) {
	if ic == nil {
		return errNilConfig
	}

	if err = ic.IPv4.validate(); err != nil {
		return fmt.Errorf("ipv4: %w", err)
	}

	if err = ic.IPv6.validate(); err != nil {
		return fmt.Errorf("ipv6: %w", err)
	}

	return nil
}
