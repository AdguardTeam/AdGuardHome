package home

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
)

var (
	nextFilterID      = time.Now().Unix() // semi-stable way to generate an unique ID
	filterTitleRegexp = regexp.MustCompile(`^! Title: +(.*)$`)
	refreshStatus     uint32 // 0:none; 1:in progress
	refreshLock       sync.Mutex
)

func initFiltering() {
	loadFilters(config.Filters)
	loadFilters(config.WhitelistFilters)
	deduplicateFilters()
	updateUniqueFilterID(config.Filters)
	updateUniqueFilterID(config.WhitelistFilters)
}

func startFiltering() {
	// Here we should start updating filters,
	//  but currently we can't wake up the periodic task to do so.
	// So for now we just start this periodic task from here.
	go periodicallyRefreshFilters()
}

func defaultFilters() []filter {
	return []filter{
		{Filter: dnsfilter.Filter{ID: 1}, Enabled: true, URL: "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt", Name: "AdGuard Simplified Domain Names filter"},
		{Filter: dnsfilter.Filter{ID: 2}, Enabled: false, URL: "https://adaway.org/hosts.txt", Name: "AdAway"},
		{Filter: dnsfilter.Filter{ID: 3}, Enabled: false, URL: "https://hosts-file.net/ad_servers.txt", Name: "hpHosts - Ad and Tracking servers only"},
		{Filter: dnsfilter.Filter{ID: 4}, Enabled: false, URL: "https://www.malwaredomainlist.com/hostslist/hosts.txt", Name: "MalwareDomainList.com Hosts List"},
	}
}

// field ordering is important -- yaml fields will mirror ordering from here
type filter struct {
	Enabled     bool
	URL         string
	Name        string    `yaml:"name"`
	RulesCount  int       `yaml:"-"`
	LastUpdated time.Time `yaml:"-"`
	checksum    uint32    // checksum of the file data
	white       bool

	dnsfilter.Filter `yaml:",inline"`
}

// Creates a helper object for working with the user rules
func userFilter() filter {
	f := filter{
		// User filter always has constant ID=0
		Enabled: true,
	}
	f.Filter.Data = []byte(strings.Join(config.UserRules, "\n"))
	return f
}

const (
	statusFound          = 1
	statusEnabledChanged = 2
	statusURLChanged     = 4
	statusURLExists      = 8
)

// Update properties for a filter specified by its URL
// Return status* flags.
func filterSetProperties(url string, newf filter, whitelist bool) int {
	r := 0
	config.Lock()
	defer config.Unlock()

	filters := &config.Filters
	if whitelist {
		filters = &config.WhitelistFilters
	}

	for i := range *filters {
		f := &(*filters)[i]
		if f.URL != url {
			continue
		}

		log.Debug("filter: set properties: %s: {%s %s %v}",
			f.URL, newf.Name, newf.URL, newf.Enabled)
		f.Name = newf.Name

		if f.URL != newf.URL {
			r |= statusURLChanged
			if filterExistsNoLock(newf.URL) {
				return statusURLExists
			}
			f.URL = newf.URL
			f.unload()
			f.LastUpdated = time.Time{}
		}

		if f.Enabled != newf.Enabled {
			r |= statusEnabledChanged
			f.Enabled = newf.Enabled
			if f.Enabled {
				if (r & statusURLChanged) == 0 {
					e := f.load()
					if e != nil {
						// This isn't a fatal error,
						//  because it may occur when someone removes the file from disk.
						// In this case the periodic update task will try to download the file.
						f.LastUpdated = time.Time{}
					}
				}
			} else {
				f.unload()
			}
		}

		return r | statusFound
	}
	return 0
}

// Return TRUE if a filter with this URL exists
func filterExists(url string) bool {
	config.RLock()
	r := filterExistsNoLock(url)
	config.RUnlock()
	return r
}

func filterExistsNoLock(url string) bool {
	for _, f := range config.Filters {
		if f.URL == url {
			return true
		}
	}
	for _, f := range config.WhitelistFilters {
		if f.URL == url {
			return true
		}
	}
	return false
}

// Add a filter
// Return FALSE if a filter with this URL exists
func filterAdd(f filter) bool {
	config.Lock()
	defer config.Unlock()

	// Check for duplicates
	if filterExistsNoLock(f.URL) {
		return false
	}

	if f.white {
		config.WhitelistFilters = append(config.WhitelistFilters, f)
	} else {
		config.Filters = append(config.Filters, f)
	}
	return true
}

// Load filters from the disk
// And if any filter has zero ID, assign a new one
func loadFilters(array []filter) {
	for i := range array {
		filter := &array[i] // otherwise we're operating on a copy
		if filter.ID == 0 {
			filter.ID = assignUniqueFilterID()
		}

		if !filter.Enabled {
			// No need to load a filter that is not enabled
			continue
		}

		err := filter.load()
		if err != nil {
			log.Error("Couldn't load filter %d contents due to %s", filter.ID, err)
		}
	}
}

