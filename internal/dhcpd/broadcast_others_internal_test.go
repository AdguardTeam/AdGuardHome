//go:build darwin || linux

package dhcpd

import (
	"bytes"
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPConn_Broadcast(t *testing.T) {
	b := &bytes.Buffer{}
	var peers []*net.UDPAddr

	udpConn := &fakePacketConn{
		writeTo: func(p []byte, addr net.Addr) (n int, err error) {
			udpPeer, ok := addr.(*net.UDPAddr)
			require.True(t, ok)

			peers = append(peers, cloneUDPAddr(udpPeer))

			n, err = b.Write(p)
			require.NoError(t, err)

			return n, nil
		},
	}
	conn := &dhcpConn{
		udpConn: udpConn,
		bcastIP: net.IP{1, 2, 3, 255},
	}
	defaultPeer := &net.UDPAddr{
		IP: net.IP{1, 2, 3, 4},
		// Use neither client nor server port.
		Port: 1234,
	}
	respData := (&dhcpv4.DHCPv4{}).ToBytes()

	_, _ = conn.broadcast(respData, cloneUDPAddr(defaultPeer))

	// The same response is written twice but for different peers.
	assert.EqualValues(t, append(respData, respData...), b.Bytes())

	require.Len(t, peers, 2)

	assert.Equal(t, cloneUDPAddr(defaultPeer), peers[0])
	assert.Equal(t, &net.UDPAddr{
		IP:   conn.bcastIP,
		Port: defaultPeer.Port,
	}, peers[1])
}
