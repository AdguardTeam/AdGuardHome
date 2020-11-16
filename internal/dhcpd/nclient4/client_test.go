// Copyright 2018 the u-root Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux
// github.com/hugelgupf/socketpair is Linux-only
// +build go1.12

package nclient4

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/hugelgupf/socketpair"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

type handler struct {
	mu       sync.Mutex
	received []*dhcpv4.DHCPv4

	// Each received packet can have more than one response (in theory,
	// from different servers sending different Advertise, for example).
	responses [][]*dhcpv4.DHCPv4
}

func (h *handler) handle(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.received = append(h.received, m)

	if len(h.responses) > 0 {
		for _, resp := range h.responses[0] {
			_, _ = conn.WriteTo(resp.ToBytes(), peer)
		}
		h.responses = h.responses[1:]
	}
}

func serveAndClient(ctx context.Context, responses [][]*dhcpv4.DHCPv4, opts ...ClientOpt) (*Client, net.PacketConn) {
	// Fake PacketConn connection.
	clientRawConn, serverRawConn, err := socketpair.PacketSocketPair()
	if err != nil {
		panic(err)
	}

	clientConn := NewBroadcastUDPConn(clientRawConn, &net.UDPAddr{Port: ClientPort})
	serverConn := NewBroadcastUDPConn(serverRawConn, &net.UDPAddr{Port: ServerPort})

	o := []ClientOpt{WithRetry(1), WithTimeout(2 * time.Second)}
	o = append(o, opts...)
	mc, err := NewWithConn(clientConn, net.HardwareAddr{0xa, 0xb, 0xc, 0xd, 0xe, 0xf}, o...)
	if err != nil {
		panic(err)
	}

	h := &handler{responses: responses}
	s, err := server4.NewServer("", nil, h.handle, server4.WithConn(serverConn))
	if err != nil {
		panic(err)
	}
	go func() {
		_ = s.Serve()
	}()

	return mc, serverConn
}

func ComparePacket(got *dhcpv4.DHCPv4, want *dhcpv4.DHCPv4) error {
	if got == nil && got == want {
		return nil
	}
	if (want == nil || got == nil) && (got != want) {
		return fmt.Errorf("packet got %v, want %v", got, want)
	}
	if !bytes.Equal(got.ToBytes(), want.ToBytes()) {
		return fmt.Errorf("packet got %v, want %v", got, want)
	}
	return nil
}

func pktsExpected(got []*dhcpv4.DHCPv4, want []*dhcpv4.DHCPv4) error {
	if len(got) != len(want) {
		return fmt.Errorf("got %d packets, want %d packets", len(got), len(want))
	}

	for i := range got {
		if err := ComparePacket(got[i], want[i]); err != nil {
			return err
		}
	}
	return nil
}

func newPacketWeirdHWAddr(op dhcpv4.OpcodeType, xid dhcpv4.TransactionID) *dhcpv4.DHCPv4 {
	p, err := dhcpv4.New()
	if err != nil {
		panic(fmt.Sprintf("newpacket: %v", err))
	}
	p.OpCode = op
	p.TransactionID = xid
	p.ClientHWAddr = net.HardwareAddr{0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 1, 2, 3, 4, 5, 6}
	return p
}

func newPacket(op dhcpv4.OpcodeType, xid dhcpv4.TransactionID) *dhcpv4.DHCPv4 {
	p, err := dhcpv4.New()
	if err != nil {
		panic(fmt.Sprintf("newpacket: %v", err))
	}
	p.OpCode = op
	p.TransactionID = xid
	p.ClientHWAddr = net.HardwareAddr{0xa, 0xb, 0xc, 0xd, 0xe, 0xf}
	return p
}

func withBufferCap(n int) ClientOpt {
	return func(c *Client) (err error) {
		c.bufferCap = n
		return
	}
}

func TestSendAndRead(t *testing.T) {
	for _, tt := range []struct {
		desc   string
		send   *dhcpv4.DHCPv4
		server []*dhcpv4.DHCPv4

		// If want is nil, we assume server[0] contains what is wanted.
		want    *dhcpv4.DHCPv4
		wantErr error
	}{
		{
			desc: "two response packets",
			send: newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
			server: []*dhcpv4.DHCPv4{
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
			},
			want: newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
		},
		{
			desc: "one response packet",
			send: newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
			server: []*dhcpv4.DHCPv4{
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
			},
			want: newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
		},
		{
			desc: "one response packet, one invalid XID, one invalid opcode, one invalid hwaddr",
			send: newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
			server: []*dhcpv4.DHCPv4{
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x77, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacketWeirdHWAddr(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
			},
			want: newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
		},
		{
			desc: "discard wrong XID",
			send: newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
			server: []*dhcpv4.DHCPv4{
				newPacket(dhcpv4.OpcodeBootReply, [4]byte{0, 0, 0, 0}),
			},
			want:    nil, // Explicitly empty.
			wantErr: ErrNoResponse,
		},
		{
			desc:    "no response, timeout",
			send:    newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
			wantErr: ErrNoResponse,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// Both server and client only get 2 seconds.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			mc, _ := serveAndClient(ctx, [][]*dhcpv4.DHCPv4{tt.server},
				// Use an unbuffered channel to make sure we
				// have no deadlocks.
				withBufferCap(0))
			defer mc.Close()

			rcvd, err := mc.SendAndRead(context.Background(), DefaultServers, tt.send, nil)
			if err != tt.wantErr {
				t.Error(err)
			}

			if err := ComparePacket(rcvd, tt.want); err != nil {
				t.Errorf("got unexpected packets: %v", err)
			}
		})
	}
}

