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

	dir := aghtest.PrepareTestDir(t)

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

func TestDNSReverseAddr(t *testing.T) {
	testCases := []struct {
		name string
		have string
		want net.IP
	}{{
		name: "good_ipv4",
		have: "1.0.0.127.in-addr.arpa",
		want: net.IP{127, 0, 0, 1},
	}, {
		name: "good_ipv6",
		have: "4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		want: net.ParseIP("::abcd:1234"),
	}, {
		name: "good_ipv6_case",
		have: "4.3.2.1.d.c.B.A.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
		want: net.ParseIP("::abcd:1234"),
	}, {
		name: "bad_ipv4_dot",
		have: "1.0.0.127.in-addr.arpa.",
	}, {
		name: "wrong_ipv4",
		have: ".0.0.127.in-addr.arpa",
	}, {
		name: "wrong_ipv6",
		have: ".3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
	}, {
		name: "bad_ipv6_dot",
		have: "4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0..ip6.arpa",
	}, {
		name: "bad_ipv6_space",
		have: "4.3.2.1.d.c.b. .0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ip := DNSUnreverseAddr(tc.have)
			assert.True(t, tc.want.Equal(ip))
		})
	}
}
