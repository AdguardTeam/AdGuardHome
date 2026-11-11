package querylog

import (
	"context"
	"log/slog"
	"time"
)

const (
	// defaultLimit is the default pagination limit.
	defaultLimit = 500

	// defaultMaxFileScanEntries is the default maximum number of log entries to
	// scan from file.
	defaultMaxFileScanEntries = 50000
)

// searchParams represent the search query sent by the client.
type searchParams struct {
	// olderThen represents a parameter for entries that are older than this
	// parameter value.  If not set, disregard it and return any value.
	olderThan time.Time

	// searchCriteria is a list of search criteria that we use to get filter
	// results.
	searchCriteria []searchCriterion

	// offset for the search.  It must not be negative.  Its sum with
	// [searchParams.limit] must be less than or equal [math.MaxInt].
	//
	// TODO(f.setrakov): Use uint.
	offset int

	// limit is the maximum number of records returned.  It must not be
	// negative.  Its sum with [searchParams.offset] must be less than or equal
	// [math.MaxInt].
	//
	// TODO(f.setrakov): Use uint.
	limit int

	// maxFileScanEntries is a maximum of log entries to scan in query log
	// files.  If not set, then no limit.
	maxFileScanEntries int
}

// newSearchParams - creates an empty instance of searchParams
func newSearchParams() *searchParams {
	return &searchParams{
		limit:              defaultLimit,
		maxFileScanEntries: defaultMaxFileScanEntries,
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
