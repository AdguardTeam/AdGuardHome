package dhcpsvc_test

import (
	"context"
	"io"
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/require"
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

// type check
var _ dhcpsvc.NetworkDeviceManager = (*testNetworkDeviceManager)(nil)

// Open implements the [dhcpsvc.NetworkDeviceManager] interface for
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
	onClose           func() (err error)
	onAddresses       func() (ips []netip.Addr)
	onHardwareAddr    func() (hw net.HardwareAddr)
	onLinkType        func() (lt layers.LinkType)
	onWritePacketData func(data []byte) (err error)
}

// type check
var _ dhcpsvc.NetworkDevice = (*testNetworkDevice)(nil)

// ReadPacketData implements the [gopacket.PacketDataSource] interface for
// *testNetworkDevice.
func (nd *testNetworkDevice) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return nd.onReadPacketData()
}

// Close implements the [io.Closer] interface for *testNetworkDevice.
func (nd *testNetworkDevice) Close() (err error) {
	return nd.onClose()
}

// Addresses implements the [dhcpsvc.NetworkDevice] interface for
// *testNetworkDevice.
func (nd *testNetworkDevice) Addresses() (ips []netip.Addr) {
	return nd.onAddresses()
}

// HardwareAddr implements the [dhcpsvc.NetworkDevice] interface for
// *testNetworkDevice.
func (nd *testNetworkDevice) HardwareAddr() (hw net.HardwareAddr) {
	return nd.onHardwareAddr()
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

// newTestNetworkDeviceAndManager creates a network device manager for testing
// and returns it along with the device it opens.  It requires that device
// opened have [testIfaceName] name.  The device itself has a link type
// [layers.LinkTypeEthernet] and a hardware address [testIfaceHWAddr].  Incoming
// packets are received from inCh and outgoing packets are sent to outCh.
func newTestNetworkDeviceAndManager(
	tb testing.TB,
	addr netip.Addr,
) (
	ndMgr *testNetworkDeviceManager,
	dev *testNetworkDevice,
	inCh chan<- gopacket.Packet,
	outCh <-chan []byte,
) {
	tb.Helper()

	dev, inCh, outCh = newTestNetworkDevice(tb, addr)

	pt := testutil.NewPanicT(tb)

	onOpen := func(
		_ context.Context,
		conf *dhcpsvc.NetworkDeviceConfig,
	) (nd dhcpsvc.NetworkDevice, err error) {
		require.Equal(pt, testIfaceName, conf.Name)

		return dev, nil
	}

	ndMgr = &testNetworkDeviceManager{
		onOpen: onOpen,
	}

	return ndMgr, dev, inCh, outCh
}

// newTestNetworkDevice creates a network device for testing.  It has a link
// type [layers.LinkTypeEthernet] and a hardware address [testIfaceHWAddr].
// Incoming packets are received from inCh and outgoing packets are sent to
// outCh.
func newTestNetworkDevice(
	tb testing.TB,
	addr netip.Addr,
) (nd *testNetworkDevice, inCh chan<- gopacket.Packet, outCh <-chan []byte) {
	tb.Helper()

	in := make(chan gopacket.Packet)
	out := make(chan []byte)

	pt := testutil.NewPanicT(tb)

	onReadPacketData := func() (data []byte, ci gopacket.CaptureInfo, err error) {
		pkt, ok := testutil.RequireReceive(pt, in, testTimeout)
		if !ok {
			return nil, gopacket.CaptureInfo{}, io.EOF
		}

		data = pkt.Data()
		ci = gopacket.CaptureInfo{
			Length:        len(data),
			CaptureLength: len(data),
		}

		return data, ci, nil
	}

	onClose := func() (err error) {
		close(in)
		close(out)

		return nil
	}

	onAddresses := func() (ips []netip.Addr) {
		return []netip.Addr{addr}
	}

	onHardwareAddr := func() (hw net.HardwareAddr) {
		return testIfaceHWAddr
	}

	onLinkType := func() (lt layers.LinkType) {
		return layers.LinkTypeEthernet
	}

	onWritePacketData := func(data []byte) (err error) {
		testutil.RequireSend(pt, out, data, testTimeout)

		return nil
	}

	return &testNetworkDevice{
		onReadPacketData:  onReadPacketData,
		onClose:           onClose,
		onAddresses:       onAddresses,
		onHardwareAddr:    onHardwareAddr,
		onLinkType:        onLinkType,
		onWritePacketData: onWritePacketData,
	}, in, out
}

// newTestNetworkDeviceAndManager creates a network device manager for testing
// and returns it.  It requires that device opened have [testIfaceName] name.
// The device itself has a link type [layers.LinkTypeEthernet] and a hardware
// address [testIfaceHWAddr].  Incoming packets are received from inCh and
// outgoing packets are sent to outCh.
func newTestNetworkDeviceManager(
	tb testing.TB,
	addr netip.Addr,
) (ndMgr *testNetworkDeviceManager, inCh chan<- gopacket.Packet, outCh <-chan []byte) {
	tb.Helper()

	dev, inCh, outCh := newTestNetworkDevice(tb, addr)

	pt := testutil.NewPanicT(tb)

	onOpen := func(
		_ context.Context,
		conf *dhcpsvc.NetworkDeviceConfig,
	) (nd dhcpsvc.NetworkDevice, err error) {
		require.Equal(pt, testIfaceName, conf.Name)

		return dev, nil
	}

	ndMgr = &testNetworkDeviceManager{
		onOpen: onOpen,
	}

	return ndMgr, inCh, outCh
}

// unexpectedWritePacketData is a helper function that panics if called, used to
// ensure that no packet data is written to the network device in tests.
func unexpectedWritePacketData(data []byte) (_ error) {
	panic(testutil.UnexpectedCall(data))
}
