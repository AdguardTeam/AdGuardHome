package home

import (
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFltsFileName = "1.txt"

func testStartFilterListener(t *testing.T, fltContent *[]byte) (l net.Listener) {
	t.Helper()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, werr := w.Write(*fltContent)
		require.NoError(t, werr)
		require.Equal(t, len(*fltContent), n)
	})

	var err error
	l, err = net.Listen("tcp", ":0")
	require.NoError(t, err)

	go func() {
		_ = http.Serve(l, h)
	}()
	t.Cleanup(func() {
		require.NoError(t, l.Close())
	})

	return l
}

func TestFilters(t *testing.T) {
	const content = `||example.org^$third-party
	# Inline comment example
	||example.com^$third-party
	0.0.0.0 example.com
	`

	fltContent := []byte(content)

	l := testStartFilterListener(t, &fltContent)

	Context = homeContext{
		workDir: t.TempDir(),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	Context.filters.Init()

	f := &filter{
		URL: (&url.URL{
			Scheme: "http",
			Host: (&netutil.IPPort{
				IP:   net.IP{127, 0, 0, 1},
				Port: l.Addr().(*net.TCPAddr).Port,
			}).String(),
			Path: path.Join(filterDir, testFltsFileName),
		}).String(),
	}

	updateAndAssert := func(t *testing.T, want require.BoolAssertionFunc, wantRulesCount int) {
		ok, err := Context.filters.update(f)
		require.NoError(t, err)
		want(t, ok)

		assert.Equal(t, wantRulesCount, f.RulesCount)

		var dir []fs.DirEntry
		dir, err = os.ReadDir(filepath.Join(Context.getDataDir(), filterDir))
		require.NoError(t, err)

		assert.Len(t, dir, 1)

		require.FileExists(t, f.Path())

		err = Context.filters.load(f)
		require.NoError(t, err)
	}

	t.Run("download", func(t *testing.T) {
		updateAndAssert(t, require.True, 3)
	})

	t.Run("refresh_idle", func(t *testing.T) {
		updateAndAssert(t, require.False, 3)
	})

	t.Run("refresh_actually", func(t *testing.T) {
		fltContent = []byte(`||example.com^`)
		t.Cleanup(func() {
			fltContent = []byte(content)
		})

		updateAndAssert(t, require.True, 1)
	})

	t.Run("load_unload", func(t *testing.T) {
		err := Context.filters.load(f)
		require.NoError(t, err)

		f.unload()
	})

	require.NoError(t, os.Remove(f.Path()))
}
