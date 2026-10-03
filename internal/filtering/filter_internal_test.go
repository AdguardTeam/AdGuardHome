package filtering

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/c2h5oh/datasize"
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

	pt := testutil.NewPanicT(tb)

	return serveHTTPLocally(tb, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

// testFilterSize is a test size of filters.
const testFilterSize = 10 * datasize.MB

// newDNSFilter returns a new properly initialized DNS filter instance.
func newDNSFilter(tb testing.TB) (d *DNSFilter) {
	tb.Helper()

	dnsFilter, err := New(&Config{
		Logger:  testLogger,
		DataDir: tb.TempDir(),
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		MaxHTTPSize: testFilterSize,
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

func TestDNSFilter_UpdateHTTPValidators(t *testing.T) {
	const (
		contentV1      = "||one.example^\n"
		contentV2      = "||two.example^\n"
		etagV1         = `"v1"`
		etagV2         = `"v2"`
		lastModifiedV1 = "Wed, 21 Oct 2015 07:28:00 GMT"
		lastModifiedV2 = "Thu, 22 Oct 2015 07:28:00 GMT"
	)

	requests := make(chan http.Header, 6)
	requestNum := &atomic.Uint32{}
	pt := testutil.NewPanicT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()

		switch requestNum.Add(1) {
		case 1:
			w.Header().Set("ETag", etagV1)
			w.Header().Set("Last-Modified", lastModifiedV1)
			n, err := w.Write([]byte(contentV1))
			require.NoError(pt, err)
			require.Equal(pt, len(contentV1), n)
		case 2:
			w.WriteHeader(http.StatusNotModified)
		case 3:
			w.Header().Set("ETag", etagV2)
			w.Header().Set("Last-Modified", lastModifiedV2)
			n, err := w.Write([]byte(contentV2))
			require.NoError(pt, err)
			require.Equal(pt, len(contentV2), n)
		case 4:
			http.Error(w, "test error", http.StatusInternalServerError)
		case 5:
			w.WriteHeader(http.StatusNotModified)
		case 6:
			w.Header().Set("ETag", etagV2)
			w.Header().Set("Last-Modified", lastModifiedV2)
			n, err := w.Write([]byte(contentV2))
			require.NoError(pt, err)
			require.Equal(pt, len(contentV2), n)
		default:
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	httpClient := srv.Client()
	httpClient.Timeout = testTimeout

	dataDir := t.TempDir()
	const filterID = 1
	filterPath := (&FilterYAML{Filter: Filter{ID: filterID}}).Path(dataDir)

	update := func() (updated bool, err error) {
		d, newErr := New(&Config{
			Logger:      testLogger,
			DataDir:     dataDir,
			HTTPClient:  httpClient,
			MaxHTTPSize: testFilterSize,
			Filters: []FilterYAML{{
				Enabled: true,
				URL:     srv.URL,
				Name:    "test-filter",
				Filter: Filter{
					ID: filterID,
				},
			}},
		}, nil)
		require.NoError(t, newErr)
		defer d.Close()

		return d.update(&d.conf.Filters[0])
	}

	assertRequestValidators := func(wantETag, wantLastModified string) {
		t.Helper()

		reqHeader := <-requests
		assert.Equal(t, wantETag, reqHeader.Get("If-None-Match"))
		assert.Equal(t, wantLastModified, reqHeader.Get("If-Modified-Since"))
	}

	updated, err := update()
	require.NoError(t, err)
	assert.True(t, updated)
	assertRequestValidators("", "")

	contentBeforeNotModified, err := os.ReadFile(filterPath)
	require.NoError(t, err)

	updated, err = update()
	assertRequestValidators(etagV1, lastModifiedV1)
	require.NoError(t, err)
	assert.False(t, updated)

	contentAfterNotModified, err := os.ReadFile(filterPath)
	require.NoError(t, err)
	assert.Equal(t, contentBeforeNotModified, contentAfterNotModified)

	updated, err = update()
	require.NoError(t, err)
	assert.True(t, updated)
	assertRequestValidators(etagV1, lastModifiedV1)

	contentV2OnDisk, err := os.ReadFile(filterPath)
	require.NoError(t, err)
	require.NotEqual(t, contentBeforeNotModified, contentV2OnDisk)

	updated, err = update()
	assertRequestValidators(etagV2, lastModifiedV2)
	require.Error(t, err)
	assert.False(t, updated)

	contentAfterError, err := os.ReadFile(filterPath)
	require.NoError(t, err)
	assert.Equal(t, contentV2OnDisk, contentAfterError)

	updated, err = update()
	assertRequestValidators(etagV2, lastModifiedV2)
	require.NoError(t, err)
	assert.False(t, updated)

	contentAfterFinalNotModified, err := os.ReadFile(filterPath)
	require.NoError(t, err)
	assert.Equal(t, contentV2OnDisk, contentAfterFinalNotModified)

	// Changing the filter contents invalidates the validators cached for the
	// previous representation, so the next request must be unconditional.
	require.NoError(t, os.WriteFile(filterPath, []byte(contentV1), 0o600))
	updated, err = update()
	assertRequestValidators("", "")
	require.NoError(t, err)
	assert.True(t, updated)

	contentAfterUnconditionalUpdate, err := os.ReadFile(filterPath)
	require.NoError(t, err)
	assert.Equal(t, contentV2OnDisk, contentAfterUnconditionalUpdate)

	dir, err := os.ReadDir(filepath.Join(dataDir, filterDir))
	require.NoError(t, err)
	assert.Len(t, dir, 2)
	assert.Equal(t, uint32(6), requestNum.Load())
}

func TestDNSFilter_UpdateHTTPValidatorsInvalidHeaderValue(t *testing.T) {
	const (
		contentV1 = "||one.example^\n"
		contentV2 = "||two.example^\n"
	)

	requests := make(chan http.Header, 2)
	requestNum := &atomic.Uint32{}
	pt := testutil.NewPanicT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()

		content := contentV2
		if requestNum.Add(1) == 1 {
			w.Header().Set("ETag", `"v1"`)
			content = contentV1
		}

		n, err := w.Write([]byte(content))
		require.NoError(pt, err)
		require.Equal(pt, len(content), n)
	}))
	t.Cleanup(srv.Close)

	d := newDNSFilter(t)
	flt := &FilterYAML{
		URL: srv.URL,
		Filter: Filter{
			ID: 1,
		},
	}

	updated, err := d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	digest, err := d.currentFilterDigest(flt)
	require.NoError(t, err)
	metadata, err := json.Marshal(filterHTTPMetadata{
		URL:    srv.URL,
		ETag:   "invalid\x01etag",
		Digest: digest,
	})
	require.NoError(t, err)
	require.True(t, json.Valid(metadata))
	require.NoError(t, os.WriteFile(flt.httpMetadataPath(d.conf.DataDir), metadata, 0o600))

	updated, err = d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))
	assert.Equal(t, uint32(2), requestNum.Load())

	content, err := os.ReadFile(flt.Path(d.conf.DataDir))
	require.NoError(t, err)
	assert.Equal(t, contentV2, string(content))
}

