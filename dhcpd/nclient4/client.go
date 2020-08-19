// Copyright 2018 the u-root Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.12

// Package nclient4 is a small, minimum-functionality client for DHCPv4.
//
// It only supports the 4-way DHCPv4 Discover-Offer-Request-Ack handshake as
// well as the Request-Ack renewal process.
// Originally from here: github.com/insomniacslk/dhcp/dhcpv4/nclient4
//  with the difference that this package can be built on UNIX (not just Linux),
//  because github.com/mdlayher/raw package supports it.
package nclient4

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

const (
	defaultBufferCap = 5

	// DefaultTimeout is the default value for read-timeout if option WithTimeout is not set
	DefaultTimeout = 5 * time.Second

	// DefaultRetries is amount of retries will be done if no answer was received within read-timeout amount of time
	DefaultRetries = 3

	// MaxMessageSize is the value to be used for DHCP option "MaxMessageSize".
	MaxMessageSize = 1500

	// ClientPort is the port that DHCP clients listen on.
	ClientPort = 68

	// ServerPort is the port that DHCP servers and relay agents listen on.
	ServerPort = 67
)

var (
	// DefaultServers is the address of all link-local DHCP servers and
	// relay agents.
	DefaultServers = &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: ServerPort,
	}
)

var (
	// ErrNoResponse is returned when no response packet is received.
	ErrNoResponse = errors.New("no matching response packet received")

	// ErrNoConn is returned when NewWithConn is called with nil-value as conn.
	ErrNoConn = errors.New("conn is nil")

	// ErrNoIfaceHWAddr is returned when NewWithConn is called with nil-value as ifaceHWAddr
	ErrNoIfaceHWAddr = errors.New("ifaceHWAddr is nil")
)

// pendingCh is a channel associated with a pending TransactionID.
type pendingCh struct {
	// SendAndRead closes done to indicate that it wishes for no more
	// messages for this particular XID.
	done <-chan struct{}

	// ch is used by the receive loop to distribute DHCP messages.
	ch chan<- *dhcpv4.DHCPv4
}

// Logger is a handler which will be used to output logging messages
type Logger interface {
	// PrintMessage print _all_ DHCP messages
	PrintMessage(prefix string, message *dhcpv4.DHCPv4)

	// Printf is use to print the rest debugging information
	Printf(format string, v ...interface{})
}

// EmptyLogger prints nothing
type EmptyLogger struct{}

// Printf is just a dummy function that does nothing
func (e EmptyLogger) Printf(format string, v ...interface{}) {}

// PrintMessage is just a dummy function that does nothing
func (e EmptyLogger) PrintMessage(prefix string, message *dhcpv4.DHCPv4) {}

// Printfer is used for actual output of the logger. For example *log.Logger is a Printfer.
type Printfer interface {
	// Printf is the function for logging output. Arguments are handled in the manner of fmt.Printf.
	Printf(format string, v ...interface{})
}

// ShortSummaryLogger is a wrapper for Printfer to implement interface Logger.
// DHCP messages are printed in the short format.
type ShortSummaryLogger struct {
	// Printfer is used for actual output of the logger
	Printfer
}

// Printf prints a log message as-is via predefined Printfer
func (s ShortSummaryLogger) Printf(format string, v ...interface{}) {
	s.Printfer.Printf(format, v...)
}

// PrintMessage prints a DHCP message in the short format via predefined Printfer
func (s ShortSummaryLogger) PrintMessage(prefix string, message *dhcpv4.DHCPv4) {
	s.Printf("%s: %s", prefix, message)
}

// DebugLogger is a wrapper for Printfer to implement interface Logger.
// DHCP messages are printed in the long format.
type DebugLogger struct {
	// Printfer is used for actual output of the logger
	Printfer
}

// Printf prints a log message as-is via predefined Printfer
func (d DebugLogger) Printf(format string, v ...interface{}) {
	d.Printfer.Printf(format, v...)
}

// PrintMessage prints a DHCP message in the long format via predefined Printfer
func (d DebugLogger) PrintMessage(prefix string, message *dhcpv4.DHCPv4) {
	d.Printf("%s: %s", prefix, message.Summary())
}

