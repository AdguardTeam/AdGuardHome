package filtering

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/c2h5oh/datasize"
	"github.com/miekg/dns"
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

	ok, err := dnsFilter.update(f, false)
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

// Rules used by the URL change tests.
const (
	oldRule = "||old.example^"
	newRule = "||new.example^"
)

// newClientListFilter is a helper that returns a DNS filter holding a single
// enabled rule list downloaded from addr, along with the list itself.  white
// tells whether the list is an allowing one.
func newClientListFilter(
	tb testing.TB,
	white bool,
	addr string,
) (d *DNSFilter, flt *FilterYAML) {
	tb.Helper()

	d = newDNSFilter(tb)
	tb.Cleanup(d.Close)

	flt = &FilterYAML{
		Filter:  Filter{ID: 1},
		URL:     addr,
		Name:    "test-filter",
		Enabled: true,
		white:   white,
	}

	_, err := d.update(flt, false)
	require.NoError(tb, err)

	if white {
		d.conf.WhitelistFilters = []FilterYAML{*flt}
	} else {
		d.conf.Filters = []FilterYAML{*flt}
	}

	d.conf.ClientFilterListIDs = clientListIDsFunc(white, map[rules.ListID]bool{flt.ID: true})

	return d, flt
}

// disableAndRepoint disables the only rule list of d while no client uses it,
// then points it at newAddr.
func disableAndRepoint(tb testing.TB, d *DNSFilter, flt *FilterYAML, newAddr string) (err error) {
	tb.Helper()

	// Drop the client reference, so that the list stays out of the engine.
	d.conf.ClientFilterListIDs = nil

	_, err = d.filterSetProperties(flt.URL, FilterYAML{
		Name:    flt.Name,
		URL:     flt.URL,
		Enabled: false,
	}, flt.white)
	require.NoError(tb, err)

	_, err = d.filterSetProperties(flt.URL, FilterYAML{
		Name:    flt.Name,
		URL:     newAddr,
		Enabled: false,
	}, flt.white)

	return err
}

// clientListIDsFunc returns a [Config.ClientFilterListIDs] that reports ids as
// used by clients, as either allowing or blocking lists.
func clientListIDsFunc(
	white bool,
	ids map[rules.ListID]bool,
) (f func() (blockIDs, allowIDs map[rules.ListID]bool)) {
	return func() (blockIDs, allowIDs map[rules.ListID]bool) {
		if white {
			return nil, ids
		}

		return ids, nil
	}
}

func TestDNSFilter_filterSetProperties_disabledURLChange(t *testing.T) {
	const comments = "! Title: Empty list\n! Nothing to see here\n"

	testCases := []struct {
		name            string
		newContent      string
		white           bool
		wantOldFiltered bool
		wantNewFiltered bool
	}{{
		name:            "blocklist_non_empty",
		newContent:      newRule + "\n",
		white:           false,
		wantOldFiltered: false,
		wantNewFiltered: true,
	}, {
		name:            "blocklist_empty",
		newContent:      "",
		white:           false,
		wantOldFiltered: false,
		wantNewFiltered: false,
	}, {
		name:            "blocklist_comments_only",
		newContent:      comments,
		white:           false,
		wantOldFiltered: false,
		wantNewFiltered: false,
	}, {
		name:            "allowlist_non_empty",
		newContent:      "@@" + newRule + "\n",
		white:           true,
		wantOldFiltered: true,
		wantNewFiltered: false,
	}, {
		name:            "allowlist_empty",
		newContent:      "",
		white:           true,
		wantOldFiltered: true,
		wantNewFiltered: true,
	}, {
		name:            "allowlist_comments_only",
		newContent:      comments,
		white:           true,
		wantOldFiltered: true,
		wantNewFiltered: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d, setts := repointDisabledList(t, tc.white, tc.newContent)

			res, err := d.CheckHost("old.example", dns.TypeA, setts)
			require.NoError(t, err)
			assert.Equal(t, tc.wantOldFiltered, res.IsFiltered, "old.example")

			res, err = d.CheckHost("new.example", dns.TypeA, setts)
			require.NoError(t, err)
			assert.Equal(t, tc.wantNewFiltered, res.IsFiltered, "new.example")
		})
	}
}

