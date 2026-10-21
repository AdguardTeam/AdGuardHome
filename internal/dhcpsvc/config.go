package dhcpsvc

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/validate"
)

// Config is the configuration for the DHCP service.
type Config struct {
	// AddressChecker is used to check the availability of IP addresses.  It
	// must not be nil.
	AddressChecker AddressChecker

	// Database is the storage for DHCP leases.  It must not be nil.
	Database Database

	// Interfaces stores configurations of DHCP server specific for the network
	// interface identified by its name.  It must not be empty and must only
	// contain valid interface names and configurations.
	Interfaces map[string]*InterfaceConfig

	// NetworkDeviceManager is the manager of network devices.  It must not be
	// nil.
	//
	// TODO(e.burkov):  Implement and set.
	NetworkDeviceManager NetworkDeviceManager

	// Logger will be used to log the DHCP events.  It must not be nil.
	Logger *slog.Logger

	// LocalDomainName is the top-level domain name to use for resolving DHCP
	// clients' hostnames.  It must be a valid domain name.
	LocalDomainName string

	// Enabled is the state of the service, whether it is enabled or not.
	Enabled bool
}

// type check
var _ validate.Interface = (*Config)(nil)

// Validate implements the [validate.Interface] for *Config.
func (conf *Config) Validate() (err error) {
	switch {
	case conf == nil:
		return errors.ErrNoValue
	case !conf.Enabled:
		return nil
	}

	errs := []error{
		validate.NotNilInterface("conf.Database", conf.Database),
		validate.NotNilInterface("conf.AddressChecker", conf.AddressChecker),
		validate.NotNil("conf.Logger", conf.Logger),
		validate.NotNilInterface("conf.NetworkDeviceManager", conf.NetworkDeviceManager),
	}

	err = netutil.ValidateDomainName(conf.LocalDomainName)
	if err != nil {
		errs = append(errs, fmt.Errorf("conf.LocalDomainName: %w", err))
	}

	if len(conf.Interfaces) == 0 {
		err = fmt.Errorf("conf.Interfaces: %w", errors.ErrEmptyValue)
		errs = append(errs, err)

		return errors.Join(errs...)
	}

	for _, iface := range slices.Sorted(maps.Keys(conf.Interfaces)) {
		ifaceConf := conf.Interfaces[iface]
		errs = validate.Append(errs, "conf.Interfaces."+iface, ifaceConf)
	}

	return errors.Join(errs...)
}

// InterfaceConfig is the configuration of a single DHCP interface.
type InterfaceConfig struct {
	// IPv4 is the configuration of DHCP protocol for IPv4.
	IPv4 *IPv4Config

	// IPv6 is the configuration of DHCP protocol for IPv6.
	IPv6 *IPv6Config
}

// type check
var _ validate.Interface = (*InterfaceConfig)(nil)

// Validate implements the [validate.Interface] interface for *InterfaceConfig.
func (ic *InterfaceConfig) Validate() (err error) {
	if ic == nil {
		return errors.ErrNoValue
	}

	errs := []error{
		errors.Annotate(ic.IPv4.Validate(), "ipv4: %w"),
		errors.Annotate(ic.IPv6.Validate(), "ipv6: %w"),
	}

	return errors.Join(errs...)
}
