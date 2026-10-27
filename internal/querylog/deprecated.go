package querylog

import (
	"fmt"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
)

// TODO(a.garipov):  Remove once the old frontend is completely gone.

// ctFilteringStatus is for searching by the filtering status.
//
// See (*searchCriterion).ctFilteringStatusCase for details.
//
// Deprecated: Remove when migration to reason criterion is complete.
const ctFilteringStatus criterionType = ctReason + 1

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

// filteringStatusValues is the set of all possible [filteringStatus] values.
var filteringStatusValues = container.NewMapSet(
	filteringStatusAll,
	filteringStatusBlocked,
	filteringStatusBlockedParental,
	filteringStatusBlockedSafebrowsing,
	filteringStatusBlockedService,
	filteringStatusFiltered,
	filteringStatusProcessed,
	filteringStatusRewritten,
	filteringStatusSafeSearch,
	filteringStatusWhitelisted,
)

// ctFilteringStatusCase returns true if the result matches the value.
func (c *searchCriterion) ctFilteringStatusCase(
	reason filtering.Reason,
	isFiltered bool,
) (matched bool) {
	switch c.value {
	case filteringStatusAll:
		return true
	case filteringStatusFiltered:
		return isFiltered || reason == filtering.NotFilteredAllowList || reasonIsRewrite(reason)
	case
		filteringStatusBlocked,
		filteringStatusBlockedParental,
		filteringStatusBlockedSafebrowsing,
		filteringStatusBlockedService,
		filteringStatusSafeSearch:
		return isFiltered && c.isFilteredWithReason(reason)
	case filteringStatusWhitelisted:
		return reason == filtering.NotFilteredAllowList
	case filteringStatusRewritten:
		return reasonIsRewrite(reason)
	case filteringStatusProcessed:
		return !reasonIsRuleList(reason)
	default:
		return false
	}
}

// reasonIsRewrite returns true if r is one of:
//
//   - [filtering.RewrittenAutoHosts]
//   - [filtering.RewrittenRule]
//   - [filtering.Rewritten]
func reasonIsRewrite(r filtering.Reason) (ok bool) {
	return r == filtering.RewrittenAutoHosts ||
		r == filtering.RewrittenRule ||
		r == filtering.Rewritten
}

// isFilteredWithReason returns true if reason matches the criterion value.
// c.value must be one of:
//
//   - [filteringStatusBlockedParental]
//   - [filteringStatusBlockedSafebrowsing]
//   - [filteringStatusBlockedService]
//   - [filteringStatusBlocked]
//   - [filteringStatusSafeSearch]
func (c *searchCriterion) isFilteredWithReason(reason filtering.Reason) (matched bool) {
	switch c.value {
	case filteringStatusBlocked:
		switch reason {
		case
			filtering.FilteredBlockList,
			filtering.FilteredBlockedService:
			return true
		default:
			return false
		}
	case filteringStatusBlockedParental:
		return reason == filtering.FilteredParental
	case filteringStatusBlockedSafebrowsing:
		return reason == filtering.FilteredSafeBrowsing
	case filteringStatusBlockedService:
		return reason == filtering.FilteredBlockedService
	case filteringStatusSafeSearch:
		return reason == filtering.FilteredSafeSearch
	default:
		panic(fmt.Errorf("%w: %q", errors.ErrBadEnumValue, c.value))
	}
}

// reasonIsRuleList returns true if r is one of:
//
//   - [filtering.FilteredBlockList]
//   - [filtering.FilteredBlockedService]
//   - [filtering.NotFilteredAllowList]
func reasonIsRuleList(r filtering.Reason) (ok bool) {
	return r == filtering.FilteredBlockList ||
		r == filtering.FilteredBlockedService ||
		r == filtering.NotFilteredAllowList
}
