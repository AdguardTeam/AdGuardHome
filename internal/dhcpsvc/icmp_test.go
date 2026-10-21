package dhcpsvc_test

import (
	"cmp"
	"context"
	"net"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakenet"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// testEchoID is an arbitrary identifier for ICMP echo messages used in the
// tests.
const testEchoID = 0x1234

// ipToUDPAddr returns a [*net.UDPAddr] for ip, like a datagram ICMP socket would.
func ipToUDPAddr(tb testing.TB, ip netip.Addr) (addr *net.UDPAddr) {
	tb.Helper()

	return net.UDPAddrFromAddrPort(netip.AddrPortFrom(ip, 0))
}

// newEchoMsg returns the serialized ICMP echo message of the given type with the
// given identifier and sequence number.
func newEchoMsg(tb testing.TB, typ icmp.Type, id, seq int) (data []byte) {
	tb.Helper()

	m := &icmp.Message{
		Type: typ,
		Code: 0,
		Body: &icmp.Echo{ID: id, Seq: seq},
	}

	data, err := m.Marshal(nil)
	require.NoError(tb, err)

	return data
}

func TestICMPAddressChecker_IsAvailable(t *testing.T) {
	t.Parallel()

	echo4 := newEchoMsg(t, ipv4.ICMPTypeEchoReply, testEchoID, dhcpsvc.EchoSequenceNumber)
	echo6 := newEchoMsg(t, ipv6.ICMPTypeEchoReply, testEchoID, dhcpsvc.EchoSequenceNumber)

	zonelessLLAddr := netip.MustParseAddr("fe80::100")
	llAddr := zonelessLLAddr.WithZone("iface0")

	testCases := []struct {
		peer net.Addr
		ip   netip.Addr
		name string
		data []byte
	}{{
		name: "v4",
		ip:   testIPv4Dynamic,
		peer: ipToUDPAddr(t, testIPv4Dynamic),
		data: echo4,
	}, {
		name: "v4_from_ipaddr",
		ip:   testIPv4Dynamic,
		peer: &net.IPAddr{IP: testIPv4Dynamic.AsSlice()},
		data: echo4,
	}, {
		name: "v6",
		ip:   testIPv6Dynamic,
		peer: ipToUDPAddr(t, testIPv6Dynamic),
		data: echo6,
	}, {
		// The zone of ip is implied by the connection, and the kernel doesn't
		// fill it in the peer address read from a raw socket, so the reply is
		// still matching here.
		name: "v6_zone_ipaddr",
		ip:   llAddr,
		peer: &net.IPAddr{IP: zonelessLLAddr.AsSlice()},
		data: echo6,
	}, {
		// A datagram socket reports the peer's zone when it's known.
		name: "v6_zone_udp",
		ip:   llAddr,
		peer: ipToUDPAddr(t, llAddr),
		data: echo6,
	}}

	for _, tc := range testCases {
		pc := newTestPacketConn(t)
		pc.OnWriteTo = func(p []byte, _ net.Addr) (n int, err error) {
			return len(p), nil
		}

		responded := false
		pc.OnReadFrom = func(p []byte) (n int, addr net.Addr, err error) {
			if responded {
				panic(testutil.UnexpectedCall(p))
			}

			responded = true

			return copy(p, tc.data), tc.peer, nil
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			chk := newTestICMPAddressChecker(t, &dhcpsvc.ICMPAddressCheckerConfig{
				Dialer: newTestICMPDialer(t, pc, ipToUDPAddr(t, tc.ip), nil),
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			ok, err := chk.IsAvailable(ctx, tc.ip)
			require.False(t, ok)
			require.NoError(t, err)
		})
	}
}

func TestICMPAddressChecker_IsAvailable_unrelated(t *testing.T) {
	t.Parallel()

	const badSequenceNumber = dhcpsvc.EchoSequenceNumber + 1

	timeExceeded, err := (&icmp.Message{
		Type: ipv4.ICMPTypeTimeExceeded,
		Code: 0,
		Body: &icmp.TimeExceeded{},
	}).Marshal(nil)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		response []byte
	}{{
		name:     "too_short",
		response: []byte{0xff},
	}, {
		name:     "time_exceeded",
		response: timeExceeded,
	}, {
		name:     "bad_seq",
		response: newEchoMsg(t, ipv4.ICMPTypeEchoReply, testEchoID, badSequenceNumber),
	}}

	echo := newEchoMsg(t, ipv4.ICMPTypeEchoReply, testEchoID, dhcpsvc.EchoSequenceNumber)
	peer := ipToUDPAddr(t, testIPv4Dynamic)

	for _, tc := range testCases {
		readCh := make(chan []byte, 2)

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pc := newTestPacketConn(t)
			pc.OnWriteTo = func(p []byte, _ net.Addr) (n int, err error) {
				return len(p), nil
			}

			pc.OnReadFrom = func(p []byte) (n int, addr net.Addr, err error) {
				pt := testutil.NewPanicT(t)
				resp, ok := testutil.RequireReceive(pt, readCh, testTimeout)
				require.True(pt, ok)

				return copy(p, resp), peer, nil
			}

			chk := newTestICMPAddressChecker(t, &dhcpsvc.ICMPAddressCheckerConfig{
				Dialer: newTestICMPDialer(t, pc, peer, nil),
			})

			testutil.RequireSend(t, readCh, tc.response, testTimeout)
			testutil.RequireSend(t, readCh, echo, testTimeout)

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			ok, availErr := chk.IsAvailable(ctx, testIPv4Dynamic)
			require.False(t, ok)
			require.NoError(t, availErr)
		})
	}
}