// repointDisabledList downloads a rule list holding oldRule, disables it while no
// client uses it, points it at a source serving newContent, and assigns it back
// to a client.  It returns the filter and the client settings to match with.
func repointDisabledList(
	t *testing.T,
	white bool,
	newContent string,
) (d *DNSFilter, setts *Settings) {
	t.Helper()

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	prefix := ""
	if white {
		prefix = "@@"
	}

	oldAddr := serveFiltersLocally(t, []byte(prefix+oldRule+"\n"))
	newAddr := serveFiltersLocally(t, []byte(newContent))

	d, flt := newClientListFilter(t, white, oldAddr)
	require.NoError(t, disableAndRepoint(t, d, flt, newAddr))

	ids := map[rules.ListID]bool{flt.ID: true}
	d.conf.ClientFilterListIDs = clientListIDsFunc(white, ids)
	blockIDs, allowIDs := d.clientFilterListIDs()

	setts = &Settings{
		ProtectionEnabled: true,
		FilteringEnabled:  true,
		UseOwnFilterLists: true,
	}

	if white {
		setts.ClientAllowListIDs = ids
		// Block both hosts globally, so that only an allowing rule can change
		// the outcome.
		d.conf.UserRules = []string{oldRule, newRule}
	} else {
		setts.ClientFilterListIDs = ids
	}

	_ = d.enableFiltersLocked(ctx, blockIDs, allowIDs, nil, false)

	return d, setts
}

func TestDNSFilter_filterSetProperties_urlChangeRollback(t *testing.T) {
	oldAddr := serveFiltersLocally(t, []byte(oldRule+"\n"))
	badAddr := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	d, flt := newClientListFilter(t, false, oldAddr)

	// Drop the client reference, so that the list stays out of the engine.
	d.conf.ClientFilterListIDs = nil

	_, err := d.filterSetProperties(flt.URL, FilterYAML{
		Name:    flt.Name,
		URL:     flt.URL,
		Enabled: false,
	}, false)
	require.NoError(t, err)

	before := d.conf.Filters[0]

	_, err = d.filterSetProperties(flt.URL, FilterYAML{
		Name:    flt.Name,
		URL:     badAddr,
		Enabled: false,
	}, false)
	require.Error(t, err)

	// A failed download must leave the entry exactly as it was, so that the
	// metadata keeps agreeing with the contents on disk.
	assert.Equal(t, before, d.conf.Filters[0])
	assert.FileExists(t, d.conf.Filters[0].Path(d.conf.DataDir))
}

// newDisabledListFilter returns a DNS filter whose only rule list is disabled and
// unreferenced, served from addr.  The list is left out of the engine, as it
// would be after a restart.
func newDisabledListFilter(tb testing.TB, addr string) (d *DNSFilter, id rules.ListID) {
	tb.Helper()

	d = newDNSFilter(tb)
	tb.Cleanup(d.Close)

	id = 1
	d.conf.Filters = []FilterYAML{{
		Filter:  Filter{ID: id},
		URL:     addr,
		Name:    "test-filter",
		Enabled: false,
	}}

	// Never refresh on a timer, as with filters_update_interval set to zero.
	d.conf.FiltersUpdateIntervalHours = 0

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	_ = d.enableFiltersLocked(ctx, nil, nil, nil, false)

	return d, id
}

