package util

import (
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

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
	ah.Init(f.Name())

	// Update from the hosts file
	ah.updateHosts()

	// Existing host
	ips := ah.Process("localhost")
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, net.ParseIP("127.0.0.1"), ips[0])

	// Unknown host
	ips = ah.Process("newhost")
	assert.Nil(t, ips)

	// Test hosts file
	table := ah.List()
	ips, _ = table["host"]
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, "127.0.0.1", ips[0].String())
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
	ips := ah.Process("newhost")
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
	ips = ah.Process("newhost")
	assert.NotNil(t, ips)
	assert.Equal(t, 1, len(ips))
	assert.Equal(t, "127.0.0.2", ips[0].String())
}
