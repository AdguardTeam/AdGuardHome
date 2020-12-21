package util

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func TestAutoHostsResolution(t *testing.T) {
	ah := AutoHosts{}

	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()

	f, _ := ioutil.TempFile(dir, "")
	defer func() { _ = os.Remove(f.Name()) }()
	defer f.Close()

	_, _ = f.WriteString("  127.0.0.1   host  localhost # comment \n")
	_, _ = f.WriteString("  ::1   localhost#comment  \n")

	ah.Init(f.Name())

	// Existing host
	ips := ah.Process("localhost", dns.TypeA)
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, net.ParseIP("127.0.0.1"), ips[0])

	// Unknown host
	ips = ah.Process("newhost", dns.TypeA)
	assert.Nil(t, ips)

	// Unknown host (comment)
	ips = ah.Process("comment", dns.TypeA)
	assert.Nil(t, ips)

	// Test hosts file
	table := ah.List()
	names, ok := table["127.0.0.1"]
	assert.True(t, ok)
	assert.Equal(t, []string{"host", "localhost"}, names)

	// Test PTR
	a, _ := dns.ReverseAddr("127.0.0.1")
	a = strings.TrimSuffix(a, ".")
	hosts := ah.ProcessReverse(a, dns.TypePTR)
	if assert.Len(t, hosts, 2) {
		assert.Equal(t, hosts[0], "host")
	}

	a, _ = dns.ReverseAddr("::1")
	a = strings.TrimSuffix(a, ".")
	hosts = ah.ProcessReverse(a, dns.TypePTR)
	if assert.Len(t, hosts, 1) {
		assert.Equal(t, hosts[0], "localhost")
	}
}

func TestAutoHostsFSNotify(t *testing.T) {
	ah := AutoHosts{}

	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()

	f, _ := ioutil.TempFile(dir, "")
	defer func() { _ = os.Remove(f.Name()) }()
	defer f.Close()

	// Init
	_, _ = f.WriteString("  127.0.0.1   host  localhost  \n")
	ah.Init(f.Name())

	// Unknown host
	ips := ah.Process("newhost", dns.TypeA)
	assert.Nil(t, ips)

	// Stat monitoring for changes
	ah.Start()
	defer ah.Close()

	// Update file
	_, _ = f.WriteString("127.0.0.2   newhost\n")
	_ = f.Sync()

	// wait until fsnotify has triggerred and processed the file-modification event
	time.Sleep(50 * time.Millisecond)

	// Check if we are notified about changes
	ips = ah.Process("newhost", dns.TypeA)
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, "127.0.0.2", ips[0].String())
}

func TestIP(t *testing.T) {
	assert.Equal(t, "127.0.0.1", DNSUnreverseAddr("1.0.0.127.in-addr.arpa").String())
	assert.Equal(t, "::abcd:1234", DNSUnreverseAddr("4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa").String())
	assert.Equal(t, "::abcd:1234", DNSUnreverseAddr("4.3.2.1.d.c.B.A.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa").String())

	assert.Nil(t, DNSUnreverseAddr("1.0.0.127.in-addr.arpa."))
	assert.Nil(t, DNSUnreverseAddr(".0.0.127.in-addr.arpa"))
	assert.Nil(t, DNSUnreverseAddr(".3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa"))
	assert.Nil(t, DNSUnreverseAddr("4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0..ip6.arpa"))
	assert.Nil(t, DNSUnreverseAddr("4.3.2.1.d.c.b. .0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa"))
}
