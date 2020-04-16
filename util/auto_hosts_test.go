package util

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func prepareTestDir() string {
	const dir = "./agh-test"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func TestAutoHostsResolution(t *testing.T) {
	ah := AutoHosts{}

	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()

	f, _ := ioutil.TempFile(dir, "")
	defer func() { _ = os.Remove(f.Name()) }()
	defer f.Close()

	_, _ = f.WriteString("  127.0.0.1   host  localhost  \n")
	_, _ = f.WriteString("  ::1   localhost  \n")

	ah.Init(f.Name())

	// Update from the hosts file
	ah.updateHosts()

	// Existing host
	ips := ah.Process("localhost", dns.TypeA)
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, net.ParseIP("127.0.0.1"), ips[0])

	// Unknown host
	ips = ah.Process("newhost", dns.TypeA)
	assert.Nil(t, ips)

	// Test hosts file
	table := ah.List()
	ips, _ = table["host"]
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, "127.0.0.1", ips[0].String())

	// Test PTR
	a, _ := dns.ReverseAddr("127.0.0.1")
	a = strings.TrimSuffix(a, ".")
	assert.True(t, ah.ProcessReverse(a, dns.TypePTR) == "host")
	a, _ = dns.ReverseAddr("::1")
	a = strings.TrimSuffix(a, ".")
	assert.True(t, ah.ProcessReverse(a, dns.TypePTR) == "localhost")
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
	ah.updateHosts()

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
	assert.True(t, dnsUnreverseAddr("1.0.0.127.in-addr.arpa").Equal(net.ParseIP("127.0.0.1").To4()))
	assert.True(t, dnsUnreverseAddr("4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa").Equal(net.ParseIP("::abcd:1234")))

	assert.True(t, dnsUnreverseAddr("1.0.0.127.in-addr.arpa.") == nil)
	assert.True(t, dnsUnreverseAddr(".0.0.127.in-addr.arpa") == nil)
	assert.True(t, dnsUnreverseAddr(".3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa") == nil)
	assert.True(t, dnsUnreverseAddr("4.3.2.1.d.c.b.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0..ip6.arpa") == nil)
}
