package filtering

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveHTTPLocally starts a new HTTP server, that handles its index with h.  It
// also gracefully closes the listener when the test under t finishes.
func serveHTTPLocally(tb testing.TB, h http.Handler) (urlStr string) {
	tb.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(tb, err)

	go func() { _ = http.Serve(l, h) }()
	testutil.CleanupAndRequireSuccess(tb, l.Close)

	addr := testutil.RequireTypeAssert[*net.TCPAddr](tb, l.Addr())

	return (&url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   addr.String(),
	}).String()
}

// serveFiltersLocally is a helper that concurrently listens on a free port to
// respond with fltContent.
func serveFiltersLocally(tb testing.TB, fltContent []byte) (urlStr string) {
	tb.Helper()

	return serveHTTPLocally(tb, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pt := testutil.PanicT{}

		n, werr := w.Write(fltContent)
		require.NoError(pt, werr)
		require.Equal(pt, len(fltContent), n)
	}))
}

// updateAndAssert loads filter content from its URL and then asserts rules
// count.
func updateAndAssert(
	tb testing.TB,
	ctx context.Context,
	dnsFilter *DNSFilter,
	f *FilterYAML,
	wantUpd require.BoolAssertionFunc,
	wantRulesCount int,
) {
	tb.Helper()

	ok, err := dnsFilter.update(f)
	require.NoError(tb, err)
	wantUpd(tb, ok)

	assert.Equal(tb, wantRulesCount, f.RulesCount)

	dir, err := os.ReadDir(filepath.Join(dnsFilter.conf.DataDir, filterDir))
	require.NoError(tb, err)
	require.FileExists(tb, f.Path(dnsFilter.conf.DataDir))

	assert.Len(tb, dir, 1)

	err = dnsFilter.load(ctx, f)
	require.NoError(tb, err)
}

// newDNSFilter returns a new properly initialized DNS filter instance.
func newDNSFilter(tb testing.TB) (d *DNSFilter) {
	tb.Helper()

	dnsFilter, err := New(&Config{
		Logger:  testLogger,
		DataDir: tb.TempDir(),
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
	}, nil)
	require.NoError(tb, err)

	return dnsFilter
}

func TestDNSFilter_Update(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	const content = `||example.org^$third-party
	# Inline comment example
	||example.com^$third-party
	0.0.0.0 example.com
	`

	fltContent := []byte(content)
	addr := serveFiltersLocally(t, fltContent)
	f := &FilterYAML{
		URL:  addr,
		Name: "test-filter",
	}

	dnsFilter := newDNSFilter(t)

	t.Run("download", func(t *testing.T) {
		updateAndAssert(t, ctx, dnsFilter, f, require.True, 3)
	})

	t.Run("refresh_idle", func(t *testing.T) {
		updateAndAssert(t, ctx, dnsFilter, f, require.False, 3)
	})

	t.Run("refresh_actually", func(t *testing.T) {
		anotherContent := []byte(`||example.com^`)
		oldURL := f.URL

		f.URL = serveFiltersLocally(t, anotherContent)
		t.Cleanup(func() { f.URL = oldURL })

		updateAndAssert(t, ctx, dnsFilter, f, require.True, 1)
	})

	t.Run("load_unload", func(t *testing.T) {
		err := dnsFilter.load(ctx, f)
		require.NoError(t, err)

		f.unload()
	})
}

func TestFilterYAML_EnsureName(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	dnsFilter := newDNSFilter(t)

	t.Run("title_custom", func(t *testing.T) {
		content := []byte("! Title: src-title\n||example.com^")

		f := &FilterYAML{
			URL:  serveFiltersLocally(t, content),
			Name: "user-custom",
		}

		updateAndAssert(t, ctx, dnsFilter, f, require.True, 1)
		assert.Equal(t, "user-custom", f.Name)
	})

	t.Run("title_from_src", func(t *testing.T) {
		content := []byte("! Title: src-title\n||example.com^")

		f := &FilterYAML{
			URL: serveFiltersLocally(t, content),
		}

		updateAndAssert(t, ctx, dnsFilter, f, require.True, 1)
		assert.Equal(t, "src-title", f.Name)
	})

	t.Run("title_default", func(t *testing.T) {
		content := []byte("||example.com^")

		f := &FilterYAML{
			URL: serveFiltersLocally(t, content),
		}

		updateAndAssert(t, ctx, dnsFilter, f, require.True, 1)
		assert.Equal(t, "List 0", f.Name)
	})
}
