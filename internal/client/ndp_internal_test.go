package client

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakeos/fakeexec"
	"github.com/AdguardTeam/golibs/testutil/faketime"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ndpTestTimeout is a timeout for tests of the IPv6 neighbor table reader.
const ndpTestTimeout = 1 * time.Second

// Neighbors used throughout the tests of the IPv6 neighbor table reader.
var (
	ndpTestIP        = netip.MustParseAddr("2001:db8::1")
	ndpTestUnknownIP = netip.MustParseAddr("2001:db8::dead")
	ndpTestMAC       = errors.Must(net.ParseMAC("aa:bb:cc:dd:ee:ff"))
	ndpTestOthMAC    = errors.Must(net.ParseMAC("11:22:33:44:55:66"))
)

// ndpTestOutput is the output of the neighbor table command containing
// [ndpTestIP] mapped to [ndpTestMAC].
const ndpTestOutput = "2001:db8::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n"

func TestParseNDPNeighbors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		want map[netip.Addr]net.HardwareAddr
		name string
		data string
	}{{
		want: map[netip.Addr]net.HardwareAddr{
			ndpTestIP:                          ndpTestMAC,
			netip.MustParseAddr("2001:db8::2"): ndpTestOthMAC,
		},
		name: "global",
		data: ndpTestOutput +
			"2001:db8::2 dev eth0 lladdr 11:22:33:44:55:66 STALE\n",
	}, {
		want: map[netip.Addr]net.HardwareAddr{
			netip.MustParseAddr("fe80::1").WithZone("eth0"): ndpTestMAC,
			netip.MustParseAddr("fe80::1").WithZone("eth1"): ndpTestOthMAC,
		},
		name: "link_local_zones",
		data: "fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff router REACHABLE\n" +
			"fe80::1 dev eth1 lladdr 11:22:33:44:55:66 REACHABLE\n",
	}, {
		want: map[netip.Addr]net.HardwareAddr{},
		name: "no_lladdr",
		data: "2001:db8::1 dev eth0 FAILED\n",
	}, {
		want: map[netip.Addr]net.HardwareAddr{},
		name: "short_line",
		data: "2001:db8::1 dev eth0\n",
	}, {
		want: map[netip.Addr]net.HardwareAddr{},
		name: "bad_ip",
		data: "not-an-ip dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n",
	}, {
		want: map[netip.Addr]net.HardwareAddr{},
		name: "bad_mac",
		data: "2001:db8::1 dev eth0 lladdr not-a-mac REACHABLE\n",
	}, {
		want: map[netip.Addr]net.HardwareAddr{},
		name: "empty",
		data: "",
	}, {
		want: map[netip.Addr]net.HardwareAddr{
			ndpTestIP: ndpTestMAC,
		},
		name: "mixed",
		data: "bad line\n" +
			"2001:db8::2 dev eth0 INCOMPLETE\n" +
			ndpTestOutput,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, parseNDPNeighbors([]byte(tc.data)))
		})
	}
}

func TestNDPNeighbors_refresh(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		n, cmd, _ := newTestNDPNeighbors(t, func(_ int64) (out string, err error) {
			return ndpTestOutput, nil
		})

		n.refresh(testutil.ContextWithTimeout(t, ndpTestTimeout))

		assert.Equal(t, int64(1), cmd.runs.Load())
		assert.Equal(t, ndpTestMAC, n.macFor(ndpTestIP))
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		n, cmd, _ := newTestNDPNeighbors(t, func(_ int64) (out string, err error) {
			return "", errors.Error("no such command")
		})

		n.refresh(testutil.ContextWithTimeout(t, ndpTestTimeout))

		assert.Equal(t, int64(1), cmd.runs.Load())
		assert.Nil(t, n.macFor(ndpTestIP))
	})
}

