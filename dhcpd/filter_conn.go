package dhcpd

import (
	"net"

	"github.com/joomcode/errorx"
	"golang.org/x/net/ipv4"
)

// filterConn listens to 0.0.0.0:67, but accepts packets only from specific interface
// This is necessary for DHCP daemon to work, since binding to IP address doesn't
// us access to see Discover/Request packets from clients.
//
// TODO: on windows, controlmessage does not work, try to find out another way
// https://github.com/golang/net/blob/master/ipv4/payload.go#L13
type filterConn struct {
	iface net.Interface
	conn  *ipv4.PacketConn
}

func newFilterConn(iface net.Interface, address string) (*filterConn, error) {
	c, err := net.ListenPacket("udp4", address)
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't listen to %s on UDP4", address)
	}

	p := ipv4.NewPacketConn(c)
	err = p.SetControlMessage(ipv4.FlagInterface, true)
	if err != nil {
		c.Close()
		return nil, errorx.Decorate(err, "Couldn't set control message FlagInterface on connection")
	}

	return &filterConn{iface: iface, conn: p}, nil
}

func (f *filterConn) ReadFrom(b []byte) (int, net.Addr, error) {
	for { // read until we find a suitable packet
		n, cm, addr, err := f.conn.ReadFrom(b)
		if err != nil {
			return 0, addr, errorx.Decorate(err, "Error when reading from socket")
		}
		if cm == nil {
			// no controlmessage was passed, so pass the packet to the caller
			return n, addr, nil
		}
		if cm.IfIndex == f.iface.Index {
			return n, addr, nil
		}
		// packet doesn't match criteria, drop it
	}
}

func (f *filterConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	cm := ipv4.ControlMessage{
		IfIndex: f.iface.Index,
	}
	return f.conn.WriteTo(b, &cm, addr)
}

func (f *filterConn) Close() error {
	return f.conn.Close()
}
