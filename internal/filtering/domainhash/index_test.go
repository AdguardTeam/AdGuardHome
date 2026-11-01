package domainhash_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeIndex writes the unique hashes of domains, sorted, into a blocklist
// file in a new temporary directory and returns its path.
func writeIndex(t *testing.T, hashBytes int, domains ...string) (path string) {
	t.Helper()

	path = filepath.Join(t.TempDir(), domainhash.FileName)

	hashes := make([]uint64, 0, len(domains))
	for _, d := range domains {
		h, ok := domainhash.HashDomain(d, hashBytes)
		require.True(t, ok, "normalize %q", d)

		hashes = append(hashes, h)
	}

	slices.Sort(hashes)
	hashes = slices.Compact(hashes)

	f, err := os.Create(path)
	require.NoError(t, err, "create")

	var buf [domainhash.MaxHashBytes]byte
	for _, h := range hashes {
		domainhash.PutHash(buf[:], h, hashBytes)

		_, err = f.Write(buf[:hashBytes])
		require.NoError(t, err, "write")
	}

	require.NoError(t, f.Close(), "close")

	return path
}

func TestIndexContainsDomain(t *testing.T) {
	const hashBytes = domainhash.DefaultHashBytes

	path := writeIndex(t, hashBytes, "tracker.example.com", "blocker.example.org")

	idx, err := domainhash.Open(path, hashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	require.Equal(t, 2, idx.Len(), "len")
	require.Equal(t, hashBytes, idx.HashBytes(), "hash bytes")

	// The blocked domain itself.
	assert.True(t, idx.ContainsDomain("tracker.example.com"), "tracker.example.com")

	// The blocked domain is matched through its parent suffix.
	assert.True(t, idx.ContainsDomain("ads.tracker.example.com"), "ads.tracker.example.com")
	assert.True(t, idx.ContainsDomain("a.b.c.tracker.example.com"), "a.b.c.tracker.example.com")

	// Normalization applies to the queried domain too.
	assert.True(t, idx.ContainsDomain("BLOCKER.Example.ORG."), "BLOCKER.Example.ORG.")
	assert.True(t, idx.ContainsDomain("www.blocker.example.org"), "www.blocker.example.org")

	// A blocked child does not block its parents.
	assert.False(t, idx.ContainsDomain("example.com"), "example.com")
	assert.False(t, idx.ContainsDomain("example.org"), "example.org")

	// Unrelated and public-suffix-only domains are never matched.
	assert.False(t, idx.ContainsDomain("example.net"), "example.net")
	assert.False(t, idx.ContainsDomain("tracker.example.net"), "tracker.example.net")
	assert.False(t, idx.ContainsDomain("tracker.example.com.evil.net"), "tracker.example.com.evil.net")
	assert.False(t, idx.ContainsDomain("com"), "com")
	assert.False(t, idx.ContainsDomain("org"), "org")

	// Invalid domains are never matched.
	assert.False(t, idx.ContainsDomain(""), "empty")
	assert.False(t, idx.ContainsDomain("tracker..example.com"), "empty label")
	assert.False(t, idx.ContainsDomain("münchen.example"), "non-ascii")
}

func TestIndexContainsHash(t *testing.T) {
	const hashBytes = domainhash.DefaultHashBytes

	path := writeIndex(t, hashBytes, "example.com", "example.org")

	idx, err := domainhash.Open(path, hashBytes)
	require.NoError(t, err, "open")
	defer func() {
		require.NoError(t, idx.Close(), "close")
	}()

	require.Equal(t, 2, idx.Len(), "len")

	h, ok := domainhash.HashDomain("example.com", hashBytes)
	require.True(t, ok, "ok")
	assert.True(t, idx.ContainsHash(h), "hash")

	// The hash is masked before the lookup, so a hash with extra high bits
	// set matches as well.
	assert.True(t, idx.ContainsHash(h|^(truncMask(hashBytes))), "masked hash")

	// A hash that is not in the file does not match.  With 40 bits of hash
	// space and only two entries in the file, the chance of the hash of
	// "example.net" colliding with one of them is negligible.
	other, ok := domainhash.HashDomain("example.net", hashBytes)
	require.True(t, ok, "ok")
	assert.False(t, idx.ContainsHash(other), "other hash")
	assert.False(t, idx.ContainsHash(0), "zero hash")
}

func TestIndexEmpty(t *testing.T) {
	path := writeIndex(t, domainhash.DefaultHashBytes)

	idx, err := domainhash.Open(path, 0)
	require.NoError(t, err, "open")

	require.Equal(t, 0, idx.Len(), "len")
	assert.Equal(t, domainhash.DefaultHashBytes, idx.HashBytes(), "hash bytes")
	assert.False(t, idx.ContainsDomain("example.com"), "example.com")
	assert.False(t, idx.ContainsHash(0), "hash")

	require.NoError(t, idx.Close(), "close")
	require.NoError(t, idx.Close(), "close again")
}

func TestIndexOpenErrors(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, err := domainhash.Open(filepath.Join(t.TempDir(), "absent.bin"), 0)
		assert.Error(t, err, "err")
	})

	t.Run("bad_size", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), domainhash.FileName)
		require.NoError(t, os.WriteFile(path, []byte{1, 2, 3}, 0o644), "write")

		_, err := domainhash.Open(path, 2)
		assert.ErrorIs(t, err, domainhash.ErrBadIndexSize, "err")
	})

	t.Run("directory", func(t *testing.T) {
		_, err := domainhash.Open(t.TempDir(), 0)
		assert.Error(t, err, "err")
	})
}