// Client is an IPv4 DHCP client.
type Client struct {
	ifaceHWAddr net.HardwareAddr
	conn        net.PacketConn
	timeout     time.Duration
	retry       int
	logger      Logger

	// bufferCap is the channel capacity for each TransactionID.
	bufferCap int

	// serverAddr is the UDP address to send all packets to.
	//
	// This may be an actual broadcast address, or a unicast address.
	serverAddr *net.UDPAddr

	// closed is an atomic bool set to 1 when done is closed.
	closed uint32

	// done is closed to unblock the receive loop.
	done chan struct{}

	// wg protects any spawned goroutines, namely the receiveLoop.
	wg sync.WaitGroup

	pendingMu sync.Mutex
	// pending stores the distribution channels for each pending
	// TransactionID. receiveLoop uses this map to determine which channel
	// to send a new DHCP message to.
	pending map[dhcpv4.TransactionID]*pendingCh
}

// New returns a client usable with an unconfigured interface.
func New(iface string, opts ...ClientOpt) (*Client, error) {
	return new(iface, nil, nil, opts...)
}

// NewWithConn creates a new DHCP client that sends and receives packets on the
// given interface.
func NewWithConn(conn net.PacketConn, ifaceHWAddr net.HardwareAddr, opts ...ClientOpt) (*Client, error) {
	return new(``, conn, ifaceHWAddr, opts...)
}

func new(iface string, conn net.PacketConn, ifaceHWAddr net.HardwareAddr, opts ...ClientOpt) (*Client, error) {
	c := &Client{
		ifaceHWAddr: ifaceHWAddr,
		timeout:     DefaultTimeout,
		retry:       DefaultRetries,
		serverAddr:  DefaultServers,
		bufferCap:   defaultBufferCap,
		conn:        conn,
		logger:      EmptyLogger{},

		done:    make(chan struct{}),
		pending: make(map[dhcpv4.TransactionID]*pendingCh),
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, fmt.Errorf("unable to apply option: %w", err)
		}
	}

	if c.ifaceHWAddr == nil {
		if iface == `` {
			return nil, ErrNoIfaceHWAddr
		}

		i, err := net.InterfaceByName(iface)
		if err != nil {
			return nil, fmt.Errorf("unable to get interface information: %w", err)
		}

		c.ifaceHWAddr = i.HardwareAddr
	}

	if c.conn == nil {
		var err error
		if iface == `` {
			return nil, ErrNoConn
		}
		c.conn, err = NewRawUDPConn(iface, ClientPort) // broadcast
		if err != nil {
			return nil, fmt.Errorf("unable to open a broadcasting socket: %w", err)
		}
	}
	c.wg.Add(1)
	go c.receiveLoop()
	return c, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	// Make sure not to close done twice.
	if !atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		return nil
	}

	err := c.conn.Close()

	// Closing c.done sets off a chain reaction:
	//
	// Any SendAndRead unblocks trying to receive more messages, which
	// means rem() gets called.
	//
	// rem() should be unblocking receiveLoop if it is blocked.
	//
	// receiveLoop should then exit gracefully.
	close(c.done)

	// Wait for receiveLoop to stop.
	c.wg.Wait()

	return err
}

func (c *Client) isClosed() bool {
	return atomic.LoadUint32(&c.closed) != 0
}

func (c *Client) receiveLoop() {
	defer c.wg.Done()
	for {
		// TODO: Clients can send a "max packet size" option in their
		// packets, IIRC. Choose a reasonable size and set it.
		b := make([]byte, MaxMessageSize)
		n, _, err := c.conn.ReadFrom(b)
		if err != nil {
			if !c.isClosed() {
				c.logger.Printf("error reading from UDP connection: %v", err)
			}
			return
		}

		msg, err := dhcpv4.FromBytes(b[:n])
		if err != nil {
			// Not a valid DHCP packet; keep listening.
			continue
		}

		if msg.OpCode != dhcpv4.OpcodeBootReply {
			// Not a response message.
			continue
		}

		// This is a somewhat non-standard check, by the looks
		// of RFC 2131. It should work as long as the DHCP
		// server is spec-compliant for the HWAddr field.
		if c.ifaceHWAddr != nil && !bytes.Equal(c.ifaceHWAddr, msg.ClientHWAddr) {
			// Not for us.
			continue
		}

		c.pendingMu.Lock()
		p, ok := c.pending[msg.TransactionID]
		if ok {
			select {
			case <-p.done:
				close(p.ch)
				delete(c.pending, msg.TransactionID)

			// This send may block.
			case p.ch <- msg:
			}
		}
		c.pendingMu.Unlock()
	}
}