// clientSettings returns the settings of a client using only the blocking list
// ids.
func clientSettings(ids ...rules.ListID) (setts *Settings) {
	set := make(map[rules.ListID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}

	return &Settings{
		ProtectionEnabled:   true,
		FilteringEnabled:    true,
		UseOwnFilterLists:   true,
		ClientFilterListIDs: set,
	}
}

func TestDNSFilter_SetClientFilterLists(t *testing.T) {
	t.Run("absent_cache", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		addr := serveFiltersLocally(t, []byte(newRule+"\n"))

		d, id := newDisabledListFilter(t, addr)
		setts := clientSettings(id)

		// Nothing is cached yet, so the promised list cannot match.
		res, err := d.CheckHost("new.example", dns.TypeA, setts)
		require.NoError(t, err)
		require.False(t, res.IsFiltered)

		ids := map[rules.ListID]bool{id: true}
		require.NoError(t, d.SetClientFilterLists(ctx, ids, nil))

		// The list must be downloaded and live once the call returns, since the
		// client policy is published right after.
		require.FileExists(t, d.conf.Filters[0].Path(d.conf.DataDir))

		res, err = d.CheckHost("new.example", dns.TypeA, setts)
		require.NoError(t, err)
		assert.True(t, res.IsFiltered)
	})

	t.Run("stale_cache", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)

		// Serve the new rule, but leave the old one cached on disk.
		addr := serveFiltersLocally(t, []byte(newRule+"\n"))
		d, id := newDisabledListFilter(t, addr)

		path := d.conf.Filters[0].Path(d.conf.DataDir)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), aghos.DefaultPermDir))
		require.NoError(t, os.WriteFile(path, []byte(oldRule+"\n"), aghos.DefaultPermFile))

		ids := map[rules.ListID]bool{id: true}
		require.NoError(t, d.SetClientFilterLists(ctx, ids, nil))

		setts := clientSettings(id)

		res, err := d.CheckHost("new.example", dns.TypeA, setts)
		require.NoError(t, err)
		assert.True(t, res.IsFiltered, "the refreshed rule must apply")

		res, err = d.CheckHost("old.example", dns.TypeA, setts)
		require.NoError(t, err)
		assert.False(t, res.IsFiltered, "the stale rule must be gone")
	})

	t.Run("already_referenced_but_cache_gone", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		addr := serveFiltersLocally(t, []byte(newRule+"\n"))

		d, id := newDisabledListFilter(t, addr)
		ids := map[rules.ListID]bool{id: true}

		// A client already uses the list, but its cache never made it to disk,
		// so being referenced is not enough to consider it enforceable.
		d.conf.ClientFilterListIDs = func() (blockIDs, allowIDs map[rules.ListID]bool) {
			return ids, nil
		}

		require.NoError(t, d.SetClientFilterLists(ctx, ids, nil))
		require.FileExists(t, d.conf.Filters[0].Path(d.conf.DataDir))

		res, err := d.CheckHost("new.example", dns.TypeA, clientSettings(id))
		require.NoError(t, err)
		assert.True(t, res.IsFiltered)
	})

	t.Run("download_failure_with_cache", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		addr := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		d, id := newDisabledListFilter(t, addr)

		// A cached copy is enough to enforce the policy, so a failed refresh
		// must not block the client update.
		path := d.conf.Filters[0].Path(d.conf.DataDir)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), aghos.DefaultPermDir))
		require.NoError(t, os.WriteFile(path, []byte(newRule+"\n"), aghos.DefaultPermFile))

		require.NoError(t, d.SetClientFilterLists(ctx, map[rules.ListID]bool{id: true}, nil))

		res, err := d.CheckHost("new.example", dns.TypeA, clientSettings(id))
		require.NoError(t, err)
		assert.True(t, res.IsFiltered, "the cached rule must apply")
	})

	t.Run("download_failure_no_cache", func(t *testing.T) {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		addr := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		d, id := newDisabledListFilter(t, addr)

		// The caller must learn about it and keep the policy unpublished.
		err := d.SetClientFilterLists(ctx, map[rules.ListID]bool{id: true}, nil)
		require.Error(t, err)

		assert.NoFileExists(t, d.conf.Filters[0].Path(d.conf.DataDir))
	})
}

// TestDNSFilter_SetClientFilterLists_transition asserts that moving a client
// from the global lists onto a globally disabled one never leaves it matching
// neither policy.  The engine must hold both the old and the prospective lists
// by the time the call returns, since the client policy is published right
// after.
func TestDNSFilter_SetClientFilterLists_transition(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	globalAddr := serveFiltersLocally(t, []byte(oldRule+"\n"))
	ownAddr := serveFiltersLocally(t, []byte(newRule+"\n"))

	const (
		globalID rules.ListID = 1
		ownID    rules.ListID = 2
	)

	d := newDNSFilter(t)
	t.Cleanup(d.Close)

	globalFlt := FilterYAML{
		Filter:  Filter{ID: globalID},
		URL:     globalAddr,
		Name:    "global-filter",
		Enabled: true,
	}
	_, err := d.update(&globalFlt, false)
	require.NoError(t, err)

	d.conf.Filters = []FilterYAML{globalFlt, {
		Filter:  Filter{ID: ownID},
		URL:     ownAddr,
		Name:    "own-filter",
		Enabled: false,
	}}
	d.conf.FiltersUpdateIntervalHours = 0
	_ = d.enableFiltersLocked(ctx, nil, nil, nil, false)

	globalSetts := &Settings{ProtectionEnabled: true, FilteringEnabled: true}

	// The client starts on the global lists.
	res, err := d.CheckHost("old.example", dns.TypeA, globalSetts)
	require.NoError(t, err)
	require.True(t, res.IsFiltered)

	require.NoError(t, d.SetClientFilterLists(ctx, map[rules.ListID]bool{ownID: true}, nil))

	// The global list must still be enforceable for everyone else, and the newly
	// referenced one must already work for the client about to be stored.
	res, err = d.CheckHost("old.example", dns.TypeA, globalSetts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered, "the global list must stay live during the transition")

	ownSetts := clientSettings(ownID)

	res, err = d.CheckHost("new.example", dns.TypeA, ownSetts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered, "the newly referenced list must already be enforceable")
}

