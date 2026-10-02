package configmgr

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// DHCPConfig is the on-disk DHCP configuration.
type DHCPConfig struct {
	// Conf4 is the configuration of the DHCPv4 server.
	Conf4 *DHCPv4Config `yaml:"dhcpv4"`

	// Conf6 is the configuration of the DHCPv6 server.
	Conf6 *DHCPv6Config `yaml:"dhcpv6"`

	// InterfaceName is the name of the network interface the DHCP server
	// listens on.
	InterfaceName string `yaml:"interface_name"`

	// LocalDomainName is the domain name used for DHCP hosts.  For example, a
	// DHCP client with the hostname "myhost" can be addressed as "myhost.lan"
	// when LocalDomainName is "lan".
	//
	// TODO(e.burkov):  Probably, remove this field.  See the TODO on
	// [dhcpd.Interface.Enabled].
	LocalDomainName string `yaml:"local_domain_name"`

	// Enabled defines if the DHCP server is enabled.
	Enabled bool `yaml:"enabled"`
}

// DHCPv4Config is the on-disk configuration of the DHCPv4 server.
type DHCPv4Config struct {
	// GatewayIP is the IPv4 address of the network gateway advertised to DHCP
	// clients.  It must be outside the RangeStart–RangeEnd range.
	GatewayIP netip.Addr `yaml:"gateway_ip"`

	// RangeStart is the first IPv4 address of the dynamic lease range.  It
	// must be within the subnet defined by GatewayIP and SubnetMask and must
	// be less than RangeEnd.
	RangeStart netip.Addr `yaml:"range_start"`

	// RangeEnd is the last IPv4 address of the dynamic lease range.  It must
	// be within the subnet defined by GatewayIP and SubnetMask and must be
	// greater than RangeStart.
	RangeEnd netip.Addr `yaml:"range_end"`

	// SubnetMask is the IPv4 subnet mask of the served network.
	SubnetMask netip.Addr `yaml:"subnet_mask"`

	// Options is the list of custom DHCPv4 options.
	//
	// Option with arbitrary hexadecimal data:
	//     DEC_CODE hex HEX_DATA
	// where DEC_CODE is a decimal DHCPv4 option code in range [1..255]
	//
	// Option with IP data (only 1 IP is supported):
	//     DEC_CODE ip IP_ADDR
	Options []string `yaml:"options"`

	// ICMPTimeout is the time in milliseconds to wait for an ICMP reply during
	// IP conflict detection.  A value of 0 disables the detection.
	ICMPTimeout uint32 `yaml:"icmp_timeout_msec"`

	// LeaseDuration is the duration of lease in seconds.
	LeaseDuration uint32 `yaml:"lease_duration"`
}

// DHCPv6Config is the on-disk configuration of the DHCPv6 server.
type DHCPv6Config struct {
	// RangeStart is the first IPv6 address of the dynamic lease range.  The
	// last allowed IP address ends with the 0xff byte.
	RangeStart net.IP `yaml:"range_start"`

	// LeaseDuration is the duration of lease in seconds.
	LeaseDuration uint32 `yaml:"lease_duration"`

	// RASLAACOnly defines whether to send ICMPv6.RA packets without MO flags.
	RASLAACOnly bool `yaml:"ra_slaac_only"`

	// RAAllowSLAAC defines whether to send ICMPv6.RA packets with MO flags.
	RAAllowSLAAC bool `yaml:"ra_allow_slaac"`
}

// type check
var _ validate.Interface = (*DHCPConfig)(nil)

// Validate implements the [validate.Interface] interface for *DHCPConfig.
func (c *DHCPConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	if !c.Enabled {
		return nil
	}

	// TODO(d.kolyshev):  Add validations.

	return errors.Join(
		errors.Annotate(c.Conf4.Validate(), "dhcpv4: %w"),
		errors.Annotate(c.Conf6.Validate(), "dhcpv6: %w"),
	)
}

// type check
var _ validate.Interface = (*DHCPv4Config)(nil)

// Validate implements the [validate.Interface] interface for *DHCPv4Config.
func (c *DHCPv4Config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	var errs []error
	gatewayIP, err := ensureV4(c.GatewayIP, "address")
	if err != nil {
		errs = append(errs, fmt.Errorf("gateway_ip: %w", err))
	}

	subnetMask, err := ensureV4(c.SubnetMask, "subnet mask")
	if err != nil {
		errs = append(errs, fmt.Errorf("subnet_mask: %w", err))
	}

	rangeStart, err := ensureV4(c.RangeStart, "address")
	if err != nil {
		errs = append(errs, fmt.Errorf("range_start: %w", err))
	}

	rangeEnd, err := ensureV4(c.RangeEnd, "address")
	if err != nil {
		errs = append(errs, fmt.Errorf("range_end: %w", err))
	}

	err = errors.Join(errs...)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	return c.validateRange(rangeStart, rangeEnd, gatewayIP, subnetMask)
}

// validateRange checks that the range is valid for the given network.
func (c *DHCPv4Config) validateRange(
	rangeStart netip.Addr,
	rangeEnd netip.Addr,
	gatewayIP netip.Addr,
	subnetMask netip.Addr,
) (err error) {
	if rangeStart.Compare(rangeEnd) >= 0 {
		return fmt.Errorf("invalid ip range: %v-%v", c.RangeStart, c.RangeEnd)
	}

	if rangeStart.Compare(gatewayIP) <= 0 && gatewayIP.Compare(rangeEnd) <= 0 {
		return fmt.Errorf(
			"gateway_ip: gateway ip %v in the ip range: %v-%v",
			gatewayIP,
			c.RangeStart,
			c.RangeEnd,
		)
	}

	// TODO(a.garipov):  Add masking helper in netutil.
	maskLen, _ := net.IPMask(subnetMask.AsSlice()).Size()
	subnet := netip.PrefixFrom(gatewayIP, maskLen)
	if !subnet.Contains(rangeStart) {
		return fmt.Errorf(
			"range_start: range start %v is outside network %v",
			c.RangeStart,
			subnet,
		)
	}

	if !subnet.Contains(rangeEnd) {
		return fmt.Errorf(
			"range_end: range end %v is outside network %v",
			c.RangeEnd,
			subnet,
		)
	}

	return nil
}

// ensureV4 returns an unmapped version of ip.  An error is returned if the
// passed ip is not an IPv4.
func ensureV4(ip netip.Addr, kind string) (ip4 netip.Addr, err error) {
	ip4 = ip.Unmap()
	if !ip4.IsValid() || !ip4.Is4() {
		return netip.Addr{}, fmt.Errorf("%v is not an IPv4 %s", ip, kind)
	}

	return ip4, nil
}

// type check
var _ validate.Interface = (*DHCPv6Config)(nil)

// Validate implements the [validate.Interface] interface for *DHCPv6Config.
func (c *DHCPv6Config) Validate() (err error) {
	// TODO(d.kolyshev):  Add validations.
	return nil
}