func TestNDPNeighbors_macFor(t *testing.T) {
	t.Parallel()

	t.Run("link_local_zone", func(t *testing.T) {
		t.Parallel()

		n, _, _ := newTestNDPNeighbors(t, func(_ int64) (out string, err error) {
			return "fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n", nil
		})

		n.refresh(testutil.ContextWithTimeout(t, ndpTestTimeout))

		linkLocal := netip.MustParseAddr("fe80::1")
		assert.Equal(t, ndpTestMAC, n.macFor(linkLocal.WithZone("eth0")))

		// The same address on another link must not match.
		assert.Nil(t, n.macFor(linkLocal.WithZone("eth1")))
	})

	t.Run("ipv4_ignored", func(t *testing.T) {
		t.Parallel()

		n, cmd, _ := newTestNDPNeighbors(t, func(_ int64) (out string, err error) {
			return ndpTestOutput, nil
		})

		assert.Nil(t, n.macFor(netip.MustParseAddr("192.0.2.1")))
		assert.Nil(t, n.macFor(netip.MustParseAddr("::ffff:192.0.2.1")))

		assert.Equal(t, int64(0), cmd.runs.Load())
	})

	// Make sure that a command that keeps failing is not run on each request,
	// see https://github.com/AdguardTeam/AdGuardHome/pull/8248.
	t.Run("failure_is_rate_limited", func(t *testing.T) {
		t.Parallel()

		n, cmd, advance := newTestNDPNeighbors(t, func(_ int64) (out string, err error) {
			return "", errors.Error("no such command")
		})

		assert.Nil(t, n.macFor(ndpTestIP))
		waitNDPRead(t, n, cmd, 1)

		// Further requests within the TTL must not read the table again, even
		// though the previous read has failed.
		for range 3 {
			assert.Nil(t, n.macFor(ndpTestIP))
		}

		assert.Equal(t, int64(1), cmd.runs.Load())

		advance(ndpUpdateInterval)

		assert.Nil(t, n.macFor(ndpTestIP))
		waitNDPRead(t, n, cmd, 2)
	})

	t.Run("failed_read_keeps_data", func(t *testing.T) {
		t.Parallel()

		n, cmd, advance := newTestNDPNeighbors(t, func(num int64) (out string, err error) {
			if num == 1 {
				return ndpTestOutput, nil
			}

			return "", errors.Error("no such command")
		})

		n.refresh(testutil.ContextWithTimeout(t, ndpTestTimeout))
		require.Equal(t, ndpTestMAC, n.macFor(ndpTestIP))

		advance(ndpUpdateInterval)

		// A request for an unknown address schedules a read, which fails.  It
		// must not discard the data that is still within the TTL.
		assert.Nil(t, n.macFor(ndpTestUnknownIP))
		waitNDPRead(t, n, cmd, 2)

		assert.Equal(t, ndpTestMAC, n.macFor(ndpTestIP))

		// Once no successful read has confirmed the data within the TTL, it
		// isn't used at all.
		advance(ndpDataTTL)

		assert.Nil(t, n.macFor(ndpTestIP))
	})

	t.Run("expired_data_is_replaced", func(t *testing.T) {
		t.Parallel()

		n, cmd, advance := newTestNDPNeighbors(t, func(num int64) (out string, err error) {
			if num == 1 {
				return ndpTestOutput, nil
			}

			return "2001:db8::1 dev eth0 lladdr 11:22:33:44:55:66 REACHABLE\n", nil
		})

		n.refresh(testutil.ContextWithTimeout(t, ndpTestTimeout))
		require.Equal(t, ndpTestMAC, n.macFor(ndpTestIP))

		advance(ndpDataTTL)

		// The expired MAC address must not be returned, since the address may
		// have been reassigned to another device.  The request that finds it
		// expired schedules a read, so the requests that follow get the current
		// one.
		assert.Nil(t, n.macFor(ndpTestIP))
		waitNDPRead(t, n, cmd, 2)

		assert.Equal(t, ndpTestOthMAC, n.macFor(ndpTestIP))
	})

	t.Run("single_read_in_flight", func(t *testing.T) {
		t.Parallel()

		started, release := make(chan struct{}, 1), make(chan struct{})
		n, cmd, _ := newTestNDPNeighbors(t, func(_ int64) (out string, err error) {
			started <- struct{}{}
			<-release

			return ndpTestOutput, nil
		})

		assert.Nil(t, n.macFor(ndpTestIP))
		testutil.RequireReceive(t, started, ndpTestTimeout)

		// The requests arriving while the table is being read must neither wait
		// for it nor start another read.
		for range 5 {
			assert.Nil(t, n.macFor(ndpTestIP))
		}

		assert.Equal(t, int64(1), cmd.runs.Load())

		close(release)
		waitNDPRead(t, n, cmd, 1)

		assert.Equal(t, ndpTestMAC, n.macFor(ndpTestIP))
	})
}

// waitNDPRead waits until cmd has been run wantRuns times and n has finished
// reading the neighbor table.
func waitNDPRead(tb testing.TB, n *ndpNeighbors, cmd *testNDPCmd, wantRuns int64) {
	tb.Helper()

	require.EventuallyWithT(tb, func(ct *assert.CollectT) {
		n.mu.RLock()
		defer n.mu.RUnlock()

		assert.Equal(ct, wantRuns, cmd.runs.Load())
		assert.False(ct, n.isReading)
	}, ndpTestTimeout, ndpTestTimeout/100)
}

// newTestNDPNeighbors returns a neighbor table reader that runs onRun instead of
// the actual command, along with the fake command and a function advancing the
// clock of the reader.  onRun receives the number of the run, starting with one.
func newTestNDPNeighbors(
	tb testing.TB,
	onRun func(num int64) (out string, err error),
) (n *ndpNeighbors, cmd *testNDPCmd, advance func(d time.Duration)) {
	tb.Helper()

	cmd = &testNDPCmd{onRun: onRun}
	clock, advance := newTestClock()

	return &ndpNeighbors{
		logger:    slogutil.NewDiscardLogger(),
		clock:     clock,
		cmdCons:   cmd.constructor(),
		mu:        &sync.RWMutex{},
		neighbors: map[netip.Addr]net.HardwareAddr{},
	}, cmd, advance
}

// newTestClock returns a clock reporting the time that advance moves forward.
// Both the clock and advance are safe for concurrent use.
func newTestClock() (clock timeutil.Clock, advance func(d time.Duration)) {
	mu := &sync.Mutex{}
	now := time.Unix(0, 0)

	return &faketime.Clock{
			OnNow: func() (t time.Time) {
				mu.Lock()
				defer mu.Unlock()

				return now
			},
		}, func(d time.Duration) {
			mu.Lock()
			defer mu.Unlock()

			now = now.Add(d)
		}
}

// testNDPCmd is a fake command printing the IPv6 neighbor table.
type testNDPCmd struct {
	// onRun returns the output and the error of a single run of the command.
	onRun func(num int64) (out string, err error)

	// runs is the number of times the command has been run.
	runs atomic.Int64
}

// constructor returns a command constructor that runs c.
func (c *testNDPCmd) constructor() (cons executil.CommandConstructor) {
	return &fakeexec.CommandConstructor{
		OnNew: func(
			_ context.Context,
			conf *executil.CommandConfig,
		) (execCmd executil.Command, err error) {
			out, runErr := c.onRun(c.runs.Add(1))

			execCmd = &fakeexec.Command{
				OnStart: func(_ context.Context) (startErr error) {
					_, startErr = conf.Stdout.Write([]byte(out))

					return startErr
				},
				OnWait: func(_ context.Context) (waitErr error) {
					return runErr
				},
			}

			return execCmd, nil
		},
	}
}