func deduplicateFilters() {
	// Deduplicate filters
	i := 0 // output index, used for deletion later
	urls := map[string]bool{}
	for _, filter := range config.Filters {
		if _, ok := urls[filter.URL]; !ok {
			// we didn't see it before, keep it
			urls[filter.URL] = true // remember the URL
			config.Filters[i] = filter
			i++
		}
	}

	// all entries we want to keep are at front, delete the rest
	config.Filters = config.Filters[:i]
}

// Set the next filter ID to max(filter.ID) + 1
func updateUniqueFilterID(filters []filter) {
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
func periodicallyRefreshFilters() {
	const maxInterval = 1 * 60 * 60
	intval := 5 // use a dynamically increasing time interval
	for {
		isNetworkErr := false
		if config.DNS.FiltersUpdateIntervalHours != 0 && atomic.CompareAndSwapUint32(&refreshStatus, 0, 1) {
			refreshLock.Lock()
			_, isNetworkErr = refreshFiltersIfNecessary(false)
			refreshLock.Unlock()
			refreshStatus = 0
			if !isNetworkErr {
				intval = maxInterval
			}
		}

		if isNetworkErr {
			intval *= 2
			if intval > maxInterval {
				intval = maxInterval
			}
		}

		time.Sleep(time.Duration(intval) * time.Second)
	}
}

// Refresh filters
func refreshFilters() (int, error) {
	if !atomic.CompareAndSwapUint32(&refreshStatus, 0, 1) {
		return 0, fmt.Errorf("Filters update procedure is already running")
	}

	refreshLock.Lock()
	nUpdated, _ := refreshFiltersIfNecessary(true)
	refreshLock.Unlock()
	refreshStatus = 0
	return nUpdated, nil
}

// Checks filters updates if necessary
// If force is true, it ignores the filter.LastUpdated field value
//
// Algorithm:
// . Get the list of filters to be updated
// . For each filter run the download and checksum check operation
// . For each filter:
//  . If filter data hasn't changed, just set new update time on file
//  . If filter data has changed:
//    . rename the old file (1.txt -> 1.txt.old)
//    . store the new data on disk (1.txt)
//  . Pass new filters to dnsfilter object - it analyzes new data while the old filters are still active
//  . dnsfilter activates new filters
//  . Remove the old filter files (1.txt.old)
//
// Return the number of updated filters
// Return TRUE - there was a network error and nothing could be updated
func refreshFiltersIfNecessary(force bool) (int, bool) {
	var updateFilters []filter
	var updateFlags []bool // 'true' if filter data has changed

	log.Debug("Filters: updating...")

	now := time.Now()
	config.RLock()
	for i := range config.Filters {
		f := &config.Filters[i] // otherwise we will be operating on a copy

		if !f.Enabled {
			continue
		}

		expireTime := f.LastUpdated.Unix() + int64(config.DNS.FiltersUpdateIntervalHours)*60*60
		if !force && expireTime > now.Unix() {
			continue
		}

		var uf filter
		uf.ID = f.ID
		uf.URL = f.URL
		uf.Name = f.Name
		uf.checksum = f.checksum
		updateFilters = append(updateFilters, uf)
	}
	config.RUnlock()

	if len(updateFilters) == 0 {
		return 0, false
	}

	nfail := 0
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated, err := uf.update()
		updateFlags = append(updateFlags, updated)
		if err != nil {
			nfail++
			log.Printf("Failed to update filter %s: %s\n", uf.URL, err)
			continue
		}
		uf.LastUpdated = now
	}

	if nfail == len(updateFilters) {
		return 0, true
	}

	updateCount := 0
	for i := range updateFilters {
		uf := &updateFilters[i]
		updated := updateFlags[i]
		if updated {
			err := uf.saveAndBackupOld()
			if err != nil {
				log.Printf("Failed to save the updated filter %d: %s", uf.ID, err)
				continue
			}
		} else {
			e := os.Chtimes(uf.Path(), uf.LastUpdated, uf.LastUpdated)
			if e != nil {
				log.Error("os.Chtimes(): %v", e)
			}
		}

		config.Lock()
		for k := range config.Filters {
			f := &config.Filters[k]
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
			f.Data = nil
			f.RulesCount = uf.RulesCount
			f.checksum = uf.checksum
			updateCount++
		}
		config.Unlock()
	}

	if updateCount != 0 {
		enableFilters(false)

		for i := range updateFilters {
			uf := &updateFilters[i]
			updated := updateFlags[i]
			if !updated {
				continue
			}
			_ = os.Remove(uf.Path() + ".old")
		}
	}

	log.Debug("Filters: update finished")
	return updateCount, false
}

// Allows printable UTF-8 text with CR, LF, TAB characters
func isPrintableText(data []byte) bool {
	for _, c := range data {
		if (c >= ' ' && c != 0x7f) || c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		return false
	}
	return true
}

// A helper function that parses filter contents and returns a number of rules and a filter name (if there's any)
func parseFilterContents(contents []byte) (int, string) {
	data := string(contents)
	rulesCount := 0
	name := ""
	seenTitle := false

	// Count lines in the filter
	for len(data) != 0 {
		line := util.SplitNext(&data, '\n')
		if len(line) == 0 {
			continue
		}

		if line[0] == '!' {
			m := filterTitleRegexp.FindAllStringSubmatch(line, -1)
			if len(m) > 0 && len(m[0]) >= 2 && !seenTitle {
				name = m[0][1]
				seenTitle = true
			}
		} else {
			rulesCount++
		}
	}

	return rulesCount, name
}

