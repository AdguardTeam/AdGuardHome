package dhcpsvc

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/c2h5oh/datasize"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// ICMPDialer establishes a connection to send ICMP echo requests to an address.
type ICMPDialer interface {
	// Dial establishes a connection to ip.  It returns the connection and the
	// remote address appropriate for the underlying connection type.  It's
	// caller's responsibility to close the returned connection.
	Dial(ctx context.Context, ip netip.Addr) (conn net.PacketConn, raddr net.Addr, err error)
}

// DefaultICMPDialerConfig is the configuration for [DefaultICMPDialer].
type DefaultICMPDialerConfig struct {
	// Logger is used for logging the dialer events.  It must not be nil.
	Logger *slog.Logger
}

// type check
var _ validate.Interface = (*DefaultICMPDialerConfig)(nil)

// Validate implements [validate.Interface] for *DefaultICMPDialerConfig.
func (conf *DefaultICMPDialerConfig) Validate() (err error) {
	if conf == nil {
		return errors.ErrNoValue
	}

	return validate.NotNil("conf.Logger", conf.Logger)
}

// DefaultICMPDialer is the [ICMPDialer] implementation based on
// [golang.org/x/net/icmp].
type DefaultICMPDialer struct {
	logger *slog.Logger
}

// type check
var _ ICMPDialer = (*DefaultICMPDialer)(nil)

// NewICMPDialer returns the production [ICMPDialer].  Use it in
// [ICMPAddressCheckerConfig] outside of tests.  conf must be valid.
func NewICMPDialer(conf *DefaultICMPDialerConfig) (d ICMPDialer) {
	return &DefaultICMPDialer{
		logger: conf.Logger,
	}
}

// Dial implements the [ICMPDialer] interface for *DefaultICMPDialer.
func (d *DefaultICMPDialer) Dial(
	ctx context.Context,
	ip netip.Addr,
) (conn net.PacketConn, addr net.Addr, err error) {
	if ip.Is4() {
		return dialICMP4(ctx, d.logger, ip)
	}

	return dialICMP6(ctx, d.logger, ip)
}

// dialICMP6 establishes an IPv6 connection to ip.  It prefers the unprivileged
// datagram socket and falls back to the raw one.  The raw socket requires root
// privileges, which is the same expectation the DHCP server itself has.
func dialICMP6(
	ctx context.Context,
	logger *slog.Logger,
	raddr netip.Addr,
) (conn net.PacketConn, addr net.Addr, err error) {
	pc, udpErr := icmp.ListenPacket("udp6", "::")
	if udpErr == nil {
		logger.DebugContext(ctx, "listening udp6")

		return pc, net.UDPAddrFromAddrPort(netip.AddrPortFrom(raddr, 0)), nil
	}

	pc, rawErr := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if rawErr != nil {
		return nil, nil, errors.Join(udpErr, rawErr)
	}

	return pc, &net.IPAddr{IP: net.IP(raddr.AsSlice()), Zone: raddr.Zone()}, nil
}

// ICMPAddressCheckerConfig is the configuration for [ICMPAddressChecker].
type ICMPAddressCheckerConfig struct {
	// Clock is used to get the current time.  It must not be nil.
	Clock timeutil.Clock

	// Dialer establishes echo connections.  It must not be nil.
	Dialer ICMPDialer

	// Logger is used for logging ICMP-related events.  It must not be nil.
	Logger *slog.Logger

	// Timeout is how long to wait for an echo reply.  It must be positive.
	Timeout time.Duration
}

// type check
var _ validate.Interface = (*ICMPAddressCheckerConfig)(nil)

// Validate implements [validate.Interface] for *ICMPAddressCheckerConfig.
func (c *ICMPAddressCheckerConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return errors.Join(
		validate.NotNilInterface("c.Clock", c.Clock),
		validate.NotNilInterface("c.Dialer", c.Dialer),
		validate.NotNil("c.Logger", c.Logger),
		validate.Positive("c.Timeout", c.Timeout),
	)
}

// ICMPAddressChecker is an [AddressChecker] that sends ICMP echo requests to
// check addresses for availability, as required by RFC 2131 Section 2.2.  An
// address is reported available if no echo reply for it is received within
// the configured timeout.
type ICMPAddressChecker struct {
	// dialer establishes connections.
	dialer ICMPDialer

	// clock is used to get current time.
	clock timeutil.Clock

	// logger is used for logging ICMP-related events.
	logger *slog.Logger

	// timeout constraints the echo request duration.
	timeout time.Duration
}

// NewICMPAddressChecker returns a new [*ICMPAddressChecker].  c must be
// valid.
func NewICMPAddressChecker(c *ICMPAddressCheckerConfig) (chk *ICMPAddressChecker) {
	return &ICMPAddressChecker{
		dialer:  c.Dialer,
		clock:   c.Clock,
		logger:  c.Logger,
		timeout: c.Timeout,
	}
}

// echoSequenceNumber is the sequence number of the ICMP echo requests sent by
// [ICMPAddressChecker].
const echoSequenceNumber = 1

// icmpReadBufSize is the size of the buffer for reading ICMP messages.  It is
// the common maximum transmission unit of Ethernet.
//
// TODO(e.burkov):  Consider making configurable.
const icmpReadBufSize = 1500 * datasize.B

// type check
var _ AddressChecker = (*ICMPAddressChecker)(nil)

