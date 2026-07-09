package dhcpsvc

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"slices"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
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
	// device is the network device that the packet was received from.  It must
	// not be nil.
	device NetworkDevice

	// ether is the Ethernet layer of the incoming packet.  It must not be nil.
	ether *layers.Ethernet

	// ip is the IPv4 layer of the incoming packet.  It must not be nil.
	ip *layers.IPv4

	// localAddr is the local IP address that the packet was sent to.  It must
	// be a valid IPv4 address assigned to the device.
	localAddr netip.Addr
}

// frameData6 stores the Ethernet and IPv6 layers of the incoming packet, as
// well as the network device that the packet was received from and its address.
type frameData6 struct {
	// device is the network device that the packet was received from.  It must
	// not be nil.
	device NetworkDevice

	// localAddr is the local IP address that the packet was sent to.  It must
	// be a valid IPv6 address assigned to the device.
	localAddr netip.Addr

	// duid is the DHCPv6 DUID constructed of the network device hardware
	// address.  It must not be nil.
	duid *layers.DHCPv6DUID

	// ether is the Ethernet layer of the incoming packet.  It must not be nil.
	ether *layers.Ethernet

	// ip is the IPv6 layer of the incoming packet.  It must not be nil.
	ip *layers.IPv6

	// duidData is the pre-encoded DUID-LL for this server interface.  It is
	// used to match the Server Identifier option in incoming DHCPv6 messages.
	// It must not be nil.
	duidData []byte
}

// newFrameData4 creates a new [frameData4] with layers extracted from pkt.  It
// returns nil if the packet is not an Ethernet or IPv4 packet, or if the
// network device has no addresses.  pkt and dev must not be nil.
func newFrameData4(pkt gopacket.Packet, dev NetworkDevice) (fd *frameData4, err error) {
	addrs := dev.Addresses()
	if len(addrs) == 0 {
		return nil, fmt.Errorf("addresses of network device: %w", errors.ErrEmptyValue)
	}

	ether, err := ethernetFromPacket(pkt, layers.EthernetTypeIPv4)
	if err != nil {
		return nil, fmt.Errorf("extracting ethernet layer: %w", err)
	}

	ipLayer, ok := pkt.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok {
		return nil, fmt.Errorf("extracting ipv4 layer: %w", errors.ErrNoValue)
	}

	addr, ok := netip.AddrFromSlice(ipLayer.DstIP)
	if !ok || !slices.Contains(addrs, addr) {
		addr = addrs[0]
	}

	return &frameData4{
		ether:     ether,
		ip:        ipLayer,
		device:    dev,
		localAddr: addr,
	}, nil
}

// newFrameData6 creates a new [frameData6] with layers extracted from pkt.  It
// returns nil if the packet is not an Ethernet or IPv6 packet, or if the
// network device has no addresses.  pkt, dev, and duid must not be nil.
func newFrameData6(
	pkt gopacket.Packet,
	dev NetworkDevice,
	duid *layers.DHCPv6DUID,
) (fd *frameData6, err error) {
	addrs := dev.Addresses()
	if len(addrs) == 0 {
		return nil, fmt.Errorf("addresses of network device: %w", errors.ErrEmptyValue)
	}

	ether, err := ethernetFromPacket(pkt, layers.EthernetTypeIPv6)
	if err != nil {
		return nil, fmt.Errorf("extracting ethernet layer: %w", err)
	}

	ipLayer, ok := pkt.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
	if !ok {
		return nil, fmt.Errorf("extracting ipv6 layer: %w", errors.ErrNoValue)
	}

	addr, ok := netip.AddrFromSlice(ipLayer.DstIP)
	if !ok || !slices.Contains(addrs, addr) {
		addr = addrs[0]
	}

	return &frameData6{
		ether:     ether,
		ip:        ipLayer,
		duid:      duid,
		duidData:  duid.Encode(),
		device:    dev,
		localAddr: addr,
	}, nil
}

// ethernetFromPacket extracts the Ethernet layer from the given packet and
// validates its contents.  pkt must not be nil, expType is the expected type of
// the Ethernet layer.
func ethernetFromPacket(
	pkt gopacket.Packet,
	expType layers.EthernetType,
) (ether *layers.Ethernet, err error) {
	ether, ok := pkt.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		return nil, errors.ErrNoValue
	}

	var errs []error

	err = netutil.ValidateMAC(ether.SrcMAC)
	if err != nil {
		errs = append(errs, fmt.Errorf("source mac: %w", err))
	}

	err = netutil.ValidateMAC(ether.DstMAC)
	if err != nil {
		errs = append(errs, fmt.Errorf("destination mac: %w", err))
	}

	err = validate.Equal("type", ether.EthernetType, expType)
	if err != nil {
		errs = append(errs, fmt.Errorf("ethernet type: %w", err))
	}

	return ether, errors.Join(errs...)
}
