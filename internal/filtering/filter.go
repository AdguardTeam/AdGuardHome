package filtering

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghrenameio"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/urlfilter/rules"
)

// filterDir is the subdirectory of a data directory to store downloaded
// filters.
const filterDir = "filters"

// FilterYAML represents a filter list in the configuration file.
//
// TODO(e.burkov):  Investigate if the field ordering is important.
type FilterYAML struct {
	Enabled bool
	// TODO(m.kazantsev):  Refactor.
	URL         string    // URL or a file path
	Name        string    `yaml:"name"`
	RulesCount  int       `yaml:"-"`
	LastUpdated time.Time `yaml:"-"`
	checksum    uint32    // checksum of the file data
	white       bool

	Filter `yaml:",inline"`
}

// Clear filter rules
func (filter *FilterYAML) unload() {
	filter.RulesCount = 0
	filter.checksum = 0
}

// Path to the filter contents
func (filter *FilterYAML) Path(dataDir string) string {
	return filepath.Join(
		dataDir,
		filterDir,
		strconv.FormatUint(uint64(filter.ID), 10)+".txt",
	)
}

// ensureName sets provided title or default name for the filter if it doesn't
// have name already.
func (filter *FilterYAML) ensureName(title string) {
	if filter.Name != "" {
		return
	}

	if title != "" {
		filter.Name = title

		return
	}

	filter.Name = fmt.Sprintf("List %d", filter.ID)
}

const (
	// errFilterNotExist is returned from [filterSetProperties] when there are
	// no lists with the desired URL to update.
	//
	// TODO(e.burkov):  Use wherever the same error is needed.
	errFilterNotExist errors.Error = "url doesn't exist"

	// errFilterExists is returned from [filterSetProperties] when there is
	// another filter having the same URL as the one updated.
	//
	// TODO(e.burkov):  Use wherever the same error is needed.
	errFilterExists errors.Error = "url already exists"
)

// filterSetProperties searches for the particular filter list by url and sets
// the values of newList to it, updating afterwards if needed.  It returns true
// if the update was performed and the filtering engine restart is required.
func (d *DNSFilter) filterSetProperties(
	listURL string,
	newList FilterYAML,
	isAllowlist bool,
) (shouldRestart bool, err error) {
	d.conf.filtersMu.Lock()
	defer d.conf.filtersMu.Unlock()

	filters := d.conf.Filters
	if isAllowlist {
		filters = d.conf.WhitelistFilters
	}

	i := slices.IndexFunc(filters, func(flt FilterYAML) bool { return flt.URL == listURL })
	if i == -1 {
		return false, errFilterNotExist
	}

	flt := &filters[i]
	d.logger.DebugContext(
		context.TODO(),
		"updating filter",
		"name", newList.Name,
		"url", newList.URL,
		"enabled", newList.Enabled,
		"filter_url", flt.URL,
	)

	// Restore the whole entry on failure, including the checksum, so that a
	// failed update doesn't leave metadata that disagrees with the contents on
	// disk.
	defer func(old FilterYAML) {
		if err != nil {
			*flt = old
		}
	}(*flt)

	flt.Name = newList.Name

	urlChanged := flt.URL != newList.URL
	if urlChanged {
		if d.filterExistsLocked(newList.URL) {
			return false, errFilterExists
		}

		shouldRestart = true

		flt.URL = newList.URL
		flt.LastUpdated = time.Time{}
		flt.unload()
	}

	if flt.Enabled != newList.Enabled {
		flt.Enabled = newList.Enabled
		shouldRestart = true
	}

	if d.shouldUnload(flt, isAllowlist, urlChanged) {
		// TODO(e.burkov):  The contents of a rule list that stays out of the
		// engine are not validated, which makes it possible to keep a bad rules
		// source.  Consider changing this behavior to be stricter.
		flt.unload()

		return shouldRestart, err
	}

	if !shouldRestart {
		return false, nil
	}

	return d.update(flt, urlChanged)
}

// filterExists returns true if a filter with the same url exists in d.  It's
// safe for concurrent use.
func (d *DNSFilter) filterExists(url string) (ok bool) {
	d.conf.filtersMu.RLock()
	defer d.conf.filtersMu.RUnlock()

	r := d.filterExistsLocked(url)

	return r
}

