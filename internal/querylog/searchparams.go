package querylog

import "time"

// searchParams represent the search query sent by the client
type searchParams struct {
	// searchCriteria - list of search criteria that we use to get filter results
	searchCriteria []searchCriterion

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

// quickMatchClientFunc is a simplified client finder for quick matches.
type quickMatchClientFunc = func(clientID, ip string) (c *Client)

// quickMatch quickly checks if the line matches the given search parameters.
// It returns false if the line doesn't match.  This method is only here for
// optimization purposes.
func (s *searchParams) quickMatch(line string, findClient quickMatchClientFunc) (ok bool) {
	for _, c := range s.searchCriteria {
		if !c.quickMatch(line, findClient) {
			return false
		}
	}

	return true
}

// match - checks if the logEntry matches the searchParams
func (s *searchParams) match(entry *logEntry) bool {
	if !s.olderThan.IsZero() && !entry.Time.Before(s.olderThan) {
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