// IsAvailable implements the [AddressChecker] interface for
// *ICMPAddressChecker.
func (c *ICMPAddressChecker) IsAvailable(ctx context.Context, ip netip.Addr) (ok bool, err error) {
	// TODO(e.burkov):  Consider adding dial timeout.
	conn, raddr, err := c.dialer.Dial(ctx, ip)
	if err != nil {
		return false, fmt.Errorf("dialing icmp: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, conn.Close()) }()

	err = conn.SetDeadline(c.clock.Now().Add(c.timeout))
	if err != nil {
		return false, fmt.Errorf("setting deadline: %w", err)
	}

	reqType, respType := icmpTypes(ip)

	// TODO(e.burkov):  Respect the context cancellation.

	err = sendEchoRequest(ctx, c.logger, conn, raddr, reqType)
	if err != nil {
		return false, err
	}

	replied, err := waitEchoResponse(ctx, c.logger, conn, ip, respType)
	if err != nil {
		return false, err
	}

	return !replied, nil
}

// sendEchoRequest marshals an ICMP echo request for raddr and writes it to the
// destination address of conn.
func sendEchoRequest(
	ctx context.Context,
	l *slog.Logger,
	conn net.PacketConn,
	raddr net.Addr,
	reqType icmp.Type,
) (err error) {
	id := newEchoID()

	msg := &icmp.Message{
		Type: reqType,
		Code: 0,
		Body: &icmp.Echo{
			ID:  id,
			Seq: echoSequenceNumber,
		},
	}

	// The pseudo-header argument is only used to calculate the checksum of an
	// ICMPv6 message, and the kernel does that for the requests itself.
	data, err := msg.Marshal(nil)
	if err != nil {
		return fmt.Errorf("marshaling echo request: %w", err)
	}

	_, err = conn.WriteTo(data, raddr)
	if err != nil {
		return fmt.Errorf("writing echo request: %w", err)
	}

	l.DebugContext(ctx, "sent icmp echo request", "peer", raddr, "id", id)

	return nil
}

// waitEchoResponse reads conn until either an echo reply from ip matching the
// request with the [echoSequenceNumber] sequence number is received or the
// deadline of conn is exceeded.  It reports whether such a reply has been
// received.
func waitEchoResponse(
	ctx context.Context,
	l *slog.Logger,
	conn net.PacketConn,
	ip netip.Addr,
	respType icmp.Type,
) (replied bool, err error) {
	proto := respType.Protocol()

	// TODO(e.burkov):  Consider using pools.
	buf := make([]byte, icmpReadBufSize.Bytes())

	for {
		n, peer, readErr := conn.ReadFrom(buf)
		if readErr != nil {
			if errors.Is(readErr, os.ErrDeadlineExceeded) {
				// No reply within the timeout means that the address is
				// available.
				return false, nil
			}

			return false, fmt.Errorf("reading echo reply: %w", readErr)
		}

		if echoReplyFrom(ctx, l, buf[:n], proto, respType, peer, ip) {
			return true, nil
		}
	}
}

// echoReplyFrom parses the ICMP message in data, received from peer, and
// reports whether it is an echo reply of type respType sent by ip, where proto
// is the ICMP protocol number.  It logs and ignores malformed messages.
func echoReplyFrom(
	ctx context.Context,
	l *slog.Logger,
	data []byte,
	proto int,
	respType icmp.Type,
	peer net.Addr,
	ip netip.Addr,
) (ok bool) {
	msg, err := icmp.ParseMessage(proto, data)
	if err != nil {
		l.DebugContext(ctx, "parsing icmp message", slogutil.KeyError, err)

		return false
	}

	return isMatchingEchoReply(msg, respType, peer, ip)
}

// icmpTypes returns the ICMP echo request and echo reply message types for the
// address family of ip.
func icmpTypes(ip netip.Addr) (reqType, replyType icmp.Type) {
	if ip.Is4() {
		return ipv4.ICMPTypeEcho, ipv4.ICMPTypeEchoReply
	}

	return ipv6.ICMPTypeEchoRequest, ipv6.ICMPTypeEchoReply
}

// isMatchingEchoReply reports whether msg is an echo reply of type replyType to
// the request with the [echoSequenceNumber] received from ip.  It doesn't check
// for a matching ID.  ip must have no zone.
func isMatchingEchoReply(
	msg *icmp.Message,
	replyType icmp.Type,
	peer net.Addr,
	ip netip.Addr,
) (ok bool) {
	if msg.Type != replyType {
		return false
	}

	echo, isEcho := msg.Body.(*icmp.Echo)
	if !isEcho || echo.Seq != echoSequenceNumber {
		return false
	}

	// The zone of ip is implied by the connection itself, so skip the
	// redundant comparison.
	ip = ip.WithZone("")

	switch a := peer.(type) {
	case *net.UDPAddr:
		return a.AddrPort().Addr().Unmap().WithZone("") == ip

	case *net.IPAddr:
		var peerIP netip.Addr
		peerIP, ok = netip.AddrFromSlice(a.IP)

		return ok && peerIP.Unmap().WithZone("") == ip

	default:
		return false
	}
}

// newEchoID returns a new random ICMP echo request ID.  The int type is used to
// match the type in [icmp.Echo.ID].
func newEchoID() (id int) {
	var buf [2]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		// Should never happen as [rand.Read] never returns an error.
		panic(fmt.Errorf("reading random bytes: %w", err))
	}

	return int(binary.BigEndian.Uint16(buf[:]))
}
