package filtering

import (
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveHTTPLocally starts a new HTTP server, that handles its index with h.  It
// also gracefully closes the listener when the test under t finishes.
func serveHTTPLocally(t *testing.T, h http.Handler) (urlStr string) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() { _ = http.Serve(l, h) }()
	testutil.CleanupAndRequireSuccess(t, l.Close)

	addr := l.Addr()
	require.IsType(t, (*net.TCPAddr)(nil), addr)

	return (&url.URL{
		Scheme: aghhttp.SchemeHTTP,
		Host:   addr.String(),
	}).String()
}

// serveFiltersLocally is a helper that concurrently listens on a free port to
// respond with fltContent.
func serveFiltersLocally(t *testing.T, fltContent []byte) (urlStr string) {
	t.Helper()

	return serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pt := testutil.PanicT{}

		n, werr := w.Write(fltContent)
		require.NoError(pt, werr)
		require.Equal(pt, len(fltContent), n)
	}))
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
		URL: addr,
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

		f.URL = serveFiltersLocally(t, anotherContent)
		t.Cleanup(func() { f.URL = oldURL })

		updateAndAssert(t, require.True, 1)
	})

	t.Run("load_unload", func(t *testing.T) {
		err = filters.load(f)
		require.NoError(t, err)

		f.unload()
	})
}