// filterExistsLocked returns true if d contains the filter with the same url.
// d.filtersMu is expected to be locked.
func (d *DNSFilter) filterExistsLocked(url string) (ok bool) {
	for _, f := range d.conf.Filters {
		if f.URL == url {
			return true
		}
	}

	for _, f := range d.conf.WhitelistFilters {
		if f.URL == url {
			return true
		}
	}

	return false
}

// Add a filter
// Return FALSE if a filter with this URL exists
func (d *DNSFilter) filterAdd(flt FilterYAML) (err error) {
	// Defer annotating to unlock sooner.
	defer func() { err = errors.Annotate(err, "adding filter: %w") }()

	d.conf.filtersMu.Lock()
	defer d.conf.filtersMu.Unlock()

	// Check for duplicates.
	if d.filterExistsLocked(flt.URL) {
		return errFilterExists
	}

	if flt.white {
		d.conf.WhitelistFilters = append(d.conf.WhitelistFilters, flt)
	} else {
		d.conf.Filters = append(d.conf.Filters, flt)
	}

	return nil
}

// loadFilters loads filters from the disk and assigns a new ID to any filter
// that has a zero one.  A disabled filter is only loaded if clientIDs contains
// it, since a client may use a filter that is disabled globally.
func (d *DNSFilter) loadFilters(
	ctx context.Context,
	array []FilterYAML,
	clientIDs map[rules.ListID]bool,
) {
	for i := range array {
		filter := &array[i] // otherwise we're operating on a copy
		if filter.ID == 0 {
			newID := d.idGen.next()
			d.logger.WarnContext(ctx, "filter has no id", "idx", i, "new_id", newID)

			filter.ID = newID
		}

		if !filter.Enabled && !clientIDs[filter.ID] {
			continue
		}

		err := d.load(ctx, filter)
		if err != nil {
			d.logger.ErrorContext(ctx, "loading filter", "id", filter.ID, slogutil.KeyError, err)
		}
	}
}

func deduplicateFilters(filters []FilterYAML) (deduplicated []FilterYAML) {
	urls := container.NewMapSet[string]()
	lastIdx := 0

	for _, filter := range filters {
		if !urls.Has(filter.URL) {
			urls.Add(filter.URL)
			filters[lastIdx] = filter
			lastIdx++
		}
	}

	return filters[:lastIdx]
}

// tryRefreshFilters is like [refreshFilters], but backs down if the update is
// already going on.
//
// TODO(e.burkov):  Get rid of the concurrency pattern which requires the
// [sync.Mutex.TryLock].
func (d *DNSFilter) tryRefreshFilters(block, allow, force bool) (updated int, isNetworkErr, ok bool) {
	if ok = d.refreshLock.TryLock(); !ok {
		return 0, false, false
	}
	defer d.refreshLock.Unlock()

	updated, isNetworkErr = d.refreshFiltersIntl(block, allow, force)

	return updated, isNetworkErr, ok
}

// listsToUpdate returns the slice of filter lists that could be updated.  A
// disabled filter list is only updated if clientIDs contains it.
func (d *DNSFilter) listsToUpdate(
	filters *[]FilterYAML,
	clientIDs map[rules.ListID]bool,
	force bool,
) (toUpd []FilterYAML) {
	now := time.Now()

	d.conf.filtersMu.RLock()
	defer d.conf.filtersMu.RUnlock()

	for i := range *filters {
		flt := &(*filters)[i] // otherwise we will be operating on a copy

		if !flt.Enabled && !clientIDs[flt.ID] {
			continue
		}

		if !force {
			exp := flt.LastUpdated.Add(time.Duration(d.conf.FiltersUpdateIntervalHours) * time.Hour)
			if now.Before(exp) {
				continue
			}
		}

		toUpd = append(toUpd, FilterYAML{
			Filter: Filter{
				ID: flt.ID,
			},
			URL:      flt.URL,
			Name:     flt.Name,
			checksum: flt.checksum,
		})
	}

	return toUpd
}

