package dhcpsvc_test

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Addresses to check in the tests.
var (
	testCheckerAddr4 = netip.MustParseAddr("192.168.1.2")
	testCheckerAddr6 = netip.MustParseAddr("2001:db8::2")
)

// testDialer is a mock [dhcpsvc.ICMPDialer] implementation.
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

func TestICMPAddressChecker_IsAvailable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		dialer     dhcpsvc.ICMPDialer
		want       assert.BoolAssertionFunc
		ip         netip.Addr
		name       string
		wantErrMsg string
	}{{
		ip:         testCheckerAddr4,
		name:       "no_reply_v4",
		wantErrMsg: "",
		want:       assert.True,
	}, {
		ip:         testCheckerAddr4,
		name:       "reply_v4",
		wantErrMsg: "",
		want:       assert.False,
	}, {
		ip:         testCheckerAddr4,
		name:       "reply_from_ipaddr_v4",
		wantErrMsg: "",
		want:       assert.False,
	}, {
		ip:         testCheckerAddr4,
		name:       "other_peer_v4",
		wantErrMsg: "",
		want:       assert.True,
	}, {
		ip:         testCheckerAddr4,
		name:       "unknown_peer_v4",
		wantErrMsg: "",
		want:       assert.True,
	}, {
		ip:         testCheckerAddr4,
		name:       "unrelated_v4",
		wantErrMsg: "",
		want:       assert.False,
	}, {
		ip:         testCheckerAddr6,
		name:       "reply_v6",
		wantErrMsg: "",
		want:       assert.False,
	}, {
		ip:         testCheckerAddr4,
		name:       "write_error",
		wantErrMsg: "writing echo request: write error",
		want:       assert.False,
	}, {
		ip:         testCheckerAddr4,
		name:       "read_error",
		wantErrMsg: "reading echo reply: read error",
		want:       assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			chk := dhcpsvc.NewICMPAddressChecker(&dhcpsvc.ICMPAddressCheckerConfig{
				Clock:   testClock,
				Dialer:  tc.dialer,
				Logger:  testLogger,
				Timeout: testTimeout,
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			ok, err := chk.IsAvailable(ctx, tc.ip)
			tc.want(t, ok)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestICMPAddressChecker_IsAvailable_badDialer(t *testing.T) {
	t.Parallel()

	d := &testDialer{
		onDial: func(
			_ context.Context,
			_ netip.Addr,
		) (conn net.PacketConn, raddr net.Addr, err error) {
			return nil, nil, assert.AnError
		},
	}

	chk := dhcpsvc.NewICMPAddressChecker(&dhcpsvc.ICMPAddressCheckerConfig{
		Clock:   testClock,
		Dialer:  d,
		Logger:  testLogger,
		Timeout: testTimeout,
	})

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	ok, err := chk.IsAvailable(ctx, testCheckerAddr4)
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