// ClientOpt is a function that configures the Client.
type ClientOpt func(c *Client) error

// WithTimeout configures the retransmission timeout.
//
// Default is 5 seconds.
func WithTimeout(d time.Duration) ClientOpt {
	return func(c *Client) (err error) {
		c.timeout = d
		return
	}
}

// WithSummaryLogger logs one-line DHCPv4 message summaries when sent & received.
func WithSummaryLogger() ClientOpt {
	return func(c *Client) (err error) {
		c.logger = ShortSummaryLogger{
			Printfer: log.New(os.Stderr, "[dhcpv4] ", log.LstdFlags),
		}
		return
	}
}

// WithDebugLogger logs multi-line full DHCPv4 messages when sent & received.
func WithDebugLogger() ClientOpt {
	return func(c *Client) (err error) {
		c.logger = DebugLogger{
			Printfer: log.New(os.Stderr, "[dhcpv4] ", log.LstdFlags),
		}
		return
	}
}

// WithLogger set the logger (see interface Logger).
func WithLogger(newLogger Logger) ClientOpt {
	return func(c *Client) (err error) {
		c.logger = newLogger
		return
	}
}

// WithUnicast forces client to send messages as unicast frames.
// By default client sends messages as broadcast frames even if server address is defined.
//
// srcAddr is both:
// * The source address of outgoing frames.
// * The address to be listened for incoming frames.
func WithUnicast(srcAddr *net.UDPAddr) ClientOpt {
	return func(c *Client) (err error) {
		if srcAddr == nil {
			srcAddr = &net.UDPAddr{Port: ServerPort}
		}
		c.conn, err = net.ListenUDP("udp4", srcAddr)
		if err != nil {
			err = fmt.Errorf("unable to start listening UDP port: %w", err)
		}
		return
	}
}

// WithHWAddr tells to the Client to receive messages destinated to selected
// hardware address
func WithHWAddr(hwAddr net.HardwareAddr) ClientOpt {
	return func(c *Client) (err error) {
		c.ifaceHWAddr = hwAddr
		return
	}
}

func withBufferCap(n int) ClientOpt {
	return func(c *Client) (err error) {
		c.bufferCap = n
		return
	}
}

// WithRetry configures the number of retransmissions to attempt.
//
// Default is 3.
func WithRetry(r int) ClientOpt {
	return func(c *Client) (err error) {
		c.retry = r
		return
	}
}

// WithServerAddr configures the address to send messages to.
func WithServerAddr(n *net.UDPAddr) ClientOpt {
	return func(c *Client) (err error) {
		c.serverAddr = n
		return
	}
}

// Matcher matches DHCP packets.
type Matcher func(*dhcpv4.DHCPv4) bool

// IsMessageType returns a matcher that checks for the message type.
//
// If t is MessageTypeNone, all packets are matched.
func IsMessageType(t dhcpv4.MessageType) Matcher {
	return func(p *dhcpv4.DHCPv4) bool {
		return p.MessageType() == t || t == dhcpv4.MessageTypeNone
	}
}

// DiscoverOffer sends a DHCPDiscover message and returns the first valid offer
// received.
func (c *Client) DiscoverOffer(ctx context.Context, modifiers ...dhcpv4.Modifier) (offer *dhcpv4.DHCPv4, err error) {
	// RFC 2131, Section 4.4.1, Table 5 details what a DISCOVER packet should
	// contain.
	discover, err := dhcpv4.NewDiscovery(c.ifaceHWAddr, dhcpv4.PrependModifiers(modifiers,
		dhcpv4.WithOption(dhcpv4.OptMaxMessageSize(MaxMessageSize)))...)
	if err != nil {
		err = fmt.Errorf("unable to create a discovery request: %w", err)
		return
	}

	offer, err = c.SendAndRead(ctx, c.serverAddr, discover, IsMessageType(dhcpv4.MessageTypeOffer))
	if err != nil {
		err = fmt.Errorf("got an error while the discovery request: %w", err)
		return
	}

	return
}

