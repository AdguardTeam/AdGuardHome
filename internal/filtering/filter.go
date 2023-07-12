package filtering

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"golang.org/x/exp/slices"
)

// filterDir is the subdirectory of a data directory to store downloaded
// filters.
const filterDir = "filters"

// nextFilterID is a way to seed a unique ID generation.
//
// TODO(e.burkov):  Use more deterministic approach.
var nextFilterID = time.Now().Unix()

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
	return filepath.Join(dataDir, filterDir, strconv.FormatInt(filter.ID, 10)+".txt")
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
	d.filtersMu.Lock()
	defer d.filtersMu.Unlock()

	filters := d.Filters
	if isAllowlist {
		filters = d.WhitelistFilters
	}

	i := slices.IndexFunc(filters, func(filt FilterYAML) bool { return filt.URL == listURL })
	if i == -1 {
		return false, errFilterNotExist
	}

	filt := &filters[i]
	log.Debug(
		"filtering: set name to %q, url to %s, enabled to %t for filter %s",
		newList.Name,
		newList.URL,
		newList.Enabled,
		filt.URL,
	)

	defer func(oldURL, oldName string, oldEnabled bool, oldUpdated time.Time, oldRulesCount int) {
		if err != nil {
			filt.URL = oldURL
			filt.Name = oldName
			filt.Enabled = oldEnabled
			filt.LastUpdated = oldUpdated
			filt.RulesCount = oldRulesCount
		}
	}(filt.URL, filt.Name, filt.Enabled, filt.LastUpdated, filt.RulesCount)

	filt.Name = newList.Name

	if filt.URL != newList.URL {
		if d.filterExistsLocked(newList.URL) {
			return false, errFilterExists
		}

		shouldRestart = true

		filt.URL = newList.URL
		filt.LastUpdated = time.Time{}
		filt.unload()
	}

	if filt.Enabled != newList.Enabled {
		filt.Enabled = newList.Enabled
		shouldRestart = true
	}

	if filt.Enabled {
		if shouldRestart {
			// Download the filter contents.
			shouldRestart, err = d.update(filt)
		}
	} else {
		// TODO(e.burkov):  The validation of the contents of the new URL is
		// currently skipped if the rule list is disabled.  This makes it
		// possible to set a bad rules source, but the validation should still
		// kick in when the filter is enabled.  Consider changing this behavior
		// to be stricter.
		filt.unload()
	}

	return shouldRestart, err
}

// filterExists returns true if a filter with the same url exists in d.  It's
// safe for concurrent use.
func (d *DNSFilter) filterExists(url string) (ok bool) {
	d.filtersMu.RLock()
	defer d.filtersMu.RUnlock()

	r := d.filterExistsLocked(url)

	return r
}