// refreshFiltersArray updates the filters array and returns the number of
// filters that have been refreshed.  updateFlags is true if filter data has
// changed.
func (d *DNSFilter) refreshFiltersArray(
	ctx context.Context,
	filters *[]FilterYAML,
	clientIDs map[rules.ListID]bool,
	force bool,
) (updateCount int, updateFilters []FilterYAML, updateFlags []bool, isNetErr bool) {
	updateFilters = d.listsToUpdate(filters, clientIDs, force)
	if len(updateFilters) == 0 {
		return 0, nil, nil, false
	}

	failNum, updateFlags := d.updateFilterList(ctx, updateFilters)
	if failNum == len(updateFilters) {
		return 0, nil, nil, true
	}

	d.conf.filtersMu.Lock()
	defer d.conf.filtersMu.Unlock()

	updateCount = d.syncUpdatedFilters(ctx, filters, updateFilters, updateFlags)

	return updateCount, updateFilters, updateFlags, false
}

// updateFilterList updates each filter in updateFilters and returns the number
// of failures and the updateFlags slice aligned with updateFilters indicating
// whether each filter's data changed.
func (d *DNSFilter) updateFilterList(
	ctx context.Context,
	updateFilters []FilterYAML,
) (failNum int, updateFlags []bool) {
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated, err := d.update(uf, false)
		updateFlags = append(updateFlags, updated)
		if err != nil {
			failNum++
			d.logger.ErrorContext(ctx, "updating filter", "url", uf.URL, slogutil.KeyError, err)
		}
	}

	return failNum, updateFlags
}

// syncUpdatedFilters syncs updated filters back to the original filters slice
// and returns the updateCount.  filters must not be nil.  updateFlags must
// align with updateFilters.  d.conf.filtersMu must be locked.
func (d *DNSFilter) syncUpdatedFilters(
	ctx context.Context,
	filters *[]FilterYAML,
	updateFilters []FilterYAML,
	updateFlags []bool,
) (updateCount int) {
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated := updateFlags[i]

		for k := range *filters {
			f := &(*filters)[k]
			if f.ID != uf.ID || f.URL != uf.URL {
				continue
			}

			f.LastUpdated = uf.LastUpdated
			if !updated {
				continue
			}

			d.logger.InfoContext(
				ctx,
				"updated filter",
				"id", f.ID,
				"rules_count", uf.RulesCount,
				"prev_rules_count", f.RulesCount,
			)

			f.Name = uf.Name
			f.RulesCount = uf.RulesCount
			f.checksum = uf.checksum
			updateCount++
		}
	}

	return updateCount
}

// refreshFiltersIntl checks filters and updates them if necessary.  If force is
// true, it ignores the filter.LastUpdated field value.
//
// Algorithm:
//
//  1. Get the list of filters to be updated.  For each filter, run the download
//     and checksum check operation.  Store downloaded data in a temporary file
//     inside data/filters directory
//
//  2. For each filter, if filter data hasn't changed, just set new update time
//     on file.  Otherwise, rename the temporary file (<temp> -> 1.txt).  Note
//     that this method works only on Unix systems.  On Windows, don't pass
//     files to filtering, pass the whole data.
//
// refreshFiltersIntl returns the number of updated filters.  It also returns
// true if there was a network error and nothing could be updated.
//
// TODO(a.garipov, e.burkov): What the hell?
func (d *DNSFilter) refreshFiltersIntl(block, allow, force bool) (int, bool) {
	ctx := context.TODO()

	updNum := 0
	d.logger.DebugContext(ctx, "starting update")
	defer func() {
		d.logger.DebugContext(ctx, "finished update", "updated", updNum)
	}()

	var lists []FilterYAML
	var toUpd []bool
	isNetErr := false

	blockIDs, allowIDs := d.clientFilterListIDs()
	if block {
		updNum, lists, toUpd, isNetErr = d.refreshFiltersArray(ctx, &d.conf.Filters, blockIDs, force)
	}
	if allow {
		updNumAl, listsAl, toUpdAl, isNetErrAl := d.refreshFiltersArray(
			ctx,
			&d.conf.WhitelistFilters,
			allowIDs,
			force,
		)

		updNum += updNumAl
		lists = append(lists, listsAl...)
		toUpd = append(toUpd, toUpdAl...)
		isNetErr = isNetErr || isNetErrAl
	}
	if isNetErr {
		return 0, true
	}

	if updNum == 0 {
		return 0, false
	}

	d.EnableFilters(false)

	for i := range lists {
		if toUpd[i] {
			removeOldFilterFile(ctx, d.logger, lists[i].Path(d.conf.DataDir))
		}
	}

	return updNum, false
}

