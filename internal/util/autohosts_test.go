package util

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

func prepareTestFile(t *testing.T) (f *os.File) {
	t.Helper()

	dir := t.TempDir()

	f, err := ioutil.TempFile(dir, "")
	require.Nil(t, err)
	require.NotNil(t, f)
	t.Cleanup(func() {
		assert.Nil(t, f.Close())
	})

	return f
}

func assertWriting(t *testing.T, f *os.File, strs ...string) {
	t.Helper()

	for _, str := range strs {
		n, err := f.WriteString(str)
		require.Nil(t, err)
		assert.Equal(t, n, len(str))
	}
}

func TestAutoHostsResolution(t *testing.T) {
	ah := &AutoHosts{}

	f := prepareTestFile(t)

	assertWriting(t, f,
		"  127.0.0.1   host  localhost # comment \n",
		"  ::1   localhost#comment  \n",
	)
	ah.Init(f.Name())

	t.Run("existing_host", func(t *testing.T) {
		ips := ah.Process("localhost", dns.TypeA)
		require.Len(t, ips, 1)
		assert.Equal(t, net.IPv4(127, 0, 0, 1), ips[0])
	})

	t.Run("unknown_host", func(t *testing.T) {
		ips := ah.Process("newhost", dns.TypeA)
		assert.Nil(t, ips)

		// Comment.
		ips = ah.Process("comment", dns.TypeA)
		assert.Nil(t, ips)
	})

	t.Run("hosts_file", func(t *testing.T) {
		names, ok := ah.List()["127.0.0.1"]
		require.True(t, ok)
		assert.Equal(t, []string{"host", "localhost"}, names)
	})

	t.Run("ptr", func(t *testing.T) {
		testCases := []struct {
			wantIP   string
			wantLen  int
			wantHost string
		}{
			{wantIP: "127.0.0.1", wantLen: 2, wantHost: "host"},
			{wantIP: "::1", wantLen: 1, wantHost: "localhost"},
		}

		for _, tc := range testCases {
			a, err := dns.ReverseAddr(tc.wantIP)
			require.Nil(t, err)

			a = strings.TrimSuffix(a, ".")
			hosts := ah.ProcessReverse(a, dns.TypePTR)
			require.Len(t, hosts, tc.wantLen)
			assert.Equal(t, tc.wantHost, hosts[0])
		}
	})
}

func TestAutoHostsFSNotify(t *testing.T) {
	ah := &AutoHosts{}

	f := prepareTestFile(t)

	assertWriting(t, f, "  127.0.0.1   host  localhost  \n")
	ah.Init(f.Name())

	t.Run("unknown_host", func(t *testing.T) {
		ips := ah.Process("newhost", dns.TypeA)
		assert.Nil(t, ips)
	})

	// Start monitoring for changes.
	ah.Start()
	t.Cleanup(ah.Close)

	assertWriting(t, f, "127.0.0.2   newhost\n")
	require.Nil(t, f.Sync())

	// Wait until fsnotify has triggerred and processed the
	// file-modification event.
	time.Sleep(50 * time.Millisecond)

	t.Run("notified", func(t *testing.T) {
		ips := ah.Process("newhost", dns.TypeA)
		assert.NotNil(t, ips)
		require.Len(t, ips, 1)
		assert.True(t, net.IP{127, 0, 0, 2}.Equal(ips[0]))
	})
}
