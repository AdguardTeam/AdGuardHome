package filtering

import (
	"io/fs"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveFiltersLocally is a helper that concurrently listens on a free port to
// respond with fltContent.  It also gracefully closes the listener when the
// test under t finishes.
func serveFiltersLocally(t *testing.T, fltContent []byte) (ipp netip.AddrPort) {
	t.Helper()

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pt := testutil.PanicT{}

		n, werr := w.Write(fltContent)
		require.NoError(pt, werr)
		require.Equal(pt, len(fltContent), n)
	})

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() { _ = http.Serve(l, h) }()
	testutil.CleanupAndRequireSuccess(t, l.Close)

	addr := l.Addr()
	require.IsType(t, new(net.TCPAddr), addr)

	return netip.AddrPortFrom(aghnet.IPv4Localhost(), uint16(addr.(*net.TCPAddr).Port))
}

func TestFilters(t *testing.T) {
	const content = `||example.org^$third-party
	# Inline comment example
	||example.com^$third-party
	0.0.0.0 example.com
	`

	fltContent := []byte(content)

	addr := serveFiltersLocally(t, fltContent)

	tempDir := t.TempDir()

	filters, err := New(&Config{
		DataDir: tempDir,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}, nil)
	require.NoError(t, err)

	f := &FilterYAML{
		URL: (&url.URL{
			Scheme: "http",
			Host:   addr.String(),
		}).String(),
	}

	updateAndAssert := func(t *testing.T, want require.BoolAssertionFunc, wantRulesCount int) {
		var ok bool
		ok, err = filters.update(f)
		require.NoError(t, err)
		want(t, ok)

		assert.Equal(t, wantRulesCount, f.RulesCount)

		var dir []fs.DirEntry
		dir, err = os.ReadDir(filepath.Join(tempDir, filterDir))
		require.NoError(t, err)

		assert.Len(t, dir, 1)

		require.FileExists(t, f.Path(tempDir))

		err = filters.load(f)
		require.NoError(t, err)
	}

	t.Run("download", func(t *testing.T) {
		updateAndAssert(t, require.True, 3)
	})

	t.Run("refresh_idle", func(t *testing.T) {
		updateAndAssert(t, require.False, 3)
	})

	t.Run("refresh_actually", func(t *testing.T) {
		anotherContent := []byte(`||example.com^`)
		oldURL := f.URL

		ipp := serveFiltersLocally(t, anotherContent)
		f.URL = (&url.URL{
			Scheme: "http",
			Host:   ipp.String(),
		}).String()
		t.Cleanup(func() { f.URL = oldURL })

		updateAndAssert(t, require.True, 1)
	})

	t.Run("load_unload", func(t *testing.T) {
		err = filters.load(f)
		require.NoError(t, err)

		f.unload()
	})
}
