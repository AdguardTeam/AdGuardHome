package dhcpsvc

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
)

// Config is the configuration for the DHCP service.
type Config struct {
	// Interfaces stores configurations of DHCP server specific for the network
	// interface identified by its name.
	Interfaces map[string]*InterfaceConfig

	// Logger will be used to log the DHCP events.
	Logger *slog.Logger

	// LocalDomainName is the top-level domain name to use for resolving DHCP
	// clients' hostnames.
	LocalDomainName string

	// DBFilePath is the path to the database file containing the DHCP leases.
	DBFilePath string

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
//
// TODO(e.burkov):  Unexport and rewrite the test.
func (conf *Config) Validate() (err error) {
	switch {
	case conf == nil:
		return errNilConfig
	case !conf.Enabled:
		return nil
	}

	var errs []error
	if conf.ICMPTimeout < 0 {
		err = newMustErr("icmp timeout", "be non-negative", conf.ICMPTimeout)
		errs = append(errs, err)
	}

	err = netutil.ValidateDomainName(conf.LocalDomainName)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		errs = append(errs, err)
	}

	// This is a best-effort check for the file accessibility.  The file will be
	// checked again when it is opened later.
	if _, err = os.Stat(conf.DBFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Errorf("db file path %q: %w", conf.DBFilePath, err))
	}

	if len(conf.Interfaces) == 0 {
		errs = append(errs, errNoInterfaces)

		return errors.Join(errs...)
	}

	for _, iface := range slices.Sorted(maps.Keys(conf.Interfaces)) {
		ic := conf.Interfaces[iface]
		err = ic.validate()
		if err != nil {
			errs = append(errs, fmt.Errorf("interface %q: %w", iface, err))
		}
	}

	return errors.Join(errs...)
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
