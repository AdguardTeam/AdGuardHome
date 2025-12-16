package dhcpsvc

import (
	"context"

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
//
// TODO(e.burkov):  Add device closing method.
type NetworkDeviceManager interface {
	// Open opens a network device.  conf must be valid.
	Open(ctx context.Context, conf *NetworkDeviceConfig) (dev NetworkDevice, err error)
}

// NetworkDevice provides reading and writing packets to a network interface.
type NetworkDevice interface {
	gopacket.PacketDataSource

	// LinkType returns the link type of the network interface.
	LinkType() (lt layers.LinkType)

	// WritePacketData writes a serialized packet to the network interface.
	WritePacketData(data []byte) (err error)
}

// frameData stores the Ethernet and IPv4 layers of the incoming packet, and
// the network device that the packet was received from.
type frameData struct {
	ether  *layers.Ethernet
	ip     *layers.IPv4
	device NetworkDevice
}
