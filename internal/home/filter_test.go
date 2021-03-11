package home

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testStartFilterListener(t *testing.T) net.Listener {
	t.Helper()

	const content = `||example.org^$third-party
	# Inline comment example
	||example.com^$third-party
	0.0.0.0 example.com
	`

	mux := http.NewServeMux()
	mux.HandleFunc("/filters/1.txt", func(w http.ResponseWriter, r *http.Request) {
		_, werr := w.Write([]byte(content))
		assert.Nil(t, werr)
	})

	listener, err := net.Listen("tcp", ":0")
	require.Nil(t, err)

	go func() {
		_ = http.Serve(listener, mux)
	}()

	t.Cleanup(func() {
		assert.Nil(t, listener.Close())
	})

	return listener
}

func TestFilters(t *testing.T) {
	l := testStartFilterListener(t)
	dir := prepareTestDir(t)

	Context = homeContext{
		workDir: dir,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	Context.filters.Init()

	f := filter{
		URL: fmt.Sprintf("http://127.0.0.1:%d/filters/1.txt", l.Addr().(*net.TCPAddr).Port),
	}

	// Download.
	ok, err := Context.filters.update(&f)
	require.Nil(t, err)
	require.True(t, ok)
	assert.Equal(t, 3, f.RulesCount)

	// Refresh.
	ok, err = Context.filters.update(&f)
	require.Nil(t, err)
	require.False(t, ok)

	err = Context.filters.load(&f)
	require.Nil(t, err)

	f.unload()
	require.Nil(t, os.Remove(f.Path()))
}
