package dhcpsvc_test

import (
	"context"
	"io"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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

// newTestNetworkDeviceManager creates a network device manager for testing.  It
// requires that device opened have a deviceName.  The device itself has a link
// type [layers.LinkTypeEthernet].  Incoming packets are received from inCh and
// outgoing packets are sent to outCh.
func newTestNetworkDeviceManager(
	tb testing.TB,
	deviceName string,
	addr netip.Addr,
) (ndMgr dhcpsvc.NetworkDeviceManager, inCh chan<- gopacket.Packet, outCh <-chan []byte) {
	tb.Helper()

	isOpened := &atomic.Bool{}

	dev, inCh, outCh := newTestNetworkDevice(tb, addr, isOpened)

	pt := testutil.NewPanicT(tb)

	onOpen := func(
		_ context.Context,
		conf *dhcpsvc.NetworkDeviceConfig,
	) (nd dhcpsvc.NetworkDevice, err error) {
		isOpened.Store(true)
		require.Equal(pt, deviceName, conf.Name)

		return dev, nil
	}

	ndMgr = &testNetworkDeviceManager{
		onOpen: onOpen,
	}

	return ndMgr, inCh, outCh
}

// newTestNetworkDevice creates a network device for testing.  It has a link
// type [layers.LinkTypeEthernet].  Incoming packets are received from inCh and
// outgoing packets are sent to outCh.
func newTestNetworkDevice(
	tb testing.TB,
	addr netip.Addr,
	isOpened *atomic.Bool,
) (nd dhcpsvc.NetworkDevice, inCh chan<- gopacket.Packet, outCh <-chan []byte) {
	tb.Helper()

	in := make(chan gopacket.Packet)
	out := make(chan []byte)

	pt := testutil.NewPanicT(tb)

	onReadPacketData := func() (data []byte, ci gopacket.CaptureInfo, err error) {
		pkt, ok := testutil.RequireReceive(pt, in, testTimeout)
		require.Equalf(pt, isOpened.Load(), ok, "unexpected receive: %v", pkt)

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
		isOpened.Store(false)
		close(in)

		return nil
	}

	onAddresses := func() (ips []netip.Addr) {
		return []netip.Addr{addr}
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
		onLinkType:        onLinkType,
		onWritePacketData: onWritePacketData,
	}, in, out
}