// filterExistsLocked returns true if d contains the filter with the same url.
// d.filtersMu is expected to be locked.
func (d *DNSFilter) filterExistsLocked(url string) (ok bool) {
	for _, f := range d.Filters {
		if f.URL == url {
			return true
		}
	}

	for _, f := range d.WhitelistFilters {
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

	d.filtersMu.Lock()
	defer d.filtersMu.Unlock()

	// Check for duplicates.
	if d.filterExistsLocked(flt.URL) {
		return errFilterExists
	}

	if flt.white {
		d.WhitelistFilters = append(d.WhitelistFilters, flt)
	} else {
		d.Filters = append(d.Filters, flt)
	}

	return nil
}

// Load filters from the disk
// And if any filter has zero ID, assign a new one
func (d *DNSFilter) loadFilters(array []FilterYAML) {
	for i := range array {
		filter := &array[i] // otherwise we're operating on a copy
		if filter.ID == 0 {
			filter.ID = assignUniqueFilterID()
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
	urls := stringutil.NewSet()
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

// Set the next filter ID to max(filter.ID) + 1
func updateUniqueFilterID(filters []FilterYAML) {
	for _, filter := range filters {
		if nextFilterID < filter.ID {
			nextFilterID = filter.ID + 1
		}
	}
}

// TODO(e.burkov):  Improve this inexhaustible source of races.
func assignUniqueFilterID() int64 {
	value := nextFilterID
	nextFilterID++
	return value
}

// Sets up a timer that will be checking for filters updates periodically
func (d *DNSFilter) periodicallyRefreshFilters() {
	const maxInterval = 1 * 60 * 60
	intval := 5 // use a dynamically increasing time interval
	for {
		isNetErr, ok := false, false
		if d.FiltersUpdateIntervalHours != 0 {
			_, isNetErr, ok = d.tryRefreshFilters(true, true, false)
			if ok && !isNetErr {
				intval = maxInterval
			}
		}

		if isNetErr {
			intval *= 2
			if intval > maxInterval {
				intval = maxInterval
			}
		}

		time.Sleep(time.Duration(intval) * time.Second)
	}
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

	d.filtersMu.RLock()
	defer d.filtersMu.RUnlock()

	for i := range *filters {
		flt := &(*filters)[i] // otherwise we will be operating on a copy

		if !flt.Enabled {
			continue
		}

		if !force {
			exp := flt.LastUpdated.Add(time.Duration(d.FiltersUpdateIntervalHours) * time.Hour)
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

	nfail := 0
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated, err := d.update(uf)
		updateFlags = append(updateFlags, updated)
		if err != nil {
			nfail++
			log.Info("filtering: updating filter from url %q: %s\n", uf.URL, err)

			continue
		}
	}

	if nfail == len(updateFilters) {
		return 0, nil, nil, true
	}

	updateCount := 0

	d.filtersMu.Lock()
	defer d.filtersMu.Unlock()

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
		updNum, lists, toUpd, isNetErr = d.refreshFiltersArray(&d.Filters, force)
	}
	if allow {
		updNumAl, listsAl, toUpdAl, isNetErrAl := d.refreshFiltersArray(&d.WhitelistFilters, force)

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

			p := uf.Path(d.DataDir)
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
			filter.Path(d.DataDir),
			filter.LastUpdated,
			filter.LastUpdated,
		)
		if chErr != nil {
			log.Error("filtering: os.Chtimes(): %s", chErr)
		}
	}

	return b, err
}

// finalizeUpdate closes and gets rid of temporary file f with filter's content
// according to updated.  It also saves new values of flt's name, rules number
// and checksum if succeeded.
func (d *DNSFilter) finalizeUpdate(
	file *os.File,
	flt *FilterYAML,
	updated bool,
	res *rulelist.ParseResult,
) (err error) {
	tmpFileName := file.Name()

	// Close the file before renaming it because it's required on Windows.
	//
	// See https://github.com/adguardTeam/adGuardHome/issues/1553.
	err = file.Close()
	if err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}

	if !updated {
		log.Debug("filtering: filter %d from url %q has no changes, skipping", flt.ID, flt.URL)

		return os.Remove(tmpFileName)
	}

	fltPath := flt.Path(d.DataDir)

	log.Info("filtering: saving contents of filter %d into %q", flt.ID, fltPath)

	// Don't use renameio or maybe packages, since those will require loading
	// the whole filter content to the memory on Windows.
	err = os.Rename(tmpFileName, fltPath)
	if err != nil {
		return errors.WithDeferred(err, os.Remove(tmpFileName))
	}

	flt.Name = aghalg.Coalesce(flt.Name, res.Title)
	flt.checksum, flt.RulesCount = res.Checksum, res.RulesCount

	return nil
}

// updateIntl updates the flt rewriting it's actual file.  It returns true if
// the actual update has been performed.
func (d *DNSFilter) updateIntl(flt *FilterYAML) (ok bool, err error) {
	log.Debug("filtering: downloading update for filter %d from %q", flt.ID, flt.URL)

	var res *rulelist.ParseResult

	var tmpFile *os.File
	tmpFile, err = os.CreateTemp(filepath.Join(d.DataDir, filterDir), "")
	if err != nil {
		return false, err
	}
	defer func() {
		finErr := d.finalizeUpdate(tmpFile, flt, ok, res)
		if ok && finErr == nil {
			log.Info(
				"filtering: updated filter %d: %d bytes, %d rules",
				flt.ID,
				res.BytesWritten,
				res.RulesCount,
			)

			return
		}

		err = errors.WithDeferred(err, finErr)
	}()

	// Change the default 0o600 permission to something more acceptable by end
	// users.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3198.
	if err = tmpFile.Chmod(0o644); err != nil {
		return false, fmt.Errorf("changing file mode: %w", err)
	}

	var r io.Reader
	if !filepath.IsAbs(flt.URL) {
		var resp *http.Response
		resp, err = d.HTTPClient.Get(flt.URL)
		if err != nil {
			log.Info("filtering: requesting filter from %q: %s, skipping", flt.URL, err)

			return false, err
		}
		defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

		if resp.StatusCode != http.StatusOK {
			log.Info("filtering got status code %d from %q, skipping", resp.StatusCode, flt.URL)

			return false, fmt.Errorf("got status code %d, want %d", resp.StatusCode, http.StatusOK)
		}

		r = resp.Body
	} else {
		var f *os.File
		f, err = os.Open(flt.URL)
		if err != nil {
			return false, fmt.Errorf("open file: %w", err)
		}
		defer func() { err = errors.WithDeferred(err, f.Close()) }()

		r = f
	}

	bufPtr := d.bufPool.Get().(*[]byte)
	defer d.bufPool.Put(bufPtr)

	p := rulelist.NewParser()
	res, err = p.Parse(tmpFile, r, *bufPtr)

	return res.Checksum != flt.checksum && err == nil, err
}

// loads filter contents from the file in dataDir
func (d *DNSFilter) load(flt *FilterYAML) (err error) {
	fileName := flt.Path(d.DataDir)

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

	bufPtr := d.bufPool.Get().(*[]byte)
	defer d.bufPool.Put(bufPtr)

	p := rulelist.NewParser()
	res, err := p.Parse(io.Discard, file, *bufPtr)
	if err != nil {
		return fmt.Errorf("parsing filter file: %w", err)
	}

	flt.RulesCount, flt.checksum, flt.LastUpdated = res.RulesCount, res.Checksum, st.ModTime()

	return nil
}

func (d *DNSFilter) EnableFilters(async bool) {
	d.filtersMu.RLock()
	defer d.filtersMu.RUnlock()

	d.enableFiltersLocked(async)
}

func (d *DNSFilter) enableFiltersLocked(async bool) {
	filters := make([]Filter, 1, len(d.Filters)+len(d.WhitelistFilters)+1)
	filters[0] = Filter{
		ID:   CustomListID,
		Data: []byte(strings.Join(d.UserRules, "\n")),
	}

	for _, filter := range d.Filters {
		if !filter.Enabled {
			continue
		}

		filters = append(filters, Filter{
			ID:       filter.ID,
			FilePath: filter.Path(d.DataDir),
		})
	}

	var allowFilters []Filter
	for _, filter := range d.WhitelistFilters {
		if !filter.Enabled {
			continue
		}

		allowFilters = append(allowFilters, Filter{
			ID:       filter.ID,
			FilePath: filter.Path(d.DataDir),
		})
	}

	err := d.setFilters(filters, allowFilters, async)
	if err != nil {
		log.Error("filtering: enabling filters: %s", err)
	}

	d.SetEnabled(d.FilteringEnabled)
}
