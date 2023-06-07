//go:build darwin || freebsd || openbsd

package dhcpd

import (
	"net"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	//lint:ignore SA1019 See the TODO in go.mod.
	"github.com/mdlayher/raw"
)

func TestDHCPConn_WriteTo_common(t *testing.T) {
	respData := (&dhcpv4.DHCPv4{}).ToBytes()
	udpAddr := &net.UDPAddr{
		IP:   net.IP{1, 2, 3, 4},
		Port: dhcpv4.ClientPort,
	}

	t.Run("unicast_ip", func(t *testing.T) {
		writeTo := func(_ []byte, addr net.Addr) (_ int, _ error) {
			assert.Equal(t, udpAddr, addr)

			return 0, nil
		}

		conn := &dhcpConn{udpConn: &fakePacketConn{writeTo: writeTo}}

		_, err := conn.WriteTo(respData, udpAddr)
		assert.NoError(t, err)
	})

	t.Run("unexpected_addr_type", func(t *testing.T) {
		type unexpectedAddrType struct {
			net.Addr
		}

		conn := &dhcpConn{}
		n, err := conn.WriteTo(nil, &unexpectedAddrType{})
		require.Error(t, err)

		testutil.AssertErrorMsg(t, "addr has an unexpected type *dhcpd.unexpectedAddrType", err)
		assert.Zero(t, n)
	})
}

func TestBuildEtherPkt(t *testing.T) {
	conn := &dhcpConn{
		srcMAC: net.HardwareAddr{1, 2, 3, 4, 5, 6},
		srcIP:  net.IP{1, 2, 3, 4},
	}
	peer := &dhcpUnicastAddr{
		Addr:   raw.Addr{HardwareAddr: net.HardwareAddr{6, 5, 4, 3, 2, 1}},
		yiaddr: net.IP{4, 3, 2, 1},
	}
	payload := (&dhcpv4.DHCPv4{}).ToBytes()

	t.Run("success", func(t *testing.T) {
		pkt, err := conn.buildEtherPkt(payload, peer)
		require.NoError(t, err)

		assert.NotEmpty(t, pkt)

		actualPkt := gopacket.NewPacket(pkt, layers.LayerTypeEthernet, gopacket.DecodeOptions{
			NoCopy: true,
		})
		require.NotNil(t, actualPkt)

		wantTypes := []gopacket.LayerType{
			layers.LayerTypeEthernet,
			layers.LayerTypeIPv4,
			layers.LayerTypeUDP,
			layers.LayerTypeDHCPv4,
		}
		actualLayers := actualPkt.Layers()
		require.Len(t, actualLayers, len(wantTypes))

		for i, wantType := range wantTypes {
			layer := actualLayers[i]
			require.NotNil(t, layer)

			assert.Equal(t, wantType, layer.LayerType())
		}
	})

	t.Run("bad_payload", func(t *testing.T) {
		// Create an invalid DHCP packet.
		invalidPayload := []byte{1, 2, 3, 4}
		pkt, err := conn.buildEtherPkt(invalidPayload, peer)
		require.NoError(t, err)

		assert.NotEmpty(t, pkt)
	})

	t.Run("serializing_error", func(t *testing.T) {
		// Create a peer with invalid MAC.
		badPeer := &dhcpUnicastAddr{
			Addr:   raw.Addr{HardwareAddr: net.HardwareAddr{5, 4, 3, 2, 1}},
			yiaddr: net.IP{4, 3, 2, 1},
		}

		pkt, err := conn.buildEtherPkt(payload, badPeer)
		require.Error(t, err)

		assert.Empty(t, pkt)
	})
}

func TestV4Server_Send(t *testing.T) {
	s := &v4Server{}

	var (
		defaultIP = net.IP{99, 99, 99, 99}
		knownIP   = net.IP{4, 2, 4, 2}
		knownMAC  = net.HardwareAddr{6, 5, 4, 3, 2, 1}
	)

	defaultPeer := &net.UDPAddr{
		IP: defaultIP,
		// Use neither client nor server port to check it actually
		// changed.
		Port: dhcpv4.ClientPort + dhcpv4.ServerPort,
	}
	defaultResp := &dhcpv4.DHCPv4{}

	testCases := []struct {
		want net.Addr
		req  *dhcpv4.DHCPv4
		resp *dhcpv4.DHCPv4
		name string
	}{{
		name: "giaddr",
		req:  &dhcpv4.DHCPv4{GatewayIPAddr: knownIP},
		resp: defaultResp,
		want: &net.UDPAddr{
			IP:   knownIP,
			Port: dhcpv4.ServerPort,
		},
	}, {
		name: "nak",
		req:  &dhcpv4.DHCPv4{},
		resp: &dhcpv4.DHCPv4{
			Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeNak),
			),
		},
		want: defaultPeer,
	}, {
		name: "ciaddr",
		req:  &dhcpv4.DHCPv4{ClientIPAddr: knownIP},
		resp: &dhcpv4.DHCPv4{},
		want: &net.UDPAddr{
			IP:   knownIP,
			Port: dhcpv4.ClientPort,
		},
	}, {
		name: "chaddr",
		req:  &dhcpv4.DHCPv4{ClientHWAddr: knownMAC},
		resp: &dhcpv4.DHCPv4{YourIPAddr: knownIP},
		want: &dhcpUnicastAddr{
			Addr:   raw.Addr{HardwareAddr: knownMAC},
			yiaddr: knownIP,
		},
	}, {
		name: "who_are_you",
		req:  &dhcpv4.DHCPv4{},
		resp: &dhcpv4.DHCPv4{},
		want: defaultPeer,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &fakePacketConn{
				writeTo: func(_ []byte, addr net.Addr) (_ int, _ error) {
					assert.Equal(t, tc.want, addr)

					return 0, nil
				},
			}

			s.send(cloneUDPAddr(defaultPeer), conn, tc.req, tc.resp)
		})
	}

	t.Run("giaddr_nak", func(t *testing.T) {
		req := &dhcpv4.DHCPv4{
			GatewayIPAddr: knownIP,
		}
		// Ensure the request is for unicast.
		req.SetUnicast()
		resp := &dhcpv4.DHCPv4{
			Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeNak),
			),
		}
		want := &net.UDPAddr{
			IP:   req.GatewayIPAddr,
			Port: dhcpv4.ServerPort,
		}

		conn := &fakePacketConn{
			writeTo: func(_ []byte, addr net.Addr) (n int, err error) {
				assert.Equal(t, want, addr)

				return 0, nil
			},
		}

		s.send(cloneUDPAddr(defaultPeer), conn, req, resp)
		assert.True(t, resp.IsBroadcast())
	})
}
