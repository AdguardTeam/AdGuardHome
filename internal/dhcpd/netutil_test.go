// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasStaticIPDhcpcdConf(t *testing.T) {
	dhcpdConf := `#comment
# comment

interface eth0
static ip_address=192.168.0.1/24

# interface wlan0
static ip_address=192.168.1.1/24

# comment
`
	assert.True(t, !hasStaticIPDhcpcdConf(dhcpdConf, "wlan0"))

	dhcpdConf = `#comment
# comment

interface eth0
static ip_address=192.168.0.1/24

# interface wlan0
static ip_address=192.168.1.1/24

# comment

interface wlan0
# comment
static ip_address=192.168.2.1/24
`
	assert.True(t, hasStaticIPDhcpcdConf(dhcpdConf, "wlan0"))
}

func TestSetStaticIPDhcpcdConf(t *testing.T) {
	dhcpcdConf := `
interface wlan0
static ip_address=192.168.0.2/24
static routers=192.168.0.1
static domain_name_servers=192.168.0.2

`
	s := updateStaticIPDhcpcdConf("wlan0", "192.168.0.2/24", "192.168.0.1", "192.168.0.2")
	assert.Equal(t, dhcpcdConf, s)

	// without gateway
	dhcpcdConf = `
interface wlan0
static ip_address=192.168.0.2/24
static domain_name_servers=192.168.0.2

`
	s = updateStaticIPDhcpcdConf("wlan0", "192.168.0.2/24", "", "192.168.0.2")
	assert.Equal(t, dhcpcdConf, s)
}