// removeOldFilterFile deletes the old filter file and logs any error at the
// appropriate level.  l must not be nil.
func removeOldFilterFile(ctx context.Context, l *slog.Logger, fltPath string) {
	err := os.Remove(fltPath + ".old")
	if err == nil {
		return
	}

	lvl := slog.LevelWarn
	if errors.Is(err, os.ErrNotExist) {
		lvl = slog.LevelDebug
	}

	l.Log(ctx, lvl, "removing old filter", "path", fltPath, slogutil.KeyError, err)
}

// update refreshes filter's content and a/mtimes of it's file.
func (d *DNSFilter) update(filter *FilterYAML, replace bool) (b bool, err error) {
	ctx := context.TODO()

	b, err = d.updateIntl(ctx, filter, replace)
	filter.LastUpdated = time.Now()
	if !b {
		chErr := os.Chtimes(
			filter.Path(d.conf.DataDir),
			filter.LastUpdated,
			filter.LastUpdated,
		)
		if chErr != nil {
			d.logger.ErrorContext(ctx, "changing last modified time", slogutil.KeyError, chErr)
		}
	}

	return b, err
}

// updateIntl updates the flt rewriting it's actual file.  It returns true if
// the actual update has been performed.  When replace is true, the contents are
// saved even if they parse to the same checksum.  flt must not be nil.
func (d *DNSFilter) updateIntl(
	ctx context.Context,
	flt *FilterYAML,
	replace bool,
) (ok bool, err error) {
	d.logger.DebugContext(ctx, "downloading update for filter", "id", flt.ID, "url", flt.URL)

	var res *rulelist.ParseResult

	tmpFile, err := aghrenameio.NewPendingFile(flt.Path(d.conf.DataDir), aghos.DefaultPermFile)
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return false, err
	}
	defer func() { err = d.finalizeUpdate(ctx, tmpFile, flt, res, err, ok) }()

	if filepath.IsAbs(flt.URL) {
		// Initialise this variable to avoid any confusion.
		path := flt.URL

		res, err = d.readFromFile(tmpFile, path)
	} else {
		res, err = d.readFromHTTP(tmpFile, flt.URL)
	}

	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return false, err
	}

	return replace || res.Checksum != flt.checksum, nil
}

// readFromHTTP reads filter data from urlStr via HTTP and parses it into the
// tmpFile file.  tmpFile must not be nil.  urlStr must be a valid URL.
func (d *DNSFilter) readFromHTTP(
	tmpFile aghrenameio.PendingFile,
	urlStr string,
) (res *rulelist.ParseResult, err error) {
	resp, err := d.conf.HTTPClient.Get(urlStr)
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return nil, err
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status code %d, want %d", resp.StatusCode, http.StatusOK)
	}

	bufPtr := d.bufPool.Get()
	defer d.bufPool.Put(bufPtr)

	p := rulelist.NewParser()
	httpBody := ioutil.LimitReader(resp.Body, d.conf.MaxHTTPSize.Bytes())

	return p.Parse(tmpFile, httpBody, *bufPtr)
}

// readFromFile reads filter data from a file located at path and parses it into
// the tmpFile file.  tmpFile must not be nil.  path must be a valid filepath.
func (d *DNSFilter) readFromFile(
	tmpFile aghrenameio.PendingFile,
	path string,
) (res *rulelist.ParseResult, err error) {
	path = filepath.Clean(path)

	if !pathMatchesAny(d.safeFSPatterns, path) {
		return nil, fmt.Errorf("path %q does not match safe patterns", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, file.Close()) }()

	bufPtr := d.bufPool.Get()
	defer d.bufPool.Put(bufPtr)

	p := rulelist.NewParser()

	return p.Parse(tmpFile, file, *bufPtr)
}

