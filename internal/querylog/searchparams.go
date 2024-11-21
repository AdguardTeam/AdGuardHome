package querylog

import (
	"context"
	"log/slog"
	"time"
)

// searchParams represent the search query sent by the client.
type searchParams struct {
	// olderThen represents a parameter for entries that are older than this
	// parameter value.  If not set, disregard it and return any value.
	olderThan time.Time

	// searchCriteria is a list of search criteria that we use to get filter
	// results.
	searchCriteria []searchCriterion

	// offset for the search.
	offset int

	// limit the number of records returned.
	limit int

	// maxFileScanEntries is a maximum of log entries to scan in query log
	// files.  If not set, then no limit.
	maxFileScanEntries int
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
type quickMatchClientFunc = func(
	ctx context.Context,
	logger *slog.Logger,
	clientID, ip string,
) (c *Client)

// quickMatch quickly checks if the line matches the given search parameters.
// It returns false if the line doesn't match.  This method is only here for
// optimization purposes.
func (s *searchParams) quickMatch(
	ctx context.Context,
	logger *slog.Logger,
	line string,
	findClient quickMatchClientFunc,
) (ok bool) {
	for _, c := range s.searchCriteria {
		if !c.quickMatch(ctx, logger, line, findClient) {
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
