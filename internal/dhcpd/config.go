package dhcpd

import (
	"fmt"
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
)

// ServerConfig is the configuration for the DHCP server.  The order of YAML
// fields is important, since the YAML configuration file follows it.
type ServerConfig struct {
	// Called when the configuration is changed by HTTP request
	ConfigModified func() `yaml:"-"`

	// Register an HTTP handler
	HTTPRegister aghhttp.RegisterFunc `yaml:"-"`

	Enabled       bool   `yaml:"enabled"`
	InterfaceName string `yaml:"interface_name"`

	// LocalDomainName is the domain name used for DHCP hosts.  For example,
	// a DHCP client with the hostname "myhost" can be addressed as "myhost.lan"
	// when LocalDomainName is "lan".
	LocalDomainName string `yaml:"local_domain_name"`

	Conf4 V4ServerConf `yaml:"dhcpv4"`
	Conf6 V6ServerConf `yaml:"dhcpv6"`

	WorkDir    string `yaml:"-"`
	DBFilePath string `yaml:"-"`
}

// DHCPServer - DHCP server interface
type DHCPServer interface {
	// ResetLeases resets leases.
	ResetLeases(leases []*Lease) (err error)
	// GetLeases returns deep clones of the current leases.
	GetLeases(flags GetLeasesFlags) (leases []*Lease)
	// AddStaticLease - add a static lease
	AddStaticLease(l *Lease) (err error)
	// RemoveStaticLease - remove a static lease
	RemoveStaticLease(l *Lease) (err error)
	// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
	FindMACbyIP(ip net.IP) net.HardwareAddr

	// WriteDiskConfig4 - copy disk configuration
	WriteDiskConfig4(c *V4ServerConf)
	// WriteDiskConfig6 - copy disk configuration
	WriteDiskConfig6(c *V6ServerConf)

	// Start - start server
	Start() (err error)
	// Stop - stop server
	Stop() (err error)
	getLeasesRef() []*Lease
}

// V4ServerConf - server configuration
type V4ServerConf struct {
	Enabled       bool   `yaml:"-" json:"-"`
	InterfaceName string `yaml:"-" json:"-"`

	GatewayIP  net.IP `yaml:"gateway_ip" json:"gateway_ip"`
	SubnetMask net.IP `yaml:"subnet_mask" json:"subnet_mask"`
	// broadcastIP is the broadcasting address pre-calculated from the
	// configured gateway IP and subnet mask.
	broadcastIP net.IP

	// The first & the last IP address for dynamic leases
	// Bytes [0..2] of the last allowed IP address must match the first IP
	RangeStart net.IP `yaml:"range_start" json:"range_start"`
	RangeEnd   net.IP `yaml:"range_end" json:"range_end"`

	LeaseDuration uint32 `yaml:"lease_duration" json:"lease_duration"` // in seconds

	// IP conflict detector: time (ms) to wait for ICMP reply
	// 0: disable
	ICMPTimeout uint32 `yaml:"icmp_timeout_msec" json:"-"`

	// Custom Options.
	//
	// Option with arbitrary hexadecimal data:
	//     DEC_CODE hex HEX_DATA
	// where DEC_CODE is a decimal DHCPv4 option code in range [1..255]
	//
	// Option with IP data (only 1 IP is supported):
	//     DEC_CODE ip IP_ADDR
	Options []string `yaml:"options" json:"-"`

	ipRange *ipRange

	leaseTime  time.Duration // the time during which a dynamic lease is considered valid
	dnsIPAddrs []net.IP      // IPv4 addresses to return to DHCP clients as DNS server addresses

	// subnet contains the DHCP server's subnet.  The IP is the IP of the
	// gateway.
	subnet *net.IPNet

	// notify is a way to signal to other components that leases have been
	// changed.  notify must be called outside of locked sections, since the
	// clients might want to get the new data.
	//
	// TODO(a.garipov): This is utter madness and must be refactored.  It just
	// begs for deadlock bugs and other nastiness.
	notify func(uint32)
}

// errNilConfig is an error returned by validation method if the config is nil.
const errNilConfig errors.Error = "nil config"

// ensureV4 returns a 4-byte version of ip.  An error is returned if the passed
// ip is not an IPv4.
func ensureV4(ip net.IP) (ip4 net.IP, err error) {
	if ip == nil {
		return nil, fmt.Errorf("%v is not an IP address", ip)
	}

	ip4 = ip.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("%v is not an IPv4 address", ip)
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

	var gatewayIP net.IP
	gatewayIP, err = ensureV4(c.GatewayIP)
	if err != nil {
		// Don't wrap an errors since it's inforative enough as is and there is
		// an annotation deferred already.
		return err
	}

	if c.SubnetMask == nil {
		return fmt.Errorf("invalid subnet mask: %v", c.SubnetMask)
	}

	subnetMask := net.IPMask(netutil.CloneIP(c.SubnetMask.To4()))
	c.subnet = &net.IPNet{
		IP:   gatewayIP,
		Mask: subnetMask,
	}
	c.broadcastIP = aghnet.BroadcastFromIPNet(c.subnet)

	c.ipRange, err = newIPRange(c.RangeStart, c.RangeEnd)
	if err != nil {
		// Don't wrap an errors since it's inforative enough as is and there is
		// an annotation deferred already.
		return err
	}

	if c.ipRange.contains(gatewayIP) {
		return fmt.Errorf("gateway ip %v in the ip range: %v-%v",
			gatewayIP,
			c.RangeStart,
			c.RangeEnd,
		)
	}

	if !c.subnet.Contains(c.RangeStart) {
		return fmt.Errorf("range start %v is outside network %v",
			c.RangeStart,
			c.subnet,
		)
	}

	if !c.subnet.Contains(c.RangeEnd) {
		return fmt.Errorf("range end %v is outside network %v",
			c.RangeEnd,
			c.subnet,
		)
	}

	return nil
}

// V6ServerConf - server configuration
type V6ServerConf struct {
	Enabled       bool   `yaml:"-" json:"-"`
	InterfaceName string `yaml:"-" json:"-"`

	// The first IP address for dynamic leases
	// The last allowed IP address ends with 0xff byte
	RangeStart net.IP `yaml:"range_start" json:"range_start"`

	LeaseDuration uint32 `yaml:"lease_duration" json:"lease_duration"` // in seconds

	RASLAACOnly  bool `yaml:"ra_slaac_only" json:"-"`  // send ICMPv6.RA packets without MO flags
	RAAllowSLAAC bool `yaml:"ra_allow_slaac" json:"-"` // send ICMPv6.RA packets with MO flags

	ipStart    net.IP        // starting IP address for dynamic leases
	leaseTime  time.Duration // the time during which a dynamic lease is considered valid
	dnsIPAddrs []net.IP      // IPv6 addresses to return to DHCP clients as DNS server addresses

	// Server calls this function when leases data changes
	notify func(uint32)
}