// finalizeUpdate closes and gets rid of temporary file f with filter's content
// according to updated.  It also saves new values of flt's name, rules number
// and checksum if succeeded.
func (d *DNSFilter) finalizeUpdate(
	ctx context.Context,
	file aghrenameio.PendingFile,
	flt *FilterYAML,
	res *rulelist.ParseResult,
	returned error,
	updated bool,
) (err error) {
	id := flt.ID
	if !updated {
		if returned == nil {
			d.logger.DebugContext(ctx, "skipping filter with no changes", "id", id, "url", flt.URL)
		}

		return errors.WithDeferred(returned, file.Cleanup())
	}

	d.logger.InfoContext(ctx, "saving contents", "id", id, "path", flt.Path(d.conf.DataDir))

	err = file.CloseReplace()
	if err != nil {
		return fmt.Errorf("finalizing update: %w", err)
	}

	rulesCount := res.RulesCount
	d.logger.InfoContext(
		ctx,
		"filter updated",
		"id", id,
		"bytes_written", res.BytesWritten,
		"rules_count", rulesCount,
	)

	flt.ensureName(res.Title)
	flt.checksum = res.Checksum
	flt.RulesCount = rulesCount

	return nil
}

// loads filter contents from the file in dataDir
func (d *DNSFilter) load(ctx context.Context, flt *FilterYAML) (err error) {
	fileName := flt.Path(d.conf.DataDir)

	d.logger.DebugContext(ctx, "loading filter", "id", flt.ID, "path", fileName)

	// #nosec G304 -- Assume that fileName is always within DataDir.
	file, err := os.Open(fileName)
	if errors.Is(err, os.ErrNotExist) {
		// Do nothing, file doesn't exist.
		return nil
	} else if err != nil {
		return fmt.Errorf("opening filter file: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, file.Close()) }()

	st, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting filter file stat: %w", err)
	}

	d.logger.DebugContext(ctx, "filter file", "id", flt.ID, "path", fileName, "len", st.Size())

	bufPtr := d.bufPool.Get()
	defer d.bufPool.Put(bufPtr)

	p := rulelist.NewParser()
	res, err := p.Parse(io.Discard, file, *bufPtr)
	if err != nil {
		return fmt.Errorf("parsing filter file: %w", err)
	}

	flt.ensureName(res.Title)
	flt.RulesCount, flt.checksum, flt.LastUpdated = res.RulesCount, res.Checksum, st.ModTime()

	return nil
}

// appendEnabledFilters appends to orig the filters that must be put into the
// engine, that is the enabled ones and those that clientIDs contains, and
// returns the set of enabled IDs among them.  enabledIDs is nil if every
// appended filter is enabled, in which case matching needs no ID filtering.
func appendEnabledFilters(
	orig []Filter,
	flts []FilterYAML,
	clientIDs map[rules.ListID]bool,
	dataDir string,
) (res []Filter, enabledIDs map[rules.ListID]bool) {
	res = orig
	ids := make(map[rules.ListID]bool, len(flts))
	hasDisabled := false

	for _, flt := range flts {
		if flt.Enabled {
			ids[flt.ID] = true
		} else if clientIDs[flt.ID] {
			hasDisabled = true
		} else {
			continue
		}

		res = append(res, Filter{
			ID:       flt.ID,
			FilePath: flt.Path(dataDir),
		})
	}

	if !hasDisabled {
		return res, nil
	}

	return res, ids
}

// shouldUnload reports whether flt must be kept out of the engine, that is when
// it is disabled and no client uses it.  A filter with a new URL is always
// fetched, since the file on disk is named after the ID and would otherwise keep
// the rules of the previous source.
func (d *DNSFilter) shouldUnload(flt *FilterYAML, isAllowlist, urlChanged bool) (ok bool) {
	return !urlChanged && !flt.Enabled && !d.clientListIDs(isAllowlist)[flt.ID]
}

// clientListIDs is like [DNSFilter.clientFilterListIDs], but returns the IDs of
// either the allowing or the blocking filter lists.
func (d *DNSFilter) clientListIDs(isAllowlist bool) (ids map[rules.ListID]bool) {
	blockIDs, allowIDs := d.clientFilterListIDs()
	if isAllowlist {
		return allowIDs
	}

	return blockIDs
}

// clientFilterListIDs returns the filter list IDs used by clients that have
// their own filter lists, or nil sets if no client does.
func (d *DNSFilter) clientFilterListIDs() (blockIDs, allowIDs map[rules.ListID]bool) {
	if d.conf.ClientFilterListIDs == nil {
		return nil, nil
	}

	return d.conf.ClientFilterListIDs()
}

// needRefresh returns the disabled filters of flts that next references and the
// engine cannot enforce yet, that is the ones no client referenced before and
// the ones whose cache is missing.  A cache can be absent because the list was
// never downloadable, in which case being referenced already is not enough.
// d.conf.filtersMu is expected to be locked.
func (d *DNSFilter) needRefresh(
	flts []FilterYAML,
	prev, next map[rules.ListID]bool,
) (res []*FilterYAML) {
	for i := range flts {
		flt := &flts[i] // otherwise we're operating on a copy
		if flt.Enabled || !next[flt.ID] {
			continue
		}

		if !prev[flt.ID] {
			res = append(res, flt)

			continue
		}

		if _, err := os.Stat(flt.Path(d.conf.DataDir)); err != nil {
			res = append(res, flt)
		}
	}

	return res
}

// refreshReferenced downloads flts, so that a list a client has just picked has
// contents to match against even when its cache is absent or stale.  A refresh
// that fails while a cached copy exists is only logged, since that copy is
// enough to enforce the policy.  d.conf.filtersMu is expected to be locked for
// writing.
func (d *DNSFilter) refreshReferenced(ctx context.Context, flts []*FilterYAML) (errs []error) {
	for _, flt := range flts {
		_, err := d.update(flt, true)
		if err == nil {
			continue
		}

		if _, statErr := os.Stat(flt.Path(d.conf.DataDir)); statErr != nil {
			errs = append(errs, fmt.Errorf("filter %d: %w", flt.ID, err))

			continue
		}

		d.logger.WarnContext(
			ctx,
			"refreshing client filter list, keeping cached copy",
			"id", flt.ID,
			slogutil.KeyError, err,
		)
	}

	return errs
}

// loadedListIDs returns the rule lists the engines currently hold.
func (d *DNSFilter) loadedListIDs() (ids map[rules.ListID]bool) {
	d.engineLock.RLock()
	defer d.engineLock.RUnlock()

	return unionListIDs(d.loadedBlockIDs, d.loadedAllowIDs)
}

// unionListIDs returns the union of a and b.  It doesn't modify either.
func unionListIDs(a, b map[rules.ListID]bool) (res map[rules.ListID]bool) {
	res = make(map[rules.ListID]bool, len(a)+len(b))
	for id := range a {
		res[id] = true
	}

	for id := range b {
		res[id] = true
	}

	return res
}

// ErrUnknownListID is returned when a client references a filter list that is
// not configured, or one of the wrong kind.
const ErrUnknownListID errors.Error = "unknown filter list id"

// validateListIDs returns an error if ids names a list absent from flts.  A
// client whose policy names a list the engine cannot hold would match neither
// that list nor the global ones, so such a request is rejected instead.
// d.conf.filtersMu is expected to be locked.
func validateListIDs(flts []FilterYAML, ids map[rules.ListID]bool, kind string) (err error) {
	var errs []error
	for id := range ids {
		if id == rulelist.IDCustom {
			continue
		}

		if !slices.ContainsFunc(flts, func(f FilterYAML) (ok bool) { return f.ID == id }) {
			errs = append(errs, fmt.Errorf("%s %d: %w", kind, id, ErrUnknownListID))
		}
	}

	return errors.Join(errs...)
}

// SetClientFilterLists puts the filter lists that blockIDs and allowIDs
// reference into the engine, downloading the disabled ones that no client used
// before.  It does nothing when the engine already holds them all.
//
// It must be called before a client configuration that relies on those lists
// becomes observable, since a policy naming a list the engine doesn't hold
// matches neither that list nor the global ones.  Lists that no client uses
// anymore are pruned by a later [DNSFilter.EnableFilters].
func (d *DNSFilter) SetClientFilterLists(
	ctx context.Context,
	blockIDs, allowIDs map[rules.ListID]bool,
) (err error) {
	d.conf.filtersMu.Lock()
	defer d.conf.filtersMu.Unlock()

	err = errors.Join(
		validateListIDs(d.conf.Filters, blockIDs, "blocking filter list"),
		validateListIDs(d.conf.WhitelistFilters, allowIDs, "allowing filter list"),
	)
	if err != nil {
		return err
	}

	prevBlock, prevAllow := d.clientFilterListIDs()

	newBlock := d.needRefresh(d.conf.Filters, prevBlock, blockIDs)
	newAllow := d.needRefresh(d.conf.WhitelistFilters, prevAllow, allowIDs)
	if len(newBlock) == 0 && len(newAllow) == 0 {
		return nil
	}

	errs := d.refreshReferenced(ctx, newBlock)
	errs = append(errs, d.refreshReferenced(ctx, newAllow)...)
	if err = errors.Join(errs...); err != nil {
		return fmt.Errorf("refreshing client filter lists: %w", err)
	}

	// Require the lists the engines hold today plus the ones this policy names,
	// so that a file that vanished cannot silently drop a live list, while a list
	// that was already absent stays tolerated.
	required := unionListIDs(d.loadedListIDs(), unionListIDs(blockIDs, allowIDs))

	// Keep the lists that clients use today as well, so that the rebuild never
	// drops a policy that is still published.
	err = d.enableFiltersLocked(
		ctx,
		unionListIDs(prevBlock, blockIDs),
		unionListIDs(prevAllow, allowIDs),
		required,
		false,
	)
	if err != nil {
		return fmt.Errorf("rebuilding with client filter lists: %w", err)
	}

	return nil
}

// EnableFilters enables filters.
func (d *DNSFilter) EnableFilters(async bool) {
	d.conf.filtersMu.RLock()
	defer d.conf.filtersMu.RUnlock()

	ctx := context.TODO()
	blockIDs, allowIDs := d.clientFilterListIDs()

	err := d.enableFiltersLocked(ctx, blockIDs, allowIDs, nil, async)
	if err != nil {
		d.logger.ErrorContext(ctx, "enabling filters", slogutil.KeyError, err)
	}
}

// enableFiltersLocked enables filters under the conf.filtersMu lock.  The
// clientIDs sets tell which disabled lists must still be put into the engine
// because a client uses them.
func (d *DNSFilter) enableFiltersLocked(
	ctx context.Context,
	blockClientIDs, allowClientIDs map[rules.ListID]bool,
	requiredIDs map[rules.ListID]bool,
	async bool,
) (err error) {
	filters := make([]Filter, 1, len(d.conf.Filters)+len(d.conf.WhitelistFilters)+1)
	filters[0] = Filter{
		ID:   rulelist.IDCustom,
		Data: []byte(strings.Join(d.conf.UserRules, "\n")),
	}

	filters, enabledBlockIDs := appendEnabledFilters(
		filters,
		d.conf.Filters,
		blockClientIDs,
		d.conf.DataDir,
	)
	allowFilters, enabledAllowIDs := appendEnabledFilters(
		nil,
		d.conf.WhitelistFilters,
		allowClientIDs,
		d.conf.DataDir,
	)

	err = d.setFilters(ctx, &filtersInitializerParams{
		enabledBlockIDs: enabledBlockIDs,
		enabledAllowIDs: enabledAllowIDs,
		allowFilters:    allowFilters,
		blockFilters:    filters,
		requiredIDs:     requiredIDs,
	}, async)

	d.SetEnabled(d.conf.FilteringEnabled)

	return err
}

// ApplyAdditionalFiltering enhances the provided filtering settings with
// blocked services and client-specific configurations.
func (d *DNSFilter) ApplyAdditionalFiltering(cliAddr netip.Addr, clientID string, setts *Settings) {
	setts.ClientIP = cliAddr

	d.ApplyBlockedServices(setts)
	d.applyClientFiltering(clientID, cliAddr, setts)
	if setts.BlockedServices != nil {
		// TODO(e.burkov):  Get rid of this crutch.
		setts.ServicesRules = nil
		svcs := setts.BlockedServices.IDs
		if !setts.BlockedServices.Schedule.Contains(time.Now()) {
			d.ApplyBlockedServicesList(setts, svcs)
		}
	}
}
