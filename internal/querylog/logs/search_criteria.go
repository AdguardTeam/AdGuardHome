package logs

import (
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/stringutil"
)

type CriterionType int

const (
	// ctTerm is for searching by the domain name, the client's IP address,
	// the client's ID or the client's name.  The domain name search
	// supports IDNAs.
	CtTerm CriterionType = iota
	CtFilteringStatus
)

const (
	FilteringStatusAll      = "all"
	FilteringStatusFiltered = "filtered" // all kinds of filtering

	FilteringStatusBlocked             = "blocked"              // blocked or blocked services
	FilteringStatusBlockedService      = "blocked_services"     // blocked
	FilteringStatusBlockedSafebrowsing = "blocked_safebrowsing" // blocked by safebrowsing
	FilteringStatusBlockedParental     = "blocked_parental"     // blocked by parental control
	FilteringStatusWhitelisted         = "whitelisted"          // whitelisted
	FilteringStatusRewritten           = "rewritten"            // all kinds of rewrites
	FilteringStatusSafeSearch          = "safe_search"          // enforced safe search
	FilteringStatusProcessed           = "processed"            // not blocked, not white-listed entries
)

// filteringStatusValues -- array with all possible filteringStatus values
var filteringStatusValues = []string{
	FilteringStatusAll, FilteringStatusFiltered, FilteringStatusBlocked,
	FilteringStatusBlockedService, FilteringStatusBlockedSafebrowsing, FilteringStatusBlockedParental,
	FilteringStatusWhitelisted, FilteringStatusRewritten, FilteringStatusSafeSearch,
	FilteringStatusProcessed,
}

func CtDomainOrClientCaseStrict(
	term string,
	asciiTerm string,
	clientID string,
	name string,
	host string,
	ip string,
) (ok bool) {
	return strings.EqualFold(host, term) ||
		(asciiTerm != "" && strings.EqualFold(host, asciiTerm)) ||
		strings.EqualFold(clientID, term) ||
		strings.EqualFold(ip, term) ||
		strings.EqualFold(name, term)
}

func CtDomainOrClientCaseNonStrict(
	term string,
	asciiTerm string,
	clientID string,
	name string,
	host string,
	ip string,
) (ok bool) {
	return stringutil.ContainsFold(clientID, term) ||
		stringutil.ContainsFold(host, term) ||
		(asciiTerm != "" && stringutil.ContainsFold(host, asciiTerm)) ||
		stringutil.ContainsFold(ip, term) ||
		stringutil.ContainsFold(name, term)
}

func CtFilteringStatusCase(c *SearchCriterion, res filtering.Result) bool {
	switch c.Value {
	case FilteringStatusAll:
		return true

	case FilteringStatusFiltered:
		return res.IsFiltered ||
			res.Reason.In(
				filtering.NotFilteredAllowList,
				filtering.Rewritten,
				filtering.RewrittenAutoHosts,
				filtering.RewrittenRule,
			)

	case FilteringStatusBlocked:
		return res.IsFiltered &&
			res.Reason.In(filtering.FilteredBlockList, filtering.FilteredBlockedService)

	case FilteringStatusBlockedService:
		return res.IsFiltered && res.Reason == filtering.FilteredBlockedService

	case FilteringStatusBlockedParental:
		return res.IsFiltered && res.Reason == filtering.FilteredParental

	case FilteringStatusBlockedSafebrowsing:
		return res.IsFiltered && res.Reason == filtering.FilteredSafeBrowsing

	case FilteringStatusWhitelisted:
		return res.Reason == filtering.NotFilteredAllowList

	case FilteringStatusRewritten:
		return res.Reason.In(
			filtering.Rewritten,
			filtering.RewrittenAutoHosts,
			filtering.RewrittenRule,
		)

	case FilteringStatusSafeSearch:
		return res.IsFiltered && res.Reason == filtering.FilteredSafeSearch
	case FilteringStatusProcessed:
		return !res.Reason.In(
			filtering.FilteredBlockList,
			filtering.FilteredBlockedService,
			filtering.NotFilteredAllowList,
		)

	default:
		return false
	}
}
