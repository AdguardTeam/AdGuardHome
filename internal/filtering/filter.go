package filtering

import (
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// FilterYAML respresents a filter list in the configuration file.
//
// TODO(e.burkov):  Investigate if the field oredering is important.
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

	defer func(oldURL, oldName string, oldEnabled bool, oldUpdated time.Time) {
		if err != nil {
			filt.URL = oldURL
			filt.Name = oldName
			filt.Enabled = oldEnabled
			filt.LastUpdated = oldUpdated
		}
	}(filt.URL, filt.Name, filt.Enabled, filt.LastUpdated)

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
		// kick in when the filter is enabled.  Consider making changing this
		// behavior to be stricter.
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
func (d *DNSFilter) filterAdd(flt FilterYAML) bool {
	d.filtersMu.Lock()
	defer d.filtersMu.Unlock()

	// Check for duplicates
	if d.filterExistsLocked(flt.URL) {
		return false
	}

	if flt.white {
		d.WhitelistFilters = append(d.WhitelistFilters, flt)
	} else {
		d.Filters = append(d.Filters, flt)
	}
	return true
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
			log.Error("Couldn't load filter %d contents due to %s", filter.ID, err)
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
// sync.Mutex.TryLock.
func (d *DNSFilter) tryRefreshFilters(block, allow, force bool) (updated int, isNetworkErr, ok bool) {
	if ok = d.refreshLock.TryLock(); !ok {
		return 0, false, ok
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
			log.Printf("Failed to update filter %s: %s\n", uf.URL, err)
			continue
		}
	}

	if nfail == len(updateFilters) {
		return 0, nil, nil, true
	}

	updateCount := 0
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated := updateFlags[i]

		d.filtersMu.Lock()
		for k := range *filters {
			f := &(*filters)[k]
			if f.ID != uf.ID || f.URL != uf.URL {
				continue
			}
			f.LastUpdated = uf.LastUpdated
			if !updated {
				continue
			}

			log.Info("Updated filter #%d.  Rules: %d -> %d",
				f.ID, f.RulesCount, uf.RulesCount)
			f.Name = uf.Name
			f.RulesCount = uf.RulesCount
			f.checksum = uf.checksum
			updateCount++
		}
		d.filtersMu.Unlock()
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
	log.Debug("filtering: updating...")

	updNum := 0
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
			_ = os.Remove(uf.Path(d.DataDir) + ".old")
		}
	}

	log.Debug("filtering: update finished")

	return updNum, false
}

// Allows printable UTF-8 text with CR, LF, TAB characters
func isPrintableText(data []byte, len int) bool {
	for i := 0; i < len; i++ {
		c := data[i]
		if (c >= ' ' && c != 0x7f) || c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		return false
	}
	return true
}

// A helper function that parses filter contents and returns a number of rules and a filter name (if there's any)
func (d *DNSFilter) parseFilterContents(file io.Reader) (int, uint32, string) {
	rulesCount := 0
	name := ""
	seenTitle := false
	r := bufio.NewReader(file)
	checksum := uint32(0)

	for {
		line, err := r.ReadString('\n')
		checksum = crc32.Update(checksum, crc32.IEEETable, []byte(line))

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			//
		} else if line[0] == '!' {
			m := d.filterTitleRegexp.FindAllStringSubmatch(line, -1)
			if len(m) > 0 && len(m[0]) >= 2 && !seenTitle {
				name = m[0][1]
				seenTitle = true
			}

		} else if line[0] == '#' {
			//
		} else {
			rulesCount++
		}

		if err != nil {
			break
		}
	}

	return rulesCount, checksum, name
}

// Perform upgrade on a filter and update LastUpdated value
func (d *DNSFilter) update(filter *FilterYAML) (bool, error) {
	b, err := d.updateIntl(filter)
	filter.LastUpdated = time.Now()
	if !b {
		e := os.Chtimes(filter.Path(d.DataDir), filter.LastUpdated, filter.LastUpdated)
		if e != nil {
			log.Error("os.Chtimes(): %v", e)
		}
	}
	return b, err
}

func (d *DNSFilter) read(reader io.Reader, tmpFile *os.File, filter *FilterYAML) (int, error) {
	htmlTest := true
	firstChunk := make([]byte, 4*1024)
	firstChunkLen := 0
	buf := make([]byte, 64*1024)
	total := 0
	for {
		n, err := reader.Read(buf)
		total += n

		if htmlTest {
			num := len(firstChunk) - firstChunkLen
			if n < num {
				num = n
			}
			copied := copy(firstChunk[firstChunkLen:], buf[:num])
			firstChunkLen += copied

			if firstChunkLen == len(firstChunk) || err == io.EOF {
				if !isPrintableText(firstChunk, firstChunkLen) {
					return total, fmt.Errorf("data contains non-printable characters")
				}

				s := strings.ToLower(string(firstChunk))
				if strings.Contains(s, "<html") || strings.Contains(s, "<!doctype") {
					return total, fmt.Errorf("data is HTML, not plain text")
				}

				htmlTest = false
				firstChunk = nil
			}
		}

		_, err2 := tmpFile.Write(buf[:n])
		if err2 != nil {
			return total, err2
		}

		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			log.Printf("Couldn't fetch filter contents from URL %s, skipping: %s", filter.URL, err)
			return total, err
		}
	}
}

