package dhcpd

import (
	"net"
	"time"
)

// DHCPServer - DHCP server interface
type DHCPServer interface {
	ResetLeases(leases []*Lease)
	GetLeases(flags int) []Lease
	GetLeasesRef() []*Lease
	AddStaticLease(lease Lease) error
	RemoveStaticLease(l Lease) error
	FindMACbyIP(ip net.IP) net.HardwareAddr

	WriteDiskConfig4(c *V4ServerConf)
	WriteDiskConfig6(c *V6ServerConf)

	Start() error
	Stop()
	Reset()
}

// V4ServerConf - server configuration
type V4ServerConf struct {
	Enabled       bool   `yaml:"enabled"`
	InterfaceName string `yaml:"interface_name"` // eth0, en0 and so on
	GatewayIP     string `yaml:"gateway_ip"`
	SubnetMask    string `yaml:"subnet_mask"`
	RangeStart    string `yaml:"range_start"`
	RangeEnd      string `yaml:"range_end"`
	LeaseDuration uint32 `yaml:"lease_duration"` // in seconds

	// IP conflict detector: time (ms) to wait for ICMP reply.
	// 0: disable
	ICMPTimeout uint32 `yaml:"icmp_timeout_msec"`

	ipStart    net.IP
	ipEnd      net.IP
	leaseTime  time.Duration
	dnsIPAddrs []net.IP // IPv4 addresses to return to DHCP clients as DNS server addresses
	routerIP   net.IP
	subnetMask net.IPMask

	notify func(uint32)
}

// V6ServerConf - server configuration
type V6ServerConf struct {
	Enabled       bool   `yaml:"enabled"`
	InterfaceName string `yaml:"interface_name"`
	RangeStart    string `yaml:"range_start"`
	LeaseDuration uint32 `yaml:"lease_duration"` // in seconds

	ipStart    net.IP
	leaseTime  time.Duration
	dnsIPAddrs []net.IP // IPv6 addresses to return to DHCP clients as DNS server addresses

	notify func(uint32)
}
