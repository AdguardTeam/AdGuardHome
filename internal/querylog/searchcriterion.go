package querylog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/stringutil"
)

type criterionType int

const (
	// ctTerm is for searching by the domain name, the client's IP address,
	// the client's ID or the client's name.  The domain name search
	// supports IDNAs.
	ctTerm criterionType = iota
	// ctFilteringStatus is for searching by the filtering status.
	//
	// See (*searchCriterion).ctFilteringStatusCase for details.
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

// searchCriterion is a search criterion that is used to match a record.
type searchCriterion struct {
	value         string
	asciiVal      string
	criterionType criterionType
	// strict, if true, means that the criterion must be applied to the
	// whole value rather than the part of it.  That is, equality and not
	// containment.
	strict bool
}

func ctDomainOrClientCaseStrict(
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

func ctDomainOrClientCaseNonStrict(
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

// quickMatch quickly checks if the line matches the given search criterion.
// It returns false if the like doesn't match.  This method is only here for
// optimization purposes.
func (c *searchCriterion) quickMatch(
	ctx context.Context,
	logger *slog.Logger,
	line string,
	findClient quickMatchClientFunc,
) (ok bool) {
	switch c.criterionType {
	case ctTerm:
		host := readJSONValue(line, `"QH":"`)
		ip := readJSONValue(line, `"IP":"`)
		clientID := readJSONValue(line, `"CID":"`)

		var name string
		if cli := findClient(ctx, logger, clientID, ip); cli != nil {
			name = cli.Name
		}

		if c.strict {
			return ctDomainOrClientCaseStrict(c.value, c.asciiVal, clientID, name, host, ip)
		}

		return ctDomainOrClientCaseNonStrict(c.value, c.asciiVal, clientID, name, host, ip)
	case ctFilteringStatus:
		// Go on, as we currently don't do quick matches against
		// filtering statuses.
		return true
	default:
		return true
	}
}

// match checks if the log entry matches this search criterion.
func (c *searchCriterion) match(entry *logEntry) bool {
	switch c.criterionType {
	case ctTerm:
		return c.ctDomainOrClientCase(entry)
	case ctFilteringStatus:
		return c.ctFilteringStatusCase(entry.Result.Reason, entry.Result.IsFiltered)
	}

	return false
}

func (c *searchCriterion) ctDomainOrClientCase(e *logEntry) bool {
	clientID := e.ClientID
	host := e.QHost

	var name string
	if e.client != nil {
		name = e.client.Name
	}

	ip := e.IP.String()
	if c.strict {
		return ctDomainOrClientCaseStrict(c.value, c.asciiVal, clientID, name, host, ip)
	}

	return ctDomainOrClientCaseNonStrict(c.value, c.asciiVal, clientID, name, host, ip)
}

// ctFilteringStatusCase returns true if the result matches the value.
func (c *searchCriterion) ctFilteringStatusCase(
	reason filtering.Reason,
	isFiltered bool,
) (matched bool) {
	switch c.value {
	case filteringStatusAll:
		return true
	case filteringStatusFiltered:
		return isFiltered || reason.In(
			filtering.NotFilteredAllowList,
			filtering.Rewritten,
			filtering.RewrittenAutoHosts,
			filtering.RewrittenRule,
		)
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
		return reason.In(
			filtering.Rewritten,
			filtering.RewrittenAutoHosts,
			filtering.RewrittenRule,
		)
	case filteringStatusProcessed:
		return !reason.In(
			filtering.FilteredBlockList,
			filtering.FilteredBlockedService,
			filtering.NotFilteredAllowList,
		)
	default:
		return false
	}
}

// isFilteredWithReason returns true if reason matches the criterion value.
// c.value must be one of:
//
//   - filteringStatusBlocked
//   - filteringStatusBlockedParental
//   - filteringStatusBlockedSafebrowsing
//   - filteringStatusBlockedService
//   - filteringStatusSafeSearch
func (c *searchCriterion) isFilteredWithReason(reason filtering.Reason) (matched bool) {
	switch c.value {
	case filteringStatusBlocked:
		return reason.In(filtering.FilteredBlockList, filtering.FilteredBlockedService)
	case filteringStatusBlockedParental:
		return reason == filtering.FilteredParental
	case filteringStatusBlockedSafebrowsing:
		return reason == filtering.FilteredSafeBrowsing
	case filteringStatusBlockedService:
		return reason == filtering.FilteredBlockedService
	case filteringStatusSafeSearch:
		return reason == filtering.FilteredSafeSearch
	default:
		panic(fmt.Errorf("unexpected value %q", c.value))
	}
}
