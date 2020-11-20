package querylog

import (
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
)

type criteriaType int

const (
	ctDomainOrClient  criteriaType = iota // domain name or client IP address
	ctFilteringStatus                     // filtering status
)

const (
	filteringStatusAll      = "all"
	filteringStatusFiltered = "filtered" // all kinds of filtering

	filteringStatusBlocked             = "blocked"              // blocked or blocked services
	filteringStatusBlockedService      = "blocked_services"     // blocked
	filteringStatusBlockedSafebrowsing = "blocked_safebrowsing" // blocked by safebrowsing
	filteringStatusBlockedParental     = "blocked_parental"     // blocked by parental control
	filteringStatusWhitelisted         = "whitelisted"          // whitelisted
	filteringStatusRewritten           = "rewritten"            // all kinds of rewrites
	filteringStatusSafeSearch          = "safe_search"          // enforced safe search
	filteringStatusProcessed           = "processed"            // not blocked, not white-listed entries
)

// filteringStatusValues -- array with all possible filteringStatus values
var filteringStatusValues = []string{
	filteringStatusAll, filteringStatusFiltered, filteringStatusBlocked,
	filteringStatusBlockedService, filteringStatusBlockedSafebrowsing, filteringStatusBlockedParental,
	filteringStatusWhitelisted, filteringStatusRewritten, filteringStatusSafeSearch,
	filteringStatusProcessed,
}

// searchCriteria - every search request may contain a list of different search criteria
// we use each of them to match the query
type searchCriteria struct {
	criteriaType criteriaType // type of the criteria
	strict       bool         // should we strictly match (equality) or not (indexOf)
	value        string       // search criteria value
}

// quickMatch - quickly checks if the log entry matches this search criteria
// the reason is to do it as quickly as possible without de-serializing the entry
func (c *searchCriteria) quickMatch(line string) bool {
	// note that we do this only for a limited set of criteria

	switch c.criteriaType {
	case ctDomainOrClient:
		return c.quickMatchJSONValue(line, "QH") ||
			c.quickMatchJSONValue(line, "IP")
	default:
		return true
	}
}

// quickMatchJSONValue - helper used by quickMatch
func (c *searchCriteria) quickMatchJSONValue(line string, propertyName string) bool {
	val := readJSONValue(line, propertyName)
	if len(val) == 0 {
		return false
	}
	val = strings.ToLower(val)
	searchVal := strings.ToLower(c.value)

	if c.strict && searchVal == val {
		return true
	}
	if !c.strict && strings.Contains(val, searchVal) {
		return true
	}

	return false
}

// match - checks if the log entry matches this search criteria
func (c *searchCriteria) match(entry *logEntry) bool {
	switch c.criteriaType {
	case ctDomainOrClient:
		return c.ctDomainOrClientCase(entry)
	case ctFilteringStatus:
		return c.ctFilteringStatusCase(entry.Result)
	}

	return false
}

func (c *searchCriteria) ctDomainOrClientCase(entry *logEntry) bool {
	qhost := strings.ToLower(entry.QHost)
	searchVal := strings.ToLower(c.value)
	if c.strict && qhost == searchVal {
		return true
	}
	if !c.strict && strings.Contains(qhost, searchVal) {
		return true
	}

	if c.strict && entry.IP == c.value {
		return true
	}
	if !c.strict && strings.Contains(entry.IP, c.value) {
		return true
	}
	return false
}

func (c *searchCriteria) ctFilteringStatusCase(res dnsfilter.Result) bool {
	switch c.value {
	case filteringStatusAll:
		return true

	case filteringStatusFiltered:
		return res.IsFiltered ||
			res.Reason.In(
				dnsfilter.NotFilteredWhiteList,
				dnsfilter.ReasonRewrite,
				dnsfilter.RewriteEtcHosts,
			)

	case filteringStatusBlocked:
		return res.IsFiltered &&
			res.Reason.In(dnsfilter.FilteredBlackList, dnsfilter.FilteredBlockedService)

	case filteringStatusBlockedService:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredBlockedService

	case filteringStatusBlockedParental:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredParental

	case filteringStatusBlockedSafebrowsing:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredSafeBrowsing

	case filteringStatusWhitelisted:
		return res.Reason == dnsfilter.NotFilteredWhiteList

	case filteringStatusRewritten:
		return res.Reason.In(dnsfilter.ReasonRewrite, dnsfilter.RewriteEtcHosts)

	case filteringStatusSafeSearch:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredSafeSearch

	case filteringStatusProcessed:
		return !res.Reason.In(
			dnsfilter.FilteredBlackList,
			dnsfilter.FilteredBlockedService,
			dnsfilter.NotFilteredWhiteList,
		)

	default:
		return false
	}
}
