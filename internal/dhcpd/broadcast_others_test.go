//go:build aix || darwin || dragonfly || linux || netbsd || solaris
// +build aix darwin dragonfly linux netbsd solaris

package dhcpd

import (
	"bytes"
	"net"
	"testing"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV4Server_Send_broadcast(t *testing.T) {
	b := &bytes.Buffer{}
	var peers []*net.UDPAddr

	conn := &fakePacketConn{
		writeTo: func(p []byte, addr net.Addr) (n int, err error) {
			udpPeer, ok := addr.(*net.UDPAddr)
			require.True(t, ok)

			peers = append(peers, cloneUDPAddr(udpPeer))

			n, err = b.Write(p)
			require.NoError(t, err)

			return n, nil
		},
	}

	defaultPeer := &net.UDPAddr{
		IP: net.IP{1, 2, 3, 4},
		// Use neither client nor server port.
		Port: 1234,
	}
	s := &v4Server{
		conf: V4ServerConf{
			broadcastIP: net.IP{1, 2, 3, 255},
		},
	}

	testCases := []struct {
		name string
		req  *dhcpv4.DHCPv4
		resp *dhcpv4.DHCPv4
	}{{
		name: "nak",
		req: &dhcpv4.DHCPv4{
			GatewayIPAddr: netutil.IPv4Zero(),
		},
		resp: &dhcpv4.DHCPv4{
			Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeNak),
			),
		},
	}, {
		name: "fully_unspecified",
		req: &dhcpv4.DHCPv4{
			GatewayIPAddr: netutil.IPv4Zero(),
			ClientIPAddr:  netutil.IPv4Zero(),
		},
		resp: &dhcpv4.DHCPv4{
			Options: dhcpv4.OptionsFromList(
				dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer),
			),
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s.send(cloneUDPAddr(defaultPeer), conn, tc.req, tc.resp)

			// The same response is written twice.
			respData := tc.resp.ToBytes()
			assert.EqualValues(t, append(respData, respData...), b.Bytes())

			require.Len(t, peers, 2)

			assert.Equal(t, &net.UDPAddr{
				IP:   defaultPeer.IP,
				Port: defaultPeer.Port,
			}, peers[0])
			assert.Equal(t, &net.UDPAddr{
				IP:   s.conf.broadcastIP,
				Port: defaultPeer.Port,
			}, peers[1])
		})

		b.Reset()
		peers = nil
	}
}
