//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"net"
	"strings"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/mdlayher/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		assert.True(t, strings.Contains(err.Error(), "peer is of unexpected type"))
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

	t.Run("non-serializable", func(t *testing.T) {
		// Create an invalid DHCP packet.
		invalidPayload := []byte{1, 2, 3, 4}
		pkt, err := conn.buildEtherPkt(invalidPayload, nil)
		require.Error(t, err)

		assert.ErrorIs(t, err, errInvalidPktDHCP)
		assert.Empty(t, pkt)
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