// Request completes the 4-way Discover-Offer-Request-Ack handshake.
//
// Note that modifiers will be applied *both* to Discover and Request packets.
func (c *Client) Request(ctx context.Context, modifiers ...dhcpv4.Modifier) (offer, ack *dhcpv4.DHCPv4, err error) {
	offer, err = c.DiscoverOffer(ctx, modifiers...)
	if err != nil {
		err = fmt.Errorf("unable to receive an offer: %w", err)
		return
	}

	// TODO(chrisko): should this be unicast to the server?
	request, err := dhcpv4.NewRequestFromOffer(offer, dhcpv4.PrependModifiers(modifiers,
		dhcpv4.WithOption(dhcpv4.OptMaxMessageSize(MaxMessageSize)))...)
	if err != nil {
		err = fmt.Errorf("unable to create a request: %w", err)
		return
	}

	ack, err = c.SendAndRead(ctx, c.serverAddr, request, nil)
	if err != nil {
		err = fmt.Errorf("got an error while processing the request: %w", err)
		return
	}

	return
}

// ErrTransactionIDInUse is returned if there were an attempt to send a message
// with the same TransactionID as we are already waiting an answer for.
type ErrTransactionIDInUse struct {
	// TransactionID is the transaction ID of the message which the error is related to.
	TransactionID dhcpv4.TransactionID
}

// Error is just the method to comply interface "error".
func (err *ErrTransactionIDInUse) Error() string {
	return fmt.Sprintf("transaction ID %s already in use", err.TransactionID)
}

// send sends p to destination and returns a response channel.
//
// Responses will be matched by transaction ID and ClientHWAddr.
//
// The returned lambda function must be called after all desired responses have
// been received in order to return the Transaction ID to the usable pool.
func (c *Client) send(dest *net.UDPAddr, msg *dhcpv4.DHCPv4) (resp <-chan *dhcpv4.DHCPv4, cancel func(), err error) {
	c.pendingMu.Lock()
	if _, ok := c.pending[msg.TransactionID]; ok {
		c.pendingMu.Unlock()
		return nil, nil, &ErrTransactionIDInUse{msg.TransactionID}
	}

	ch := make(chan *dhcpv4.DHCPv4, c.bufferCap)
	done := make(chan struct{})
	c.pending[msg.TransactionID] = &pendingCh{done: done, ch: ch}
	c.pendingMu.Unlock()

	cancel = func() {
		// Why can't we just close ch here?
		//
		// Because receiveLoop may potentially be blocked trying to
		// send on ch. We gotta unblock it first, and then we can take
		// the lock and remove the XID from the pending transaction
		// map.
		close(done)

		c.pendingMu.Lock()
		if p, ok := c.pending[msg.TransactionID]; ok {
			close(p.ch)
			delete(c.pending, msg.TransactionID)
		}
		c.pendingMu.Unlock()
	}

	if _, err := c.conn.WriteTo(msg.ToBytes(), dest); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("error writing packet to connection: %w", err)
	}
	return ch, cancel, nil
}

// This error should never be visible to users.
// It is used only to increase the timeout in retryFn.
var errDeadlineExceeded = errors.New("INTERNAL ERROR: deadline exceeded")

// SendAndRead sends a packet p to a destination dest and waits for the first
// response matching `match` as well as its Transaction ID and ClientHWAddr.
//
// If match is nil, the first packet matching the Transaction ID and
// ClientHWAddr is returned.
func (c *Client) SendAndRead(ctx context.Context, dest *net.UDPAddr, p *dhcpv4.DHCPv4, match Matcher) (*dhcpv4.DHCPv4, error) {
	var response *dhcpv4.DHCPv4
	err := c.retryFn(func(timeout time.Duration) error {
		ch, rem, err := c.send(dest, p)
		if err != nil {
			return err
		}
		c.logger.PrintMessage("sent message", p)
		defer rem()

		for {
			select {
			case <-c.done:
				return ErrNoResponse

			case <-time.After(timeout):
				return errDeadlineExceeded

			case <-ctx.Done():
				return ctx.Err()

			case packet := <-ch:
				if match == nil || match(packet) {
					c.logger.PrintMessage("received message", packet)
					response = packet
					return nil
				}
			}
		}
	})
	if err == errDeadlineExceeded {
		return nil, ErrNoResponse
	}
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) retryFn(fn func(timeout time.Duration) error) error {
	timeout := c.timeout

	// Each retry takes the amount of timeout at worst.
	for i := 0; i < c.retry || c.retry < 0; i++ { // TODO: why is this called "retry" if this is "tries" ("retries"+1)?
		switch err := fn(timeout); err {
		case nil:
			// Got it!
			return nil

		case errDeadlineExceeded:
			// Double timeout, then retry.
			timeout *= 2

		default:
			return err
		}
	}

	return errDeadlineExceeded
}
