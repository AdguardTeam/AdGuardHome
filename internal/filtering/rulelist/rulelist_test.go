package rulelist_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testURLFilterID is the common [rulelist.URLFilterID] for tests.
const testURLFilterID rulelist.URLFilterID = 1

// testTitle is the common title for tests.
const testTitle = "Test Title"

// Common rule texts for tests.
const (
	testRuleTextAllowed     = "||allowed.example^\n"
	testRuleTextBadTab      = "||bad-tab-and-comment.example^\t# A comment.\n"
	testRuleTextBlocked     = "||blocked.example^\n"
	testRuleTextBlocked2    = "||blocked-2.example^\n"
	testRuleTextEtcHostsTab = "0.0.0.0 tab..example^\t# A comment.\n"
	testRuleTextHTML        = "<!DOCTYPE html>\n"
	testRuleTextTitle       = "! Title:  " + testTitle + " \n"

	// testRuleTextCosmetic is a cosmetic rule with a zero-width non-joiner.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/6003.
	testRuleTextCosmetic = "||cosmetic.example## :has-text(/\u200c/i)\n"
)

// urlFilterIDCounter is the atomic integer used to create unique filter IDs.
var urlFilterIDCounter = &atomic.Int32{}

// newURLFilterID returns a new unique URLFilterID.
func newURLFilterID() (id rulelist.URLFilterID) {
	return rulelist.URLFilterID(urlFilterIDCounter.Add(1))
}

// newFilter is a helper for creating new filters in tests.  It does not
// register the closing of the filter using t.Cleanup; callers must do that
// either directly or by using the filter in an engine.
func newFilter(t testing.TB, u *url.URL, name string) (f *rulelist.Filter) {
	t.Helper()

	f, err := rulelist.NewFilter(&rulelist.FilterConfig{
		URL:         u,
		Name:        name,
		UID:         rulelist.MustNewUID(),
		URLFilterID: newURLFilterID(),
		Enabled:     true,
	})
	require.NoError(t, err)

	return f
}

// newFilterLocations is a test helper that sets up both the filtering-rule list
// file and the HTTP-server.  It also registers file removal and server stopping
// using t.Cleanup.
func newFilterLocations(
	t testing.TB,
	cacheDir string,
	fileData string,
	httpData string,
) (fileURL, srvURL *url.URL) {
	t.Helper()

	f, err := os.CreateTemp(cacheDir, "")
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	filePath := f.Name()
	err = os.WriteFile(filePath, []byte(fileData), 0o644)
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return os.Remove(filePath)
	})

	fileURL = &url.URL{
		Scheme: urlutil.SchemeFile,
		Path:   filePath,
	}

	srv := newStringHTTPServer(httpData)
	t.Cleanup(srv.Close)

	srvURL, err = url.Parse(srv.URL)
	require.NoError(t, err)

	return fileURL, srvURL
}

// newStringHTTPServer returns a new HTTP server that serves s.
func newStringHTTPServer(s string) (srv *httptest.Server) {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pt := testutil.PanicT{}

		_, err := io.WriteString(w, s)
		require.NoError(pt, err)
	}))
}