// finalizeUpdate closes and gets rid of temporary file f with filter's content
// according to updated.  It also saves new values of flt's name, rules number
// and checksum if sucсeeded.
func (d *DNSFilter) finalizeUpdate(
	file *os.File,
	flt *FilterYAML,
	updated bool,
	name string,
	rnum int,
	cs uint32,
) (err error) {
	tmpFileName := file.Name()

	// Close the file before renaming it because it's required on Windows.
	//
	// See https://github.com/adguardTeam/adGuardHome/issues/1553.
	if err = file.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}

	if !updated {
		log.Tracef("filter #%d from %s has no changes, skip", flt.ID, flt.URL)

		return os.Remove(tmpFileName)
	}

	log.Printf("saving filter %d contents to: %s", flt.ID, flt.Path(d.DataDir))

	if err = os.Rename(tmpFileName, flt.Path(d.DataDir)); err != nil {
		return errors.WithDeferred(err, os.Remove(tmpFileName))
	}

	flt.Name = stringutil.Coalesce(flt.Name, name)
	flt.checksum = cs
	flt.RulesCount = rnum

	return nil
}

// processUpdate copies filter's content from src to dst and returns the name,
// rules number, and checksum for it.  It also returns the number of bytes read
// from src.
func (d *DNSFilter) processUpdate(
	src io.Reader,
	dst *os.File,
	flt *FilterYAML,
) (name string, rnum int, cs uint32, n int, err error) {
	if n, err = d.read(src, dst, flt); err != nil {
		return "", 0, 0, 0, err
	}

	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return "", 0, 0, 0, err
	}

	rnum, cs, name = d.parseFilterContents(dst)

	return name, rnum, cs, n, nil
}

// updateIntl updates the flt rewriting it's actual file.  It returns true if
// the actual update has been performed.
func (d *DNSFilter) updateIntl(flt *FilterYAML) (ok bool, err error) {
	log.Tracef("downloading update for filter %d from %s", flt.ID, flt.URL)

	var name string
	var rnum, n int
	var cs uint32

	var tmpFile *os.File
	tmpFile, err = os.CreateTemp(filepath.Join(d.DataDir, filterDir), "")
	if err != nil {
		return false, err
	}
	defer func() {
		err = errors.WithDeferred(err, d.finalizeUpdate(tmpFile, flt, ok, name, rnum, cs))
		ok = ok && err == nil
		if ok {
			log.Printf("updated filter %d: %d bytes, %d rules", flt.ID, n, rnum)
		}
	}()

	// Change the default 0o600 permission to something more acceptable by
	// end users.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3198.
	if err = tmpFile.Chmod(0o644); err != nil {
		return false, fmt.Errorf("changing file mode: %w", err)
	}

	var r io.Reader
	if filepath.IsAbs(flt.URL) {
		var file io.ReadCloser
		file, err = os.Open(flt.URL)
		if err != nil {
			return false, fmt.Errorf("open file: %w", err)
		}
		defer func() { err = errors.WithDeferred(err, file.Close()) }()

		r = file
	} else {
		var resp *http.Response
		resp, err = d.HTTPClient.Get(flt.URL)
		if err != nil {
			log.Printf("requesting filter from %s, skip: %s", flt.URL, err)

			return false, err
		}
		defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

		if resp.StatusCode != http.StatusOK {
			log.Printf("got status code %d from %s, skip", resp.StatusCode, flt.URL)

			return false, fmt.Errorf("got status code != 200: %d", resp.StatusCode)
		}

		r = resp.Body
	}

	name, rnum, cs, n, err = d.processUpdate(r, tmpFile, flt)

	return cs != flt.checksum, err
}

// loads filter contents from the file in dataDir
func (d *DNSFilter) load(filter *FilterYAML) (err error) {
	filterFilePath := filter.Path(d.DataDir)

	log.Tracef("filtering: loading filter %d from %s", filter.ID, filterFilePath)

	file, err := os.Open(filterFilePath)
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

	log.Tracef("filtering: File %s, id %d, length %d", filterFilePath, filter.ID, st.Size())

	rulesCount, checksum, _ := d.parseFilterContents(file)

	filter.RulesCount = rulesCount
	filter.checksum = checksum
	filter.LastUpdated = st.ModTime()

	return nil
}

func (d *DNSFilter) EnableFilters(async bool) {
	d.filtersMu.RLock()
	defer d.filtersMu.RUnlock()

	d.enableFiltersLocked(async)
}

func (d *DNSFilter) enableFiltersLocked(async bool) {
	filters := []Filter{{
		ID:   CustomListID,
		Data: []byte(strings.Join(d.UserRules, "\n")),
	}}

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

	if err := d.SetFilters(filters, allowFilters, async); err != nil {
		log.Debug("enabling filters: %s", err)
	}

	d.SetEnabled(d.FilteringEnabled)
}