func TestICMPAddressChecker_IsAvailable_otherPeer(t *testing.T) {
	t.Parallel()

	otherAddr4 := netip.MustParseAddr("192.168.1.3")

	testCases := []struct {
		peer net.Addr
		name string
	}{{
		name: "other_addr",
		peer: ipToUDPAddr(t, otherAddr4),
	}, {
		name: "unknown_type",
		peer: &net.UnixAddr{Name: t.TempDir(), Net: "unix"},
	}}

	echo := newEchoMsg(t, ipv4.ICMPTypeEchoReply, testEchoID, dhcpsvc.EchoSequenceNumber)

	for _, tc := range testCases {
		pc := newTestPacketConn(t)
		pc.OnWriteTo = func(p []byte, _ net.Addr) (n int, err error) {
			return len(p), nil
		}

		replied := false
		pc.OnReadFrom = func(p []byte) (n int, addr net.Addr, err error) {
			if replied {
				return 0, nil, os.ErrDeadlineExceeded
			}

			replied = true

			return copy(p, echo), tc.peer, nil
		}

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			chk := newTestICMPAddressChecker(t, &dhcpsvc.ICMPAddressCheckerConfig{
				Dialer: newTestICMPDialer(t, pc, ipToUDPAddr(t, testIPv4Dynamic), nil),
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			ok, err := chk.IsAvailable(ctx, testIPv4Dynamic)
			require.True(t, ok)
			require.NoError(t, err)
		})
	}
}

func TestICMPAddressChecker_IsAvailable_connError(t *testing.T) {
	t.Parallel()

	checkedAddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(testIPv4Dynamic, 0))

	basePC := newTestPacketConn(t)

	writePC := *basePC
	writePC.OnWriteTo = func(_ []byte, _ net.Addr) (n int, err error) { return 0, assert.AnError }

	readPC := *basePC
	readPC.OnWriteTo = func(p []byte, addr net.Addr) (n int, err error) { return len(p), nil }
	readPC.OnReadFrom = func(p []byte) (n int, addr net.Addr, err error) {
		return 0, nil, assert.AnError
	}

	deadlinePC := *basePC
	deadlinePC.OnWriteTo = func(p []byte, addr net.Addr) (n int, err error) { return len(p), nil }
	deadlinePC.OnReadFrom = func(p []byte) (n int, addr net.Addr, err error) {
		return 0, nil, os.ErrDeadlineExceeded
	}

	testCases := []struct {
		conn       net.PacketConn
		want       assert.BoolAssertionFunc
		name       string
		wantErrMsg string
	}{{
		conn:       &writePC,
		want:       assert.False,
		wantErrMsg: "writing echo request: assert.AnError general error for testing",
		name:       "write_error",
	}, {
		conn:       &readPC,
		want:       assert.False,
		wantErrMsg: "reading echo reply: assert.AnError general error for testing",
		name:       "read_error",
	}, {
		conn:       &deadlinePC,
		want:       assert.True,
		wantErrMsg: "",
		name:       "no_reply",
	}}

	for _, tc := range testCases {
		chk := newTestICMPAddressChecker(t, &dhcpsvc.ICMPAddressCheckerConfig{
			Dialer: newTestICMPDialer(t, tc.conn, checkedAddr, nil),
		})

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			ok, err := chk.IsAvailable(ctx, testIPv4Dynamic)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			tc.want(t, ok)
		})
	}
}

func TestICMPAddressChecker_IsAvailable_badDialer(t *testing.T) {
	t.Parallel()

	chk := newTestICMPAddressChecker(t, &dhcpsvc.ICMPAddressCheckerConfig{
		Dialer: newTestICMPDialer(t, nil, nil, assert.AnError),
	})

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	ok, err := chk.IsAvailable(ctx, testIPv4Dynamic)
	require.False(t, ok)

	assert.ErrorIs(t, err, assert.AnError)
}