func TestDNSFilter_UpdateHTTPValidatorsEditedContents(t *testing.T) {
	const (
		serverContent = "||server.example^\n"
		editedContent = "||edited.example^\n"
	)

	requests := make(chan http.Header, 2)
	pt := testutil.NewPanicT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		w.Header().Set("ETag", `"v1"`)

		n, err := w.Write([]byte(serverContent))
		require.NoError(pt, err)
		require.Equal(pt, len(serverContent), n)
	}))
	t.Cleanup(srv.Close)

	d := newDNSFilter(t)
	flt := &FilterYAML{
		URL: srv.URL,
		Filter: Filter{
			ID: 1,
		},
	}

	updated, err := d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	require.NoError(t, os.WriteFile(flt.Path(d.conf.DataDir), []byte(editedContent), 0o600))

	updated, err = d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	content, err := os.ReadFile(flt.Path(d.conf.DataDir))
	require.NoError(t, err)
	assert.Equal(t, serverContent, string(content))
}

func TestDNSFilter_UpdateHTTPValidatorsEditedContentsMatchServer(t *testing.T) {
	const (
		contentV1 = "||one.example^\n"
		contentV2 = "||two.example^\n"
	)

	requests := make(chan http.Header, 2)
	requestNum := &atomic.Uint32{}
	pt := testutil.NewPanicT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()

		content := contentV2
		etag := `"v2"`
		if requestNum.Add(1) == 1 {
			content = contentV1
			etag = `"v1"`
		}

		w.Header().Set("ETag", etag)
		n, err := w.Write([]byte(content))
		require.NoError(pt, err)
		require.Equal(pt, len(content), n)
	}))
	t.Cleanup(srv.Close)

	d := newDNSFilter(t)
	flt := &FilterYAML{
		URL: srv.URL,
		Filter: Filter{
			ID: 1,
		},
	}

	updated, err := d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))
	checksumV1 := flt.checksum

	// Simulate an external atomic replacement of the cache with the same V2
	// representation that the server now returns.  The active filter is still
	// V1, so the update must be reported even though disk and server match.
	require.NoError(t, os.WriteFile(flt.Path(d.conf.DataDir), []byte(contentV2), 0o600))

	updated, err = d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))
	assert.NotEqual(t, checksumV1, flt.checksum)
}

func TestDNSFilter_UpdateHTTPValidatorsDifferentContentsSameChecksum(t *testing.T) {
	const (
		serverContent = "ab.com\ncd.com\n"
		editedContent = "ab.comc\nd.com\n"
		etag          = `"v1"`
	)

	requests := make(chan http.Header, 2)
	pt := testutil.NewPanicT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		w.Header().Set("ETag", etag)
		n, err := w.Write([]byte(serverContent))
		require.NoError(pt, err)
		require.Equal(pt, len(serverContent), n)
	}))
	t.Cleanup(srv.Close)

	d := newDNSFilter(t)
	flt := &FilterYAML{
		URL: srv.URL,
		Filter: Filter{
			ID: 1,
		},
	}

	updated, err := d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	res, err := rulelist.NewParser().Parse(
		io.Discard,
		strings.NewReader(editedContent),
		make([]byte, rulelist.DefaultRuleBufSize),
	)
	require.NoError(t, err)
	require.Equal(t, flt.checksum, res.Checksum)
	require.NoError(t, os.WriteFile(flt.Path(d.conf.DataDir), []byte(editedContent), 0o600))

	updated, err = d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	content, err := os.ReadFile(flt.Path(d.conf.DataDir))
	require.NoError(t, err)
	assert.Equal(t, serverContent, string(content))
}

