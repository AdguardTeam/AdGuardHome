package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
)

var (
	nextFilterID      = time.Now().Unix() // semi-stable way to generate an unique ID
	filterTitleRegexp = regexp.MustCompile(`^! Title: +(.*)$`)
)

// field ordering is important -- yaml fields will mirror ordering from here
type filter struct {
	Enabled     bool      `json:"enabled"`
	URL         string    `json:"url"`
	Name        string    `json:"name" yaml:"name"`
	RulesCount  int       `json:"rulesCount" yaml:"-"`
	LastUpdated time.Time `json:"lastUpdated,omitempty" yaml:"-"`

	dnsfilter.Filter `yaml:",inline"`
}

// Creates a helper object for working with the user rules
func userFilter() filter {
	return filter{
		// User filter always has constant ID=0
		Enabled: true,
		Filter: dnsfilter.Filter{
			Rules: config.UserRules,
		},
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
	for range time.Tick(time.Minute) {
		refreshFiltersIfNecessary(false)
	}
}

// Checks filters updates if necessary
// If force is true, it ignores the filter.LastUpdated field value
func refreshFiltersIfNecessary(force bool) int {
	config.Lock()

	// fetch URLs
	updateCount := 0
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we will be operating on a copy

		if filter.ID == 0 { // protect against users modifying the yaml and removing the ID
			filter.ID = assignUniqueFilterID()
		}

		if len(filter.Rules) == 0 {
			// Try reloading filter from the disk before updating
			// This is useful for the case when we simply enable a previously downloaded filter
			_ = filter.load()
		}

		updated, err := filter.update(force)
		if err != nil {
			log.Printf("Failed to update filter %s: %s\n", filter.URL, err)
			continue
		}
		if updated {
			// Saving it to the filters dir now
			err = filter.save()
			if err != nil {
				log.Printf("Failed to save the updated filter %d: %s", filter.ID, err)
				continue
			}

			updateCount++
		}
	}
	config.Unlock()

	if updateCount > 0 && isRunning() {
		err := reconfigureDNSServer()
		if err != nil {
			msg := fmt.Sprintf("SHOULD NOT HAPPEN: cannot reconfigure DNS server with the new filters: %s", err)
			panic(msg)
		}
	}
	return updateCount
}

// A helper function that parses filter contents and returns a number of rules and a filter name (if there's any)
func parseFilterContents(contents []byte) (int, string, []string) {
	lines := strings.Split(string(contents), "\n")
	rulesCount := 0
	name := ""
	seenTitle := false

	// Count lines in the filter
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] == '!' {
			if m := filterTitleRegexp.FindAllStringSubmatch(line, -1); len(m) > 0 && len(m[0]) >= 2 && !seenTitle {
				name = m[0][1]
				seenTitle = true
			}
		} else if len(line) != 0 {
			rulesCount++
		}
	}

	return rulesCount, name, lines
}

// Checks for filters updates
// If "force" is true -- does not check the filter's LastUpdated field
// Call "save" to persist the filter contents
func (filter *filter) update(force bool) (bool, error) {
	if filter.ID == 0 { // protect against users deleting the ID
		filter.ID = assignUniqueFilterID()
	}
	if !filter.Enabled {
		return false, nil
	}
	if !force && time.Since(filter.LastUpdated) <= updatePeriod {
		return false, nil
	}

	log.Tracef("Downloading update for filter %d from %s", filter.ID, filter.URL)

	resp, err := client.Get(filter.URL)
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

	contentType := strings.ToLower(resp.Header.Get("content-type"))
	if !strings.HasPrefix(contentType, "text/plain") {
		log.Printf("Non-text response %s from %s, skipping", contentType, filter.URL)
		return false, fmt.Errorf("non-text response %s", contentType)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Couldn't fetch filter contents from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	// Extract filter name and count number of rules
	rulesCount, filterName, rules := parseFilterContents(body)

	if filterName != "" {
		filter.Name = filterName
	}

	// Check if the filter has been really changed
	if reflect.DeepEqual(filter.Rules, rules) {
		log.Tracef("Filter #%d at URL %s hasn't changed, not updating it", filter.ID, filter.URL)
		return false, nil
	}

	log.Printf("Filter %d has been updated: %d bytes, %d rules", filter.ID, len(body), rulesCount)
	filter.RulesCount = rulesCount
	filter.Rules = rules

	return true, nil
}

// saves filter contents to the file in dataDir
func (filter *filter) save() error {
	filterFilePath := filter.Path()
	log.Printf("Saving filter %d contents to: %s", filter.ID, filterFilePath)
	body := []byte(strings.Join(filter.Rules, "\n"))

	err := file.SafeWrite(filterFilePath, body)

	// update LastUpdated field after saving the file
	filter.LastUpdated = filter.LastTimeUpdated()
	return err
}

// loads filter contents from the file in dataDir
func (filter *filter) load() error {
	if !filter.Enabled {
		// No need to load a filter that is not enabled
		return nil
	}

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
	rulesCount, _, rules := parseFilterContents(filterFileContents)

	filter.RulesCount = rulesCount
	filter.Rules = rules
	filter.LastUpdated = filter.LastTimeUpdated()

	return nil
}

// Path to the filter contents
func (filter *filter) Path() string {
	return filepath.Join(config.ourWorkingDir, dataDir, filterDir, strconv.FormatInt(filter.ID, 10)+".txt")
}

// LastUpdated returns the time when the filter was last time updated
func (filter *filter) LastTimeUpdated() time.Time {
	filterFilePath := filter.Path()
	if _, err := os.Stat(filterFilePath); os.IsNotExist(err) {
		// if the filter file does not exist, return 0001-01-01
		return time.Time{}
	}

	s, err := os.Stat(filterFilePath)
	if err != nil {
		// if the filter file does not exist, return 0001-01-01
		return time.Time{}
	}

	// filter file modified time
	return s.ModTime()
}
