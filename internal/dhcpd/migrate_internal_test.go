package dhcpd

import (
	"encoding/json"
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

	wantLeases := []*dbLease{{
		Expiry:   time.Unix(1, 0).Format(time.RFC3339),
		Hostname: "test1",
		HWAddr:   "11:22:33:44:55:66",
		IP:       netip.MustParseAddr("1.2.3.4"),
		IsStatic: true,
	}, {
		Expiry:   time.Unix(1231231231, 0).Format(time.RFC3339),
		Hostname: "test2",
		HWAddr:   "66:55:44:33:22:11",
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

	for i, wantLease := range wantLeases {
		assert.Equal(t, wantLease.Hostname, leases[i].Hostname)
		assert.Equal(t, wantLease.HWAddr, leases[i].HWAddr)
		assert.Equal(t, wantLease.IP, leases[i].IP)
		assert.Equal(t, wantLease.IsStatic, leases[i].IsStatic)

		require.Equal(t, wantLease.Expiry, leases[i].Expiry)
	}
}
