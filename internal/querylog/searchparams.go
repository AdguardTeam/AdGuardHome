package querylog

import "time"

// searchParams represent the search query sent by the client
type searchParams struct {
	// searchCriteria - list of search criteria that we use to get filter results
	searchCriteria []searchCriteria

	// olderThen - return entries that are older than this value
	// if not set - disregard it and return any value
	olderThan time.Time

	offset             int // offset for the search
	limit              int // limit the number of records returned
	maxFileScanEntries int // maximum log entries to scan in query log files. if 0 - no limit
}

// newSearchParams - creates an empty instance of searchParams
func newSearchParams() *searchParams {
	return &searchParams{
		// default max log entries to return
		limit: 500,

		// by default, we scan up to 50k entries at once
		maxFileScanEntries: 50000,
	}
}

// quickMatchesGetDataParams - quickly checks if the line matches the searchParams
// this method does not guarantee anything and the reason is to do a quick check
// without deserializing anything
func (s *searchParams) quickMatch(line string) bool {
	for _, c := range s.searchCriteria {
		if !c.quickMatch(line) {
			return false
		}
	}

	return true
}

// match - checks if the logEntry matches the searchParams
func (s *searchParams) match(entry *logEntry) bool {
	if !s.olderThan.IsZero() && entry.Time.UnixNano() >= s.olderThan.UnixNano() {
		// Ignore entries newer than what was requested
		return false
	}

	for _, c := range s.searchCriteria {
		if !c.match(entry) {
			return false
		}
	}

	return true
}
