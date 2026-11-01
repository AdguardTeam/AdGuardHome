//go:build unix

package domainhash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blocklist.bin")
	require.NoError(t, os.WriteFile(path, make([]byte, 4096), 0o644), "write")

	f, err := os.Open(path)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, f.Close(), "close")
	}()

	data, unmap, err := mapFile(f, 4096)
	require.NoError(t, err, "map")
	require.Len(t, data, 4096, "len")
	require.NoError(t, unmap(), "unmap")

	// An empty file is not mapped, which is not an error.
	data, unmap, err = mapFile(f, 0)
	require.NoError(t, err, "map empty")
	assert.Nil(t, data, "data")
	require.NoError(t, unmap(), "unmap")
}