// Perform upgrade on a filter
func (filter *filter) update() (bool, error) {
	log.Tracef("Downloading update for filter %d from %s", filter.ID, filter.URL)

	resp, err := Context.client.Get(filter.URL)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Printf("Couldn't request filter from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	if resp.StatusCode != 200 {
		log.Printf("Got status code %d from URL %s, skipping", resp.StatusCode, filter.URL)
		return false, fmt.Errorf("got status code != 200: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Couldn't fetch filter contents from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	// Check if the filter has been really changed
	checksum := crc32.ChecksumIEEE(body)
	if filter.checksum == checksum {
		log.Tracef("Filter #%d at URL %s hasn't changed, not updating it", filter.ID, filter.URL)
		return false, nil
	}

	var firstChunk []byte
	if len(body) <= 4096 {
		firstChunk = body
	} else {
		firstChunk = body[:4096]
	}
	if !isPrintableText(firstChunk) {
		return false, fmt.Errorf("Data contains non-printable characters")
	}

	s := strings.ToLower(string(firstChunk))
	if strings.Index(s, "<html") >= 0 ||
		strings.Index(s, "<!doctype") >= 0 {
		return false, fmt.Errorf("Data is HTML, not plain text")
	}

	// Extract filter name and count number of rules
	rulesCount, filterName := parseFilterContents(body)
	log.Printf("Filter %d has been updated: %d bytes, %d rules", filter.ID, len(body), rulesCount)
	if filterName != "" {
		filter.Name = filterName
	}
	filter.RulesCount = rulesCount
	filter.Data = body
	filter.checksum = checksum

	return true, nil
}

// saves filter contents to the file in dataDir
// This method is safe to call during filters update,
//  because it creates a new file and then renames it,
//  so the currently opened file descriptors to the old filter file remain valid.
func (filter *filter) save() error {
	filterFilePath := filter.Path()
	log.Printf("Saving filter %d contents to: %s", filter.ID, filterFilePath)

	err := file.SafeWrite(filterFilePath, filter.Data)

	// update LastUpdated field after saving the file
	filter.LastUpdated = filter.LastTimeUpdated()
	return err
}

func (filter *filter) saveAndBackupOld() error {
	filterFilePath := filter.Path()
	err := os.Rename(filterFilePath, filterFilePath+".old")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return filter.save()
}

// loads filter contents from the file in dataDir
func (filter *filter) load() error {
	filterFilePath := filter.Path()
	log.Tracef("Loading filter %d contents to: %s", filter.ID, filterFilePath)

	if _, err := os.Stat(filterFilePath); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return err
	}

	filterFileContents, err := ioutil.ReadFile(filterFilePath)
	if err != nil {
		return err
	}

	log.Tracef("File %s, id %d, length %d", filterFilePath, filter.ID, len(filterFileContents))
	rulesCount, _ := parseFilterContents(filterFileContents)

	filter.RulesCount = rulesCount
	filter.Data = nil
	filter.checksum = crc32.ChecksumIEEE(filterFileContents)
	filter.LastUpdated = filter.LastTimeUpdated()

	return nil
}

// Clear filter rules
func (filter *filter) unload() {
	filter.Data = nil
	filter.RulesCount = 0
}

// Path to the filter contents
func (filter *filter) Path() string {
	return filepath.Join(Context.getDataDir(), filterDir, strconv.FormatInt(filter.ID, 10)+".txt")
}

// LastTimeUpdated returns the time when the filter was last time updated
func (filter *filter) LastTimeUpdated() time.Time {
	filterFilePath := filter.Path()
	s, err := os.Stat(filterFilePath)
	if os.IsNotExist(err) {
		// if the filter file does not exist, return 0001-01-01
		return time.Time{}
	}

	if err != nil {
		// if the filter file does not exist, return 0001-01-01
		return time.Time{}
	}

	// filter file modified time
	return s.ModTime()
}

func enableFilters(async bool) {
	var filters []dnsfilter.Filter
	var whiteFilters []dnsfilter.Filter
	if config.DNS.FilteringEnabled {
		// convert array of filters

		userFilter := userFilter()
		f := dnsfilter.Filter{
			ID:   userFilter.ID,
			Data: userFilter.Data,
		}
		filters = append(filters, f)

		for _, filter := range config.Filters {
			if !filter.Enabled {
				continue
			}
			f := dnsfilter.Filter{
				ID:       filter.ID,
				FilePath: filter.Path(),
			}
			filters = append(filters, f)
		}
		for _, filter := range config.WhitelistFilters {
			if !filter.Enabled {
				continue
			}
			f := dnsfilter.Filter{
				ID:       filter.ID,
				FilePath: filter.Path(),
			}
			whiteFilters = append(whiteFilters, f)
		}
	}

	_ = Context.dnsFilter.SetFilters(filters, whiteFilters, async)
}
