package logs

import "time"

// searchParams represent the search query sent by the client
type SearchParams struct {
	// searchCriteria - list of search criteria that we use to get filter results
	SearchCriteria []SearchCriterion

	// olderThen - return entries that are older than this value
	// if not set - disregard it and return any value
	OlderThan time.Time

	Offset             int // offset for the search
	Limit              int // limit the number of records returned
	MaxFileScanEntries int // maximum log entries to scan in query log files. if 0 - no limit
}

func NewSearchParams() *SearchParams {
	return &SearchParams{
		// default max log entries to return
		Limit: 500,

		// by default, we scan up to 50k entries at once
		MaxFileScanEntries: 50000,
	}
}

// searchCriterion is a search criterion that is used to match a record.
type SearchCriterion struct {
	Value         string
	AsciiVal      string
	CriterionType CriterionType
	// strict, if true, means that the criterion must be applied to the
	// whole value rather than the part of it.  That is, equality and not
	// containment.
	Strict bool
}

// newSearchParams - creates an empty instance of searchParams
func newSearchParams() *SearchParams {
	return &SearchParams{
		// default max log entries to return
		Limit: 500,

		// by default, we scan up to 50k entries at once
		MaxFileScanEntries: 50000,
	}
}

// quickMatchClientFunc is a simplified client finder for quick matches.
type quickMatchClientFunc = func(clientID, ip string) (c *Client)
