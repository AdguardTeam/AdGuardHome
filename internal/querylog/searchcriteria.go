package querylog

import (
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
)

type criteriaType int

const (
	// ctDomainOrClient is for searching by the domain name, the client's IP
	// address, or the clinet's ID.
	ctDomainOrClient criteriaType = iota
	// ctFilteringStatus is for searching by the filtering status.
	//
	// See (*searchCriteria).ctFilteringStatusCase for details.
	ctFilteringStatus
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
	value        string       // search criteria value
	criteriaType criteriaType // type of the criteria
	strict       bool         // should we strictly match (equality) or not (indexOf)
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

func (c *searchCriteria) ctDomainOrClientCaseStrict(term, clientID, name, host, ip string) bool {
	return strings.EqualFold(host, term) ||
		strings.EqualFold(clientID, term) ||
		strings.EqualFold(ip, term) ||
		strings.EqualFold(name, term)
}

func (c *searchCriteria) ctDomainOrClientCase(e *logEntry) bool {
	clientID := e.ClientID
	host := e.QHost

	var name string
	if e.client != nil {
		name = e.client.Name
	}

	ip := e.IP.String()
	term := strings.ToLower(c.value)
	if c.strict {
		return c.ctDomainOrClientCaseStrict(term, clientID, name, host, ip)
	}

	// TODO(a.garipov): Write a case-insensitive version of strings.Contains
	// instead of generating garbage.  Or, perhaps in the future, use
	// a locale-appropriate matcher from golang.org/x/text.
	clientID = strings.ToLower(clientID)
	host = strings.ToLower(host)
	ip = strings.ToLower(ip)
	name = strings.ToLower(name)
	term = strings.ToLower(term)

	return strings.Contains(clientID, term) ||
		strings.Contains(host, term) ||
		strings.Contains(ip, term) ||
		strings.Contains(name, term)
}

func (c *searchCriteria) ctFilteringStatusCase(res dnsfilter.Result) bool {
	switch c.value {
	case filteringStatusAll:
		return true

	case filteringStatusFiltered:
		return res.IsFiltered ||
			res.Reason.In(
				dnsfilter.NotFilteredAllowList,
				dnsfilter.Rewritten,
				dnsfilter.RewrittenAutoHosts,
				dnsfilter.RewrittenRule,
			)

	case filteringStatusBlocked:
		return res.IsFiltered &&
			res.Reason.In(dnsfilter.FilteredBlockList, dnsfilter.FilteredBlockedService)

	case filteringStatusBlockedService:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredBlockedService

	case filteringStatusBlockedParental:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredParental

	case filteringStatusBlockedSafebrowsing:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredSafeBrowsing

	case filteringStatusWhitelisted:
		return res.Reason == dnsfilter.NotFilteredAllowList

	case filteringStatusRewritten:
		return res.Reason.In(
			dnsfilter.Rewritten,
			dnsfilter.RewrittenAutoHosts,
			dnsfilter.RewrittenRule,
		)

	case filteringStatusSafeSearch:
		return res.IsFiltered && res.Reason == dnsfilter.FilteredSafeSearch

	case filteringStatusProcessed:
		return !res.Reason.In(
			dnsfilter.FilteredBlockList,
			dnsfilter.FilteredBlockedService,
			dnsfilter.NotFilteredAllowList,
		)

	default:
		return false
	}
}
