package dhcpsvc_test

import (
	"context"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// testNetworkDeviceManager is a mock implementation of the
// [dhcpsvc.NetworkDeviceManager] interface.
//
// TODO(e.burkov):  Move to aghtest.
type testNetworkDeviceManager struct {
	onOpen func(
		ctx context.Context,
		conf *dhcpsvc.NetworkDeviceConfig,
	) (nd dhcpsvc.NetworkDevice, err error)
}

// Open implements the [NetworkDeviceManager] interface for
// *testNetworkDeviceManager.
func (ndm *testNetworkDeviceManager) Open(
	ctx context.Context,
	conf *dhcpsvc.NetworkDeviceConfig,
) (dev dhcpsvc.NetworkDevice, err error) {
	return ndm.onOpen(ctx, conf)
}

// testNetworkDevice is a mock implementation of the [dhcpsvc.NetworkDevice]
// interface.
//
// TODO(e.burkov):  Move to aghtest.
type testNetworkDevice struct {
	onReadPacketData  func() (data []byte, ci gopacket.CaptureInfo, err error)
	onLinkType        func() (lt layers.LinkType)
	onWritePacketData func(data []byte) (err error)
}

// ReadPacketData implements the [dhcpsvc.NetworkDevice] interface for
// *testNetworkDevice.
func (nd *testNetworkDevice) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return nd.onReadPacketData()
}

// WritePacketData implements the [dhcpsvc.NetworkDevice] interface for
// *testNetworkDevice.
func (nd *testNetworkDevice) WritePacketData(data []byte) (err error) {
	return nd.onWritePacketData(data)
}

// LinkType implements the [dhcpsvc.NetworkDevice] interface for
// *testNetworkDevice.
func (nd *testNetworkDevice) LinkType() (lt layers.LinkType) {
	return nd.onLinkType()
}
