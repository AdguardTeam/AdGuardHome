//go:build linux

package arpdb

import (
	"net"
	"net/netip"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const arpAOutputWrt = `
IP address    HW type     Flags       HW address            Mask     Device
1.2.3.4.5     0x1         0x2         aa:bb:cc:dd:ee:ff     *        wan
1.2.3.4       0x1         0x2         12:34:56:78:910       *        wan
192.168.1.2   0x1         0x2         ab:cd:ef:ab:cd:ef     *        wan
::ffff:ffff   0x1         0x2         ef:cd:ab:ef:cd:ab     *        wan`

const arpAOutput = `
invalid.mac (1.2.3.4) at 12:34:56:78:910 on el0 ifscope [ethernet]
invalid.ip  (1.2.3.4.5) at ab:cd:ef:ab:cd:12 on ek0 ifscope [ethernet]
invalid.fmt 1 at 12:cd:ef:ab:cd:ef on er0 ifscope [ethernet]
? (192.168.1.2) at ab:cd:ef:ab:cd:ef on en0 ifscope [ethernet]
? (::ffff:ffff) at ef:cd:ab:ef:cd:ab on em0 expires in 100 seconds [ethernet]`

const ipNeighOutput = `
1.2.3.4.5 dev enp0s3 lladdr aa:bb:cc:dd:ee:ff DELAY
1.2.3.4 dev enp0s3 lladdr 12:34:56:78:910 DELAY
192.168.1.2 dev enp0s3 lladdr ab:cd:ef:ab:cd:ef DELAY
::ffff:ffff dev enp0s3 lladdr ef:cd:ab:ef:cd:ab router STALE`

var wantNeighs = []Neighbor{{
	IP:  netip.MustParseAddr("192.168.1.2"),
	MAC: net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF},
}, {
	IP:  netip.MustParseAddr("::ffff:ffff"),
	MAC: net.HardwareAddr{0xEF, 0xCD, 0xAB, 0xEF, 0xCD, 0xAB},
}}

func TestFSysARPDB(t *testing.T) {
	require.NoError(t, fstest.TestFS(testdata, "proc_net_arp"))

	a := &fsysARPDB{
		ns: &neighs{
			mu: &sync.RWMutex{},
			ns: make([]Neighbor, 0),
		},
		fsys:     testdata,
		filename: "proc_net_arp",
	}

	err := a.Refresh()
	require.NoError(t, err)

	ns := a.Neighbors()
	assert.Equal(t, wantNeighs, ns)
}

func TestCmdARPDB_linux(t *testing.T) {
	sh := mapShell{
		"arp -a":   {err: nil, out: arpAOutputWrt, code: 0},
		"ip neigh": {err: nil, out: ipNeighOutput, code: 0},
	}
	substShell(t, sh.RunCmd)

	t.Run("wrt", func(t *testing.T) {
		a := &cmdARPDB{
			logger: slogutil.NewDiscardLogger(),
			parse:  parseArpAWrt,
			cmd:    "arp",
			args:   []string{"-a"},
			ns: &neighs{
				mu: &sync.RWMutex{},
				ns: make([]Neighbor, 0),
			},
		}

		err := a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, wantNeighs, a.Neighbors())
	})

	t.Run("ip_neigh", func(t *testing.T) {
		a := &cmdARPDB{
			logger: slogutil.NewDiscardLogger(),
			parse:  parseIPNeigh,
			cmd:    "ip",
			args:   []string{"neigh"},
			ns: &neighs{
				mu: &sync.RWMutex{},
				ns: make([]Neighbor, 0),
			},
		}
		err := a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, wantNeighs, a.Neighbors())
	})
}
