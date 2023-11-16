package dhcpsvc

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/google/gopacket/layers"
)

// IPv4Config is the interface-specific configuration for DHCPv4.
type IPv4Config struct {
	// GatewayIP is the IPv4 address of the network's gateway.  It is used as
	// the default gateway for DHCP clients and also used in calculating the
	// network-specific broadcast address.
	GatewayIP netip.Addr

	// SubnetMask is the IPv4 subnet mask of the network.  It should be a valid
	// IPv4 CIDR (i.e. all 1s followed by all 0s).
	SubnetMask netip.Addr

	// RangeStart is the first address in the range to assign to DHCP clients.
	RangeStart netip.Addr

	// RangeEnd is the last address in the range to assign to DHCP clients.
	RangeEnd netip.Addr

	// Options is the list of DHCP options to send to DHCP clients.
	Options layers.DHCPOptions

	// LeaseDuration is the TTL of a DHCP lease.
	LeaseDuration time.Duration

	// Enabled is the state of the DHCPv4 service, whether it is enabled or not
	// on the specific interface.
	Enabled bool
}

// validate returns an error in conf if any.
func (conf *IPv4Config) validate() (err error) {
	switch {
	case conf == nil:
		return errNilConfig
	case !conf.Enabled:
		return nil
	case !conf.GatewayIP.Is4():
		return newMustErr("gateway ip", "be a valid ipv4", conf.GatewayIP)
	case !conf.SubnetMask.Is4():
		return newMustErr("subnet mask", "be a valid ipv4 cidr mask", conf.SubnetMask)
	case !conf.RangeStart.Is4():
		return newMustErr("range start", "be a valid ipv4", conf.RangeStart)
	case !conf.RangeEnd.Is4():
		return newMustErr("range end", "be a valid ipv4", conf.RangeEnd)
	case conf.LeaseDuration <= 0:
		return newMustErr("lease duration", "be less than %d", conf.LeaseDuration)
	default:
		return nil
	}
}

// iface4 is a DHCP interface for IPv4 address family.
type iface4 struct {
	// gateway is the IP address of the network gateway.
	gateway netip.Addr

	// subnet is the network subnet.
	subnet netip.Prefix

	// addrSpace is the IPv4 address space allocated for leasing.
	addrSpace ipRange

	// name is the name of the interface.
	name string

	// TODO(e.burkov):  Add options.

	// leaseTTL is the time-to-live of dynamic leases on this interface.
	leaseTTL time.Duration
}

// newIface4 creates a new DHCP interface for IPv4 address family with the given
// configuration.  It returns an error if the given configuration can't be used.
func newIface4(name string, conf *IPv4Config) (i *iface4, err error) {
	if !conf.Enabled {
		return nil, nil
	}

	maskLen, _ := net.IPMask(conf.SubnetMask.AsSlice()).Size()
	subnet := netip.PrefixFrom(conf.GatewayIP, maskLen)

	switch {
	case !subnet.Contains(conf.RangeStart):
		return nil, fmt.Errorf("range start %s is not within %s", conf.RangeStart, subnet)
	case !subnet.Contains(conf.RangeEnd):
		return nil, fmt.Errorf("range end %s is not within %s", conf.RangeEnd, subnet)
	}

	addrSpace, err := newIPRange(conf.RangeStart, conf.RangeEnd)
	if err != nil {
		return nil, err
	} else if addrSpace.contains(conf.GatewayIP) {
		return nil, fmt.Errorf("gateway ip %s in the ip range %s", conf.GatewayIP, addrSpace)
	}

	return &iface4{
		name:      name,
		gateway:   conf.GatewayIP,
		subnet:    subnet,
		addrSpace: addrSpace,
		leaseTTL:  conf.LeaseDuration,
	}, nil
}
