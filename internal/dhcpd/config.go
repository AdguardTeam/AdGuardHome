package dhcpd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
)

// ServerConfig is the configuration for the DHCP server.
type ServerConfig struct {
	// Conf6 is the configuration of the DHCPv6 server.
	Conf6 V6ServerConf

	// CommandConstructor is used to run external commands.  It must not be nil.
	CommandConstructor executil.CommandConstructor

	// ConfModifier is used to update the global configuration.  It must not be
	// nil.
	ConfModifier agh.ConfigModifier

	// HTTPReg is used to register an HTTP handler.
	HTTPReg aghhttp.Registrar

	// Logger is used for logging the operation of the DHCP server.  It must not
	// be nil.
	Logger *slog.Logger

	// InterfaceName is the name of the network interface the DHCP server
	// listens on.
	InterfaceName string

	// LocalDomainName is the domain name used for DHCP hosts.  For example, a
	// DHCP client with the hostname "myhost" can be addressed as "myhost.lan"
	// when LocalDomainName is "lan".
	//
	// TODO(e.burkov):  Probably, remove this field.  See the TODO on
	// [Interface.Enabled].
	LocalDomainName string

	// WorkDir is used to store DHCP leases.
	//
	// Deprecated:  Remove it when migration of DHCP leases will not be needed.
	WorkDir string

	// DataDir is used to store DHCP leases.
	DataDir string

	// dbFilePath is the path to the file with stored DHCP leases.
	dbFilePath string

	// Conf4 is the configuration of the DHCPv4 server.
	Conf4 V4ServerConf

	// Enabled defines if the DHCP server is enabled.
	Enabled bool
}

// DHCPServer - DHCP server interface
type DHCPServer interface {
	// ResetLeases resets leases.
	ResetLeases(leases []*dhcpsvc.Lease) (err error)
	// GetLeases returns deep clones of the current leases.
	GetLeases(flags GetLeasesFlags) (leases []*dhcpsvc.Lease)
	// AddStaticLease - add a static lease
	AddStaticLease(l *dhcpsvc.Lease) (err error)
	// RemoveStaticLease - remove a static lease
	RemoveStaticLease(l *dhcpsvc.Lease) (err error)

	// UpdateStaticLease updates IP, hostname of the lease.
	UpdateStaticLease(l *dhcpsvc.Lease) (err error)

	// FindMACbyIP returns a MAC address by the IP address of its lease, if
	// there is one.
	FindMACbyIP(ip netip.Addr) (mac net.HardwareAddr)

	// HostByIP returns a hostname by the IP address of its lease, if there is
	// one.
	HostByIP(ip netip.Addr) (host string)

	// IPByHost returns an IP address by the hostname of its lease, if there is
	// one.
	IPByHost(host string) (ip netip.Addr)

	// WriteDiskConfig4 copies IPv4 configuration, dc must not be nil.
	WriteDiskConfig4(dc *configmgr.DHCPv4Config)

	// WriteDiskConfig6 copy IPv6 configuration, dc must not be nil.
	WriteDiskConfig6(dc *configmgr.DHCPv6Config)

	// Start - start server
	Start(ctx context.Context) (err error)
	// Stop - stop server
	Stop() (err error)
	getLeasesRef() []*dhcpsvc.Lease
}

// V4ServerConf is the configuration of the DHCPv4 server.
type V4ServerConf struct {
	// broadcastIP is the broadcasting address pre-calculated from the
	// configured gateway IP and subnet mask.
	broadcastIP netip.Addr

	// GatewayIP is the IPv4 address of the network gateway advertised to DHCP
	// clients.  It must be outside the RangeStart–RangeEnd range.
	GatewayIP netip.Addr `json:"gateway_ip"`

	// RangeStart is the first IPv4 address of the dynamic lease range.
	//
	// Bytes [0..2] of the last allowed IP address must match the first IP.
	RangeStart netip.Addr `json:"range_start"`

	// RangeEnd is the last IPv4 address of the dynamic lease range.
	RangeEnd netip.Addr `json:"range_end"`

	// SubnetMask is the IPv4 subnet mask of the served network.
	SubnetMask netip.Addr `json:"subnet_mask"`

	// ipRange is the dynamic lease range.  It must not be nil.
	ipRange *ipRange

	// Logger is used for logging the operation of the DHCPv4 server.  It must
	// not be nil.
	Logger *slog.Logger `json:"-"`

	// notify is a way to signal to other components that leases have been
	// changed.  notify must be called outside of locked sections, since the
	// clients might want to get the new data.
	//
	// TODO(a.garipov): This is utter madness and must be refactored.  It just
	// begs for deadlock bugs and other nastiness.
	notify func(uint32)

	// subnet contains the DHCP server's subnet.  The IP is the IP of the
	// gateway.
	subnet netip.Prefix

	// InterfaceName is the name of the network interface the DHCPv4 server
	// listens on.
	InterfaceName string `json:"-"`

	// Options is the list of custom DHCPv4 options.
	//
	// Option with arbitrary hexadecimal data:
	//     DEC_CODE hex HEX_DATA
	// where DEC_CODE is a decimal DHCPv4 option code in range [1..255]
	//
	// Option with IP data (only 1 IP is supported):
	//     DEC_CODE ip IP_ADDR
	Options []string `json:"-"`

	// dnsIPAddrs is the IPv4 addresses to return to DHCP clients as DNS server
	// addresses.
	dnsIPAddrs []netip.Addr

	// leaseTime is the time during which a dynamic lease is considered valid.
	leaseTime time.Duration

	// LeaseDuration is the duration of a lease in seconds.
	LeaseDuration uint32 `json:"lease_duration"`

	// ICMPTimeout is the time in milliseconds to wait for an ICMP reply during
	// IP conflict detection.  A value of 0 disables the detection.
	ICMPTimeout uint32 `json:"-"`

	// Enabled defines if the DHCPv4 server is enabled.
	Enabled bool `json:"-"`
}