func TestDNSFilter_UpdateHTTPValidatorsMissingContents(t *testing.T) {
	const serverContent = "||server.example^\n"

	requests := make(chan http.Header, 2)
	pt := testutil.NewPanicT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		w.Header().Set("ETag", `"v1"`)

		n, err := w.Write([]byte(serverContent))
		require.NoError(pt, err)
		require.Equal(pt, len(serverContent), n)
	}))
	t.Cleanup(srv.Close)

	d := newDNSFilter(t)
	flt := &FilterYAML{
		URL: srv.URL,
		Filter: Filter{
			ID: 1,
		},
	}

	updated, err := d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	filterPath := flt.Path(d.conf.DataDir)
	require.NoError(t, os.Remove(filterPath))

	updated, err = d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, (<-requests).Get("If-None-Match"))

	content, err := os.ReadFile(filterPath)
	require.NoError(t, err)
	assert.Equal(t, serverContent, string(content))
}

func TestDNSFilter_UpdateHTTPValidatorsRedirect(t *testing.T) {
	const (
		contentV1 = "||one.example^\n"
		contentV2 = "||two.example^\n"
		etag      = `"same"`
		lastMod   = "Wed, 21 Oct 2015 07:28:00 GMT"
	)

	targetRequests := make(chan http.Header, 1)
	pt := testutil.NewPanicT(t)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests <- r.Header.Clone()
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		w.Header().Set("ETag", etag)
		n, err := w.Write([]byte(contentV2))
		require.NoError(pt, err)
		require.Equal(pt, len(contentV2), n)
	}))
	t.Cleanup(target.Close)

	requestNum := &atomic.Uint32{}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestNum.Add(1) == 1 {
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", lastMod)
			n, err := w.Write([]byte(contentV1))
			require.NoError(pt, err)
			require.Equal(pt, len(contentV1), n)

			return
		}

		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(origin.Close)

	d := newDNSFilter(t)
	redirectNum := &atomic.Uint32{}
	d.conf.HTTPClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) (err error) {
		redirectNum.Add(1)

		return nil
	}
	flt := &FilterYAML{
		URL: origin.URL,
		Filter: Filter{
			ID: 1,
		},
	}

	updated, err := d.update(flt)
	require.NoError(t, err)
	require.True(t, updated)
	require.FileExists(t, flt.httpMetadataPath(d.conf.DataDir))

	updated, err = d.update(flt)
	targetHeader := <-targetRequests
	require.NoError(t, err)
	require.True(t, updated)
	assert.Empty(t, targetHeader.Get("If-None-Match"))
	assert.Empty(t, targetHeader.Get("If-Modified-Since"))
	assert.Equal(t, uint32(1), redirectNum.Load())

	content, err := os.ReadFile(flt.Path(d.conf.DataDir))
	require.NoError(t, err)
	assert.Equal(t, contentV2, string(content))
	require.NoFileExists(t, flt.httpMetadataPath(d.conf.DataDir))
}

func TestDNSFilter_CurrentHTTPFilterStateMetadataSelection(t *testing.T) {
	testCases := []struct {
		name     string
		metadata []byte
	}{
		{
			name: "missing",
		}, {
			name:     "malformed",
			metadata: []byte("{"),
		}, {
			name:     "validator_free",
			metadata: []byte(`{"url":"https://example.org/filter.txt","digest":"bad"}`),
		}, {
			name: "invalid_digest",
			metadata: []byte(
				`{"url":"https://example.org/filter.txt","etag":"\"v1\"","digest":"bad"}`,
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := newDNSFilter(t)
			flt := &FilterYAML{
				URL:        "https://example.org/filter.txt",
				RulesCount: 1,
				checksum:   1,
				Filter: Filter{
					ID: 1,
				},
			}
			require.NoError(t, os.WriteFile(
				flt.Path(d.conf.DataDir),
				[]byte("<html></html>"),
				0o600,
			))
			if tc.metadata != nil {
				require.NoError(t, os.WriteFile(
					flt.httpMetadataPath(d.conf.DataDir),
					tc.metadata,
					0o600,
				))
			}

			ctx := context.Background()
			metadata, hasMetadata := d.loadHTTPMetadata(ctx, flt)
			digest, hasContents, useMetadata, forceUpdate := d.currentHTTPFilterState(
				ctx,
				flt,
				metadata,
				hasMetadata,
			)
			assert.NotEmpty(t, digest)
			assert.True(t, hasContents)
			assert.False(t, useMetadata)
			assert.False(t, forceUpdate)
		})
	}
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
