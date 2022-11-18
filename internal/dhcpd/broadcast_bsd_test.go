//go:build freebsd || openbsd

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
	var peer *net.UDPAddr

	udpConn := &fakePacketConn{
		writeTo: func(p []byte, addr net.Addr) (n int, err error) {
			udpPeer, ok := addr.(*net.UDPAddr)
			require.True(t, ok)

			peer = cloneUDPAddr(udpPeer)

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

	assert.EqualValues(t, respData, b.Bytes())
	assert.Equal(t, &net.UDPAddr{
		IP:   conn.bcastIP,
		Port: defaultPeer.Port,
	}, peer)
}