// errNilConfig is an error returned by validation method if the config is nil.
const errNilConfig errors.Error = "nil config"

// ensureV4 returns an unmapped version of ip.  An error is returned if the
// passed ip is not an IPv4.
func ensureV4(ip netip.Addr, kind string) (ip4 netip.Addr, err error) {
	ip4 = ip.Unmap()
	if !ip4.IsValid() || !ip4.Is4() {
		return netip.Addr{}, fmt.Errorf("%v is not an IPv4 %s", ip, kind)
	}

	return ip4, nil
}

// Validate returns an error if c is not a valid configuration.
//
// TODO(e.burkov):  Don't set the config fields when the server itself will stop
// containing the config.
func (c *V4ServerConf) Validate() (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if c == nil {
		return errNilConfig
	}

	gatewayIP, err := ensureV4(c.GatewayIP, "address")
	if err != nil {
		// Don't wrap the error since it's informative enough as is and there is
		// an annotation deferred already.
		return err
	}

	subnetMask, err := ensureV4(c.SubnetMask, "subnet mask")
	if err != nil {
		// Don't wrap the error since it's informative enough as is and there is
		// an annotation deferred already.
		return err
	}
	maskLen, _ := net.IPMask(subnetMask.AsSlice()).Size()

	c.subnet = netip.PrefixFrom(gatewayIP, maskLen)
	c.broadcastIP = aghnet.BroadcastFromPref(c.subnet)

	rangeStart, err := ensureV4(c.RangeStart, "address")
	if err != nil {
		// Don't wrap the error since it's informative enough as is and there is
		// an annotation deferred already.
		return err
	}

	rangeEnd, err := ensureV4(c.RangeEnd, "address")
	if err != nil {
		// Don't wrap the error since it's informative enough as is and there is
		// an annotation deferred already.
		return err
	}

	c.ipRange, err = newIPRange(rangeStart.AsSlice(), rangeEnd.AsSlice())
	if err != nil {
		// Don't wrap the error since it's informative enough as is and there is
		// an annotation deferred already.
		return err
	}

	if c.ipRange.contains(gatewayIP.AsSlice()) {
		return fmt.Errorf(
			"gateway ip %v in the ip range: %v-%v",
			gatewayIP,
			c.RangeStart,
			c.RangeEnd,
		)
	}

	if !c.subnet.Contains(rangeStart) {
		return fmt.Errorf(
			"range start %v is outside network %v",
			c.RangeStart,
			c.subnet,
		)
	}

	if !c.subnet.Contains(rangeEnd) {
		return fmt.Errorf(
			"range end %v is outside network %v",
			c.RangeEnd,
			c.subnet,
		)
	}

	return nil
}

// V6ServerConf is the configuration of the DHCPv6 server.
type V6ServerConf struct {
	// Logger is used for logging the operation of the DHCPv6 server.  It must
	// not be nil.
	Logger *slog.Logger `json:"-"`

	// notify is called when the leases data changes.
	notify func(uint32)

	// InterfaceName is the name of the network interface the DHCPv6 server
	// listens on.
	InterfaceName string `json:"-"`

	// RangeStart is the first IPv6 address of the dynamic lease range.  The
	// last allowed IP address ends with the 0xff byte.
	RangeStart net.IP `json:"range_start"`

	// ipStart is the starting IP address for dynamic leases.
	ipStart net.IP

	// dnsIPAddrs is the IPv6 addresses to return to DHCP clients as DNS server
	// addresses.
	dnsIPAddrs []net.IP

	// leaseTime is the time during which a dynamic lease is considered valid.
	leaseTime time.Duration

	// LeaseDuration is the duration of a lease in seconds.
	LeaseDuration uint32 `json:"lease_duration"`

	// Enabled defines if the DHCPv6 server is enabled.
	Enabled bool `json:"-"`

	// RASLAACOnly defines whether to send ICMPv6.RA packets without MO flags.
	RASLAACOnly bool `json:"-"`

	// RAAllowSLAAC defines whether to send ICMPv6.RA packets with MO flags.
	RAAllowSLAAC bool `json:"-"`
}
