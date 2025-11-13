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
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testURLFilterID is the common rules.ListID for tests.
const testURLFilterID rules.ListID = 1

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
var urlFilterIDCounter = &atomic.Uint64{}

// newURLFilterID returns a new unique URLFilterID.
func newURLFilterID() (id rules.ListID) {
	return rules.ListID(urlFilterIDCounter.Add(1))
}

// newFilter is a helper for creating new filters in tests.  It does not
// register the closing of the filter using t.Cleanup; callers must do that
// either directly or by using the filter in an engine.
func newFilter(tb testing.TB, u *url.URL, name string) (f *rulelist.Filter) {
	tb.Helper()

	f, err := rulelist.NewFilter(&rulelist.FilterConfig{
		URL:         u,
		Name:        name,
		UID:         rulelist.MustNewUID(),
		URLFilterID: newURLFilterID(),
		Enabled:     true,
	})
	require.NoError(tb, err)

	return f
}

// newFilterLocations is a test helper that sets up both the filtering-rule list
// file and the HTTP-server.  It also registers file removal and server stopping
// using t.Cleanup.
func newFilterLocations(
	tb testing.TB,
	cacheDir string,
	fileData string,
	httpData string,
) (fileURL, srvURL *url.URL) {
	tb.Helper()

	f, err := os.CreateTemp(cacheDir, "")
	require.NoError(tb, err)

	err = f.Close()
	require.NoError(tb, err)

	filePath := f.Name()
	err = os.WriteFile(filePath, []byte(fileData), 0o644)
	require.NoError(tb, err)

	testutil.CleanupAndRequireSuccess(tb, func() (err error) {
		return os.Remove(filePath)
	})

	fileURL = &url.URL{
		Scheme: urlutil.SchemeFile,
		Path:   filePath,
	}

	srv := newStringHTTPServer(httpData)
	tb.Cleanup(srv.Close)

	srvURL, err = url.Parse(srv.URL)
	require.NoError(tb, err)

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

func TestIDs(t *testing.T) {
	// Use a variable to prevent compilation errors.
	id := rulelist.IDCustom
	assert.Equal(t, rulelist.APIIDCustom, rulelist.APIID(id))

	id = rulelist.IDBlockedService
	assert.Equal(t, rulelist.APIIDBlockedService, rulelist.APIID(id))

	id = rulelist.IDSafeSearch
	assert.Equal(t, rulelist.APIIDSafeSearch, rulelist.APIID(id))
}
