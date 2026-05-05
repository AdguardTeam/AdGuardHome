package dhcpsvc

import (
	"context"
	"io"
	"net"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// NetworkDeviceConfig is the configuration for a network device.
type NetworkDeviceConfig struct {
	// Name is the name of the network device.  It must be a valid interface
	// name on the system.
	Name string
}

// Validate implements the [validate.Interface] interface for
// *NetworkDeviceConfig.
func (conf *NetworkDeviceConfig) Validate() (err error) {
	if conf == nil {
		return errors.ErrNoValue
	}

	return validate.NotEmpty("Name", conf.Name)
}

// NetworkDeviceManager creates and manages network devices.
type NetworkDeviceManager interface {
	// Open opens a network device.  conf must be valid.
	//
	// An attempt to open the same device multiple times may return an error.
	Open(ctx context.Context, conf *NetworkDeviceConfig) (dev NetworkDevice, err error)
}

// EmptyNetworkDeviceManager is an empty implementation of
// [NetworkDeviceManager].
type EmptyNetworkDeviceManager struct{}

// type check
var _ NetworkDeviceManager = EmptyNetworkDeviceManager{}

// Open implements the [NetworkDeviceManager] interface for
// [EmptyNetworkDeviceManager].  It always returns [EmptyNetworkDevice].
func (EmptyNetworkDeviceManager) Open(
	_ context.Context,
	_ *NetworkDeviceConfig,
) (nd NetworkDevice, err error) {
	return nil, nil
}

// NetworkDevice provides an ability of reading and writing packets to a network
// interface.  It used to generalize implementations for different platforms and
// to simplify testing.
//
// It's based on [pcap.Handle].
type NetworkDevice interface {
	gopacket.PacketDataSource

	// No methods of a device should be called after Close.
	io.Closer

	// Addresses returns all IP addresses assigned to the device.  It must
	// return at least one valid address, unless the implementation documents
	// the opposite.
	Addresses() (ips []netip.Addr)

	// HardwareAddr returns the hardware (MAC) address of the device.  It must
	// return a valid hardware address, unless the implementation documents the
	// opposite.
	HardwareAddr() (hw net.HardwareAddr)

	// LinkType returns the link type of the network interface.  It must return
	// a valid link type, unless the implementation documents the opposite.
	LinkType() (lt layers.LinkType)

	// WritePacketData writes a serialized packet to the network interface.
	WritePacketData(data []byte) (err error)
}

// EmptyNetworkDevice is an empty implementation of NetworkDevice.
type EmptyNetworkDevice struct{}

// type check
var _ NetworkDevice = EmptyNetworkDevice{}

// ReadPacketData implements the [gopacket.PacketDataSource] interface for
// [EmptyNetworkDevice].  It always returns no data, empty capture info and a
// nil error.
func (EmptyNetworkDevice) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return nil, gopacket.CaptureInfo{}, nil
}

// Close implements the [io.Closer] interface for [EmptyNetworkDevice].  It
// always returns nil.
func (EmptyNetworkDevice) Close() (err error) {
	return nil
}

// Addresses implements the [NetworkDevice] interface for [EmptyNetworkDevice].
// It always returns nil.
func (EmptyNetworkDevice) Addresses() (ips []netip.Addr) {
	return nil
}

// HardwareAddr implements the [NetworkDevice] interface for
// [EmptyNetworkDevice].  It always returns nil.
func (EmptyNetworkDevice) HardwareAddr() (hw net.HardwareAddr) {
	return nil
}

// LinkType implements the [NetworkDevice] interface for [EmptyNetworkDevice].
// It always returns [layers.LinkTypeNull].
func (EmptyNetworkDevice) LinkType() (lt layers.LinkType) {
	return layers.LinkTypeNull
}

// WritePacketData implements the [NetworkDevice] interface for
// [EmptyNetworkDevice].  It always returns nil.
func (EmptyNetworkDevice) WritePacketData(_ []byte) (err error) {
	return nil
}

// frameData4 stores the Ethernet and IPv4 layers of the incoming packet, as
// well as the network device that the packet was received from and its address.
type frameData4 struct {
	// ether is the Ethernet layer of the incoming packet.  It must not be nil.
	ether *layers.Ethernet

	// ip is the IPv4 layer of the incoming packet.  It must not be nil.
	ip *layers.IPv4

	// device is the network device that the packet was received from.  It must
	// not be nil.
	device NetworkDevice

	// localAddr is the local IP address that the packet was sent to.  It must
	// be a valid IPv4 address assigned to the device.
	localAddr netip.Addr
}

// frameData6 stores the Ethernet and IPv6 layers of the incoming packet, as
// well as the network device that the packet was received from and its address.
type frameData6 struct {
	// ether is the Ethernet layer of the incoming packet.  It must not be nil.
	ether *layers.Ethernet

	// ip is the IPv6 layer of the incoming packet.  It must not be nil.
	ip *layers.IPv6

	// duid is the DHCPv6 DUID constructed of the network device hardware
	// address.  It must not be nil.
	duid *layers.DHCPv6DUID

	// device is the network device that the packet was received from.  It must
	// not be nil.
	device NetworkDevice

	// localAddr is the local IP address that the packet was sent to.  It must
	// be a valid IPv6 address assigned to the device.
	localAddr netip.Addr
}
