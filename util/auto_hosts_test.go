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

func TestAutoHosts(t *testing.T) {
	ah := AutoHosts{}

	dir := prepareTestDir()
	defer func() { _ = os.RemoveAll(dir) }()

	f, _ := ioutil.TempFile(dir, "")
	defer os.Remove(f.Name())
	defer f.Close()

	_, _ = f.WriteString("  127.0.0.1   host  localhost  \n")

	ah.Init(f.Name())
	ah.Start()
	// wait until we parse the file
	time.Sleep(50 * time.Millisecond)

	ips := ah.Process("localhost")
	assert.True(t, ips[0].Equal(net.ParseIP("127.0.0.1")))
	ips = ah.Process("newhost")
	assert.True(t, ips == nil)

	table := ah.List()
	ips, _ = table["host"]
	assert.True(t, ips[0].String() == "127.0.0.1")

	_, _ = f.WriteString("127.0.0.2   newhost\n")
	// wait until fsnotify has triggerred and processed the file-modification event
	time.Sleep(50 * time.Millisecond)

	ips = ah.Process("newhost")
	assert.True(t, ips[0].Equal(net.ParseIP("127.0.0.2")))

	ah.Close()
}
