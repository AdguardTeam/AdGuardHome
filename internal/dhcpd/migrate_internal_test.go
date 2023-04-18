package dhcpd

import (
	"encoding/json"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testData = `[
{"mac":"ESIzRFVm","ip":"AQIDBA==","host":"test1","exp":1},
{"mac":"ZlVEMyIR","ip":"BAMCAQ==","host":"test2","exp":1231231231}
]`

func TestMigrateDB(t *testing.T) {
	dir := t.TempDir()

	oldLeasesPath := filepath.Join(dir, dbFilename)
	dataDirPath := filepath.Join(dir, dataFilename)

	err := os.WriteFile(oldLeasesPath, []byte(testData), 0o644)
	require.NoError(t, err)

	wantLeases := []*Lease{{
		Expiry:   time.Time{},
		Hostname: "test1",
		HWAddr:   net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		IP:       netip.MustParseAddr("1.2.3.4"),
		IsStatic: true,
	}, {
		Expiry:   time.Unix(1231231231, 0),
		Hostname: "test2",
		HWAddr:   net.HardwareAddr{0x66, 0x55, 0x44, 0x33, 0x22, 0x11},
		IP:       netip.MustParseAddr("4.3.2.1"),
		IsStatic: false,
	}}

	conf := &ServerConfig{
		WorkDir: dir,
		DataDir: dir,
	}

	err = migrateDB(conf)
	require.NoError(t, err)

	_, err = os.Stat(oldLeasesPath)
	require.ErrorIs(t, err, os.ErrNotExist)

	var data []byte
	data, err = os.ReadFile(dataDirPath)
	require.NoError(t, err)

	dl := &dataLeases{}
	err = json.Unmarshal(data, dl)
	require.NoError(t, err)

	leases := dl.Leases

	for i, wl := range wantLeases {
		assert.Equal(t, wl.Hostname, leases[i].Hostname)
		assert.Equal(t, wl.HWAddr, leases[i].HWAddr)
		assert.Equal(t, wl.IP, leases[i].IP)
		assert.Equal(t, wl.IsStatic, leases[i].IsStatic)

		require.True(t, wl.Expiry.Equal(leases[i].Expiry))
	}
}
