package dhcpd

import (
	"net"
	"time"
)

// DHCPServer - DHCP server interface
type DHCPServer interface {
	// ResetLeases - reset leases
	ResetLeases(leases []*Lease)
	// GetLeases - get leases
	GetLeases(flags int) []Lease
	// GetLeasesRef - get reference to leases array
	GetLeasesRef() []*Lease
	// AddStaticLease - add a static lease
	AddStaticLease(lease Lease) error
	// RemoveStaticLease - remove a static lease
	RemoveStaticLease(l Lease) error
	// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
	FindMACbyIP(ip net.IP) net.HardwareAddr

	// WriteDiskConfig4 - copy disk configuration
	WriteDiskConfig4(c *V4ServerConf)
	// WriteDiskConfig6 - copy disk configuration
	WriteDiskConfig6(c *V6ServerConf)

	// Start - start server
	Start() error
	// Stop - stop server
	Stop()
}

// V4ServerConf - server configuration
type V4ServerConf struct {
	Enabled       bool   `yaml:"-"`
	InterfaceName string `yaml:"-"`

	GatewayIP  string `yaml:"gateway_ip"`
	SubnetMask string `yaml:"subnet_mask"`

	// The first & the last IP address for dynamic leases
	// Bytes [0..2] of the last allowed IP address must match the first IP
	RangeStart string `yaml:"range_start"`
	RangeEnd   string `yaml:"range_end"`

	LeaseDuration uint32 `yaml:"lease_duration"` // in seconds

	// IP conflict detector: time (ms) to wait for ICMP reply
	// 0: disable
	ICMPTimeout uint32 `yaml:"icmp_timeout_msec"`

	ipStart    net.IP        // starting IP address for dynamic leases
	ipEnd      net.IP        // ending IP address for dynamic leases
	leaseTime  time.Duration // the time during which a dynamic lease is considered valid
	dnsIPAddrs []net.IP      // IPv4 addresses to return to DHCP clients as DNS server addresses
	routerIP   net.IP        // value for Option Router
	subnetMask net.IPMask    // value for Option SubnetMask

	// Server calls this function when leases data changes
	notify func(uint32)
}

// V6ServerConf - server configuration
type V6ServerConf struct {
	Enabled       bool   `yaml:"-"`
	InterfaceName string `yaml:"-"`

	// The first IP address for dynamic leases
	// The last allowed IP address ends with 0xff byte
	RangeStart string `yaml:"range_start"`

	LeaseDuration uint32 `yaml:"lease_duration"` // in seconds

	ipStart    net.IP        // starting IP address for dynamic leases
	leaseTime  time.Duration // the time during which a dynamic lease is considered valid
	dnsIPAddrs []net.IP      // IPv6 addresses to return to DHCP clients as DNS server addresses

	// Server calls this function when leases data changes
	notify func(uint32)
}