func TestParallelSendAndRead(t *testing.T) {
	pkt := newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33})

	// Both the server and client only get 2 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mc, _ := serveAndClient(ctx, [][]*dhcpv4.DHCPv4{},
		WithTimeout(10*time.Second),
		// Use an unbuffered channel to make sure nothing blocks.
		withBufferCap(0))
	defer mc.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := mc.SendAndRead(context.Background(), DefaultServers, pkt, nil); err != ErrNoResponse {
			t.Errorf("SendAndRead(%v) = %v, want %v", pkt, err, ErrNoResponse)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(4 * time.Second)

		if err := mc.Close(); err != nil {
			t.Errorf("closing failed: %v", err)
		}
	}()

	wg.Wait()
}

func TestReuseXID(t *testing.T) {
	pkt := newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33})

	// Both the server and client only get 2 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mc, _ := serveAndClient(ctx, [][]*dhcpv4.DHCPv4{})
	defer mc.Close()

	if _, err := mc.SendAndRead(context.Background(), DefaultServers, pkt, nil); err != ErrNoResponse {
		t.Errorf("SendAndRead(%v) = %v, want %v", pkt, err, ErrNoResponse)
	}

	if _, err := mc.SendAndRead(context.Background(), DefaultServers, pkt, nil); err != ErrNoResponse {
		t.Errorf("SendAndRead(%v) = %v, want %v", pkt, err, ErrNoResponse)
	}
}

func TestSimpleSendAndReadDiscardGarbage(t *testing.T) {
	pkt := newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33})

	responses := newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33})

	// Both the server and client only get 2 seconds.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mc, udpConn := serveAndClient(ctx, [][]*dhcpv4.DHCPv4{{responses}})
	defer mc.Close()

	// Too short for valid DHCPv4 packet.
	_, _ = udpConn.WriteTo([]byte{0x01}, nil)
	_, _ = udpConn.WriteTo([]byte{0x01, 0x2}, nil)

	rcvd, err := mc.SendAndRead(ctx, DefaultServers, pkt, nil)
	if err != nil {
		t.Errorf("SendAndRead(%v) = %v, want nil", pkt, err)
	}

	if err := ComparePacket(rcvd, responses); err != nil {
		t.Errorf("got unexpected packets: %v", err)
	}
}

func TestMultipleSendAndRead(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		send    []*dhcpv4.DHCPv4
		server  [][]*dhcpv4.DHCPv4
		wantErr []error
	}{
		{
			desc: "two requests, two responses",
			send: []*dhcpv4.DHCPv4{
				newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x33, 0x33, 0x33, 0x33}),
				newPacket(dhcpv4.OpcodeBootRequest, [4]byte{0x44, 0x44, 0x44, 0x44}),
			},
			server: [][]*dhcpv4.DHCPv4{
				[]*dhcpv4.DHCPv4{ // Response for first packet.
					newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x33, 0x33, 0x33, 0x33}),
				},
				[]*dhcpv4.DHCPv4{ // Response for second packet.
					newPacket(dhcpv4.OpcodeBootReply, [4]byte{0x44, 0x44, 0x44, 0x44}),
				},
			},
			wantErr: []error{
				nil,
				nil,
			},
		},
	} {
		// Both server and client only get 2 seconds.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		mc, _ := serveAndClient(ctx, tt.server)
		defer mc.Close()

		for i, send := range tt.send {
			ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			rcvd, err := mc.SendAndRead(ctx, DefaultServers, send, nil)

			if wantErr := tt.wantErr[i]; err != wantErr {
				t.Errorf("SendAndReadOne(%v): got %v, want %v", send, err, wantErr)
			}
			if err := pktsExpected([]*dhcpv4.DHCPv4{rcvd}, tt.server[i]); err != nil {
				t.Errorf("got unexpected packets: %v", err)
			}
		}
	}
}
