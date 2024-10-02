package filtering

import (
	"fmt"
	"io"
	"net/http"
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
	"github.com/AdguardTeam/golibs/log"
)

// filterDir is the subdirectory of a data directory to store downloaded
// filters.
const filterDir = "filters"

// FilterYAML represents a filter list in the configuration file.
//
// TODO(e.burkov):  Investigate if the field ordering is important.
type FilterYAML struct {
	Enabled     bool
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
		strconv.FormatInt(int64(filter.ID), 10)+".txt")
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
	log.Debug(
		"filtering: set name to %q, url to %s, enabled to %t for filter %s",
		newList.Name,
		newList.URL,
		newList.Enabled,
		flt.URL,
	)

	defer func(oldURL, oldName string, oldEnabled bool, oldUpdated time.Time, oldRulesCount int) {
		if err != nil {
			flt.URL = oldURL
			flt.Name = oldName
			flt.Enabled = oldEnabled
			flt.LastUpdated = oldUpdated
			flt.RulesCount = oldRulesCount
		}
	}(flt.URL, flt.Name, flt.Enabled, flt.LastUpdated, flt.RulesCount)

	flt.Name = newList.Name

	if flt.URL != newList.URL {
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

	if flt.Enabled {
		if shouldRestart {
			// Download the filter contents.
			shouldRestart, err = d.update(flt)
		}
	} else {
		// TODO(e.burkov):  The validation of the contents of the new URL is
		// currently skipped if the rule list is disabled.  This makes it
		// possible to set a bad rules source, but the validation should still
		// kick in when the filter is enabled.  Consider changing this behavior
		// to be stricter.
		flt.unload()
	}

	return shouldRestart, err
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

// Load filters from the disk
// And if any filter has zero ID, assign a new one
func (d *DNSFilter) loadFilters(array []FilterYAML) {
	for i := range array {
		filter := &array[i] // otherwise we're operating on a copy
		if filter.ID == 0 {
			newID := d.idGen.next()
			log.Info("filtering: warning: filter at index %d has no id; assigning to %d", i, newID)

			filter.ID = newID
		}

		if !filter.Enabled {
			// No need to load a filter that is not enabled
			continue
		}

		err := d.load(filter)
		if err != nil {
			log.Error("filtering: loading filter %d: %s", filter.ID, err)
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

// listsToUpdate returns the slice of filter lists that could be updated.
func (d *DNSFilter) listsToUpdate(filters *[]FilterYAML, force bool) (toUpd []FilterYAML) {
	now := time.Now()

	d.conf.filtersMu.RLock()
	defer d.conf.filtersMu.RUnlock()

	for i := range *filters {
		flt := &(*filters)[i] // otherwise we will be operating on a copy

		if !flt.Enabled {
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

func (d *DNSFilter) refreshFiltersArray(filters *[]FilterYAML, force bool) (int, []FilterYAML, []bool, bool) {
	var updateFlags []bool // 'true' if filter data has changed

	updateFilters := d.listsToUpdate(filters, force)
	if len(updateFilters) == 0 {
		return 0, nil, nil, false
	}

	failNum := 0
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated, err := d.update(uf)
		updateFlags = append(updateFlags, updated)
		if err != nil {
			failNum++
			log.Error("filtering: updating filter from url %q: %s\n", uf.URL, err)

			continue
		}
	}

	if failNum == len(updateFilters) {
		return 0, nil, nil, true
	}

	updateCount := 0

	d.conf.filtersMu.Lock()
	defer d.conf.filtersMu.Unlock()

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

			log.Info(
				"filtering: updated filter %d; rule count: %d (was %d)",
				f.ID,
				uf.RulesCount,
				f.RulesCount,
			)

			f.Name = uf.Name
			f.RulesCount = uf.RulesCount
			f.checksum = uf.checksum
			updateCount++
		}
	}

	return updateCount, updateFilters, updateFlags, false
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
	updNum := 0
	log.Debug("filtering: starting updating")
	defer func() { log.Debug("filtering: finished updating, %d updated", updNum) }()

	var lists []FilterYAML
	var toUpd []bool
	isNetErr := false

	if block {
		updNum, lists, toUpd, isNetErr = d.refreshFiltersArray(&d.conf.Filters, force)
	}
	if allow {
		updNumAl, listsAl, toUpdAl, isNetErrAl := d.refreshFiltersArray(&d.conf.WhitelistFilters, force)

		updNum += updNumAl
		lists = append(lists, listsAl...)
		toUpd = append(toUpd, toUpdAl...)
		isNetErr = isNetErr || isNetErrAl
	}
	if isNetErr {
		return 0, true
	}

	if updNum != 0 {
		d.EnableFilters(false)

		for i := range lists {
			uf := &lists[i]
			updated := toUpd[i]
			if !updated {
				continue
			}

			p := uf.Path(d.conf.DataDir)
			err := os.Remove(p + ".old")
			if err != nil {
				log.Debug("filtering: removing old filter file %q: %s", p, err)
			}
		}
	}

	return updNum, false
}

// update refreshes filter's content and a/mtimes of it's file.
func (d *DNSFilter) update(filter *FilterYAML) (b bool, err error) {
	b, err = d.updateIntl(filter)
	filter.LastUpdated = time.Now()
	if !b {
		chErr := os.Chtimes(
			filter.Path(d.conf.DataDir),
			filter.LastUpdated,
			filter.LastUpdated,
		)
		if chErr != nil {
			log.Error("filtering: os.Chtimes(): %s", chErr)
		}
	}

	return b, err
}

// updateIntl updates the flt rewriting it's actual file.  It returns true if
// the actual update has been performed.
func (d *DNSFilter) updateIntl(flt *FilterYAML) (ok bool, err error) {
	log.Debug("filtering: downloading update for filter %d from %q", flt.ID, flt.URL)

	var res *rulelist.ParseResult

	tmpFile, err := aghrenameio.NewPendingFile(flt.Path(d.conf.DataDir), aghos.DefaultPermFile)
	if err != nil {
		return false, err
	}
	defer func() { err = d.finalizeUpdate(tmpFile, flt, res, err, ok) }()

	r, err := d.reader(flt.URL)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return false, err
	}
	defer func() { err = errors.WithDeferred(err, r.Close()) }()

	bufPtr := d.bufPool.Get()
	defer d.bufPool.Put(bufPtr)

	p := rulelist.NewParser()
	res, err = p.Parse(tmpFile, r, *bufPtr)

	return res.Checksum != flt.checksum && err == nil, err
}

// finalizeUpdate closes and gets rid of temporary file f with filter's content
// according to updated.  It also saves new values of flt's name, rules number
// and checksum if succeeded.
func (d *DNSFilter) finalizeUpdate(
	file aghrenameio.PendingFile,
	flt *FilterYAML,
	res *rulelist.ParseResult,
	returned error,
	updated bool,
) (err error) {
	id := flt.ID
	if !updated {
		if returned == nil {
			log.Debug("filtering: filter %d from url %q has no changes, skipping", id, flt.URL)
		}

		return errors.WithDeferred(returned, file.Cleanup())
	}

	log.Info("filtering: saving contents of filter %d into %q", id, flt.Path(d.conf.DataDir))

	err = file.CloseReplace()
	if err != nil {
		return fmt.Errorf("finalizing update: %w", err)
	}

	rulesCount := res.RulesCount
	log.Info("filtering: updated filter %d: %d bytes, %d rules", id, res.BytesWritten, rulesCount)

	flt.ensureName(res.Title)
	flt.checksum = res.Checksum
	flt.RulesCount = rulesCount

	return nil
}

// reader returns an io.ReadCloser reading filtering-rule list data form either
// a file on the filesystem or the filter's HTTP URL.
func (d *DNSFilter) reader(fltURL string) (r io.ReadCloser, err error) {
	if !filepath.IsAbs(fltURL) {
		r, err = d.readerFromURL(fltURL)
		if err != nil {
			return nil, fmt.Errorf("reading from url: %w", err)
		}

		return r, nil
	}

	fltURL = filepath.Clean(fltURL)
	if !pathMatchesAny(d.safeFSPatterns, fltURL) {
		return nil, fmt.Errorf("path %q does not match safe patterns", fltURL)
	}

	r, err = os.Open(fltURL)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	return r, nil
}

// readerFromURL returns an io.ReadCloser reading filtering-rule list data form
// the filter's URL.
func (d *DNSFilter) readerFromURL(fltURL string) (r io.ReadCloser, err error) {
	resp, err := d.conf.HTTPClient.Get(fltURL)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status code %d, want %d", resp.StatusCode, http.StatusOK)
	}

	return resp.Body, nil
}

// loads filter contents from the file in dataDir
func (d *DNSFilter) load(flt *FilterYAML) (err error) {
	fileName := flt.Path(d.conf.DataDir)

	log.Debug("filtering: loading filter %d from %q", flt.ID, fileName)

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

	log.Debug("filtering: file %q, id %d, length %d", fileName, flt.ID, st.Size())

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

func (d *DNSFilter) EnableFilters(async bool) {
	d.conf.filtersMu.RLock()
	defer d.conf.filtersMu.RUnlock()

	d.enableFiltersLocked(async)
}

func (d *DNSFilter) enableFiltersLocked(async bool) {
	filters := make([]Filter, 1, len(d.conf.Filters)+len(d.conf.WhitelistFilters)+1)
	filters[0] = Filter{
		ID:   rulelist.URLFilterIDCustom,
		Data: []byte(strings.Join(d.conf.UserRules, "\n")),
	}

	for _, filter := range d.conf.Filters {
		if !filter.Enabled {
			continue
		}

		filters = append(filters, Filter{
			ID:       filter.ID,
			FilePath: filter.Path(d.conf.DataDir),
		})
	}

	var allowFilters []Filter
	for _, filter := range d.conf.WhitelistFilters {
		if !filter.Enabled {
			continue
		}

		allowFilters = append(allowFilters, Filter{
			ID:       filter.ID,
			FilePath: filter.Path(d.conf.DataDir),
		})
	}

	err := d.setFilters(filters, allowFilters, async)
	if err != nil {
		log.Error("filtering: enabling filters: %s", err)
	}

	d.SetEnabled(d.conf.FilteringEnabled)
}