func TestICMPAddressCheckerConfig_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		conf       *dhcpsvc.ICMPAddressCheckerConfig
		name       string
		wantErrMsg string
	}{{
		name:       "nil_conf",
		conf:       nil,
		wantErrMsg: "no value",
	}, {
		name: "valid",
		conf: &dhcpsvc.ICMPAddressCheckerConfig{
			Clock:   testClock,
			Dialer:  &testDialer{},
			Logger:  testLogger,
			Timeout: testTimeout,
		},
		wantErrMsg: "",
	}, {
		name: "zero_timeout",
		conf: &dhcpsvc.ICMPAddressCheckerConfig{
			Clock:   testClock,
			Dialer:  &testDialer{},
			Logger:  testLogger,
			Timeout: 0,
		},
		wantErrMsg: "c.Timeout: not positive: 0s",
	}, {
		name: "nil_dialer",
		conf: &dhcpsvc.ICMPAddressCheckerConfig{
			Clock:   testClock,
			Dialer:  nil,
			Logger:  testLogger,
			Timeout: testTimeout,
		},
		wantErrMsg: "c.Dialer: no value",
	}, {
		name: "nil_clock",
		conf: &dhcpsvc.ICMPAddressCheckerConfig{
			Clock:   nil,
			Dialer:  &testDialer{},
			Logger:  testLogger,
			Timeout: testTimeout,
		},
		wantErrMsg: "c.Clock: no value",
	}, {
		name: "nil_logger",
		conf: &dhcpsvc.ICMPAddressCheckerConfig{
			Clock:   testClock,
			Dialer:  &testDialer{},
			Logger:  nil,
			Timeout: testTimeout,
		},
		wantErrMsg: "c.Logger: no value",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}

func TestDefaultICMPDialerConfig_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		conf       *dhcpsvc.DefaultICMPDialerConfig
		name       string
		wantErrMsg string
	}{{
		name:       "nil_conf",
		conf:       nil,
		wantErrMsg: "no value",
	}, {
		name: "valid",
		conf: &dhcpsvc.DefaultICMPDialerConfig{
			Logger: testLogger,
		},
		wantErrMsg: "",
	}, {
		name: "nil_logger",
		conf: &dhcpsvc.DefaultICMPDialerConfig{
			Logger: nil,
		},
		wantErrMsg: "conf.Logger: no value",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}

// newTestPacketConn returns a new [fakenet.PacketConn] with all callbacks set
// to panic, except for [fakenet.PacketConn.OnClose] and
// [fakenet.PacketConn.OnSetDeadline], which are no-ops.
func newTestPacketConn(tb testing.TB) (pc *fakenet.PacketConn) {
	tb.Helper()

	return &fakenet.PacketConn{
		OnClose: func() (err error) {
			return nil
		},
		OnLocalAddr: func() (_ net.Addr) {
			panic(testutil.UnexpectedCall())
		},
		OnReadFrom: func(p []byte) (_ int, _ net.Addr, _ error) {
			panic(testutil.UnexpectedCall(p))
		},
		OnSetDeadline: func(_ time.Time) (err error) {
			return nil
		},
		OnSetReadDeadline: func(t time.Time) (_ error) {
			panic(testutil.UnexpectedCall(t))
		},
		OnSetWriteDeadline: func(t time.Time) (_ error) {
			panic(testutil.UnexpectedCall(t))
		},
		OnWriteTo: func(p []byte, addr net.Addr) (_ int, _ error) {
			panic(testutil.UnexpectedCall(p, addr))
		},
	}
}

// testDialer is a mock [dhcpsvc.ICMPDialer] implementation for tests.
type testDialer struct {
	onDial func(ctx context.Context, ip netip.Addr) (conn net.PacketConn, raddr net.Addr, err error)
}

// type check
var _ dhcpsvc.ICMPDialer = (*testDialer)(nil)

// Dial implements the [dhcpsvc.ICMPDialer] interface for *testDialer.
func (d *testDialer) Dial(
	ctx context.Context,
	ip netip.Addr,
) (conn net.PacketConn, raddr net.Addr, err error) {
	return d.onDial(ctx, ip)
}

// newTestICMPDialer returns a new [testDialer] that either always returns the
// specified error or successfully returns a new test packet connection.
func newTestICMPDialer(
	tb testing.TB,
	conn net.PacketConn,
	ip net.Addr,
	dialErr error,
) (d *testDialer) {
	tb.Helper()

	return &testDialer{
		onDial: func(_ context.Context, _ netip.Addr) (pc net.PacketConn, a net.Addr, err error) {
			return conn, ip, dialErr
		},
	}
}

// newTestICMPAddressChecker returns a new [dhcpsvc.ICMPAddressChecker] using
// the provided configuration, falling back to default test values for any unset
// fields.
func newTestICMPAddressChecker(
	tb testing.TB,
	conf *dhcpsvc.ICMPAddressCheckerConfig,
) (c *dhcpsvc.ICMPAddressChecker) {
	tb.Helper()

	clock := cmp.Or[timeutil.Clock](conf.Clock, testClock)
	dialer := cmp.Or[dhcpsvc.ICMPDialer](conf.Dialer, &testDialer{
		onDial: func(ctx context.Context, addr netip.Addr) (_ net.PacketConn, _ net.Addr, _ error) {
			panic(testutil.UnexpectedCall(ctx, addr))
		},
	})
	logger := cmp.Or(conf.Logger, testLogger)
	timeout := cmp.Or(conf.Timeout, testTimeout)

	return dhcpsvc.NewICMPAddressChecker(&dhcpsvc.ICMPAddressCheckerConfig{
		Clock:   clock,
		Dialer:  dialer,
		Logger:  logger,
		Timeout: timeout,
	})
}