// TestDNSFilter_SetClientFilterLists_noop asserts that saving a client whose
// lists the engine already holds neither downloads anything nor rebuilds, so
// that an unrelated edit stays cheap.
func TestDNSFilter_SetClientFilterLists_noop(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	var hits atomic.Int64
	addr := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(newRule + "\n"))
	}))

	const id rules.ListID = 1

	d := newDNSFilter(t)
	t.Cleanup(d.Close)

	// An enabled list is always in the engine, so referencing it changes
	// nothing.
	flt := FilterYAML{
		Filter:  Filter{ID: id},
		URL:     addr,
		Name:    "test-filter",
		Enabled: true,
	}
	_, err := d.update(&flt, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), hits.Load())

	d.conf.Filters = []FilterYAML{flt}
	_ = d.enableFiltersLocked(ctx, nil, nil, nil, false)

	require.NoError(t, d.SetClientFilterLists(ctx, map[rules.ListID]bool{id: true}, nil))

	assert.Equal(t, int64(1), hits.Load(), "an enabled list must not be downloaded again")
}

func TestDNSFilter_SetClientFilterLists_unknownID(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	addr := serveFiltersLocally(t, []byte(newRule+"\n"))

	d, id := newDisabledListFilter(t, addr)

	t.Run("blocking", func(t *testing.T) {
		err := d.SetClientFilterLists(ctx, map[rules.ListID]bool{12345: true}, nil)
		require.ErrorIs(t, err, ErrUnknownListID)
	})

	t.Run("wrong_category", func(t *testing.T) {
		// The list is a blocking one, so naming it as an allowing one must be
		// rejected rather than silently doing nothing.
		err := d.SetClientFilterLists(ctx, nil, map[rules.ListID]bool{id: true})
		require.ErrorIs(t, err, ErrUnknownListID)
	})

	t.Run("known_is_accepted", func(t *testing.T) {
		require.NoError(t, d.SetClientFilterLists(ctx, map[rules.ListID]bool{id: true}, nil))
	})
}

func TestDNSFilter_SetClientFilterLists_missingGlobalFile(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	globalAddr := serveFiltersLocally(t, []byte(oldRule+"\n"))
	ownAddr := serveFiltersLocally(t, []byte(newRule+"\n"))

	const (
		globalID rules.ListID = 1
		ownID    rules.ListID = 2
	)

	d := newDNSFilter(t)
	t.Cleanup(d.Close)

	globalFlt := FilterYAML{
		Filter:  Filter{ID: globalID},
		URL:     globalAddr,
		Name:    "global-filter",
		Enabled: true,
	}
	_, err := d.update(&globalFlt, false)
	require.NoError(t, err)

	d.conf.Filters = []FilterYAML{globalFlt, {
		Filter:  Filter{ID: ownID},
		URL:     ownAddr,
		Name:    "own-filter",
		Enabled: false,
	}}
	d.conf.FiltersUpdateIntervalHours = 0
	require.NoError(t, d.enableFiltersLocked(ctx, nil, nil, nil, false))

	globalSetts := &Settings{ProtectionEnabled: true, FilteringEnabled: true}

	res, err := d.CheckHost("old.example", dns.TypeA, globalSetts)
	require.NoError(t, err)
	require.True(t, res.IsFiltered)

	// The global list's cache disappears behind our back.
	require.NoError(t, os.Remove(globalFlt.Path(d.conf.DataDir)))

	// Assigning the disabled list must not swap in an engine that silently lost
	// the global list.
	err = d.SetClientFilterLists(ctx, map[rules.ListID]bool{ownID: true}, nil)
	require.Error(t, err)

	// The previous engine must still be enforcing it.
	res, err = d.CheckHost("old.example", dns.TypeA, globalSetts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered, "the old engine must be retained")
}
