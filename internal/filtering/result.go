package filtering

import (
	"fmt"
	"net/netip"
	"slices"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/urlfilter/rules"
)

// Result contains the result of a request check.  All fields transitively have
// omitempty tags so that the query log doesn't become too large.
//
// TODO(a.garipov): Clarify relationships between fields.  Perhaps replace with
// a sum type or an interface?
type Result struct {
	// DNSRewriteResult is the $dnsrewrite filter rule result.
	DNSRewriteResult *DNSRewriteResult `json:",omitempty"`

	// CanonName is the CNAME value from the lookup rewrite result.  It is empty
	// unless Reason is set to Rewritten or RewrittenRule.
	CanonName string `json:",omitempty"`

	// ServiceName is the name of the blocked service.  It is empty unless
	// Reason is set to FilteredBlockedService.
	ServiceName string `json:",omitempty"`

	// IPList is the lookup rewrite result.  It is empty unless Reason is set to
	// Rewritten.
	IPList []netip.Addr `json:",omitempty"`

	// Rules are applied rules.  If Rules are not empty, each rule is not nil.
	Rules []*ResultRule `json:",omitempty"`

	// Reason is the reason for blocking or unblocking the request.
	Reason Reason `json:",omitempty"`

	// IsFiltered is true if the request is filtered.
	//
	// TODO(d.kolyshev): Get rid of this flag.
	IsFiltered bool `json:",omitempty"`
}

// ResultRule contains information about applied rules.
type ResultRule struct {
	// Text is the text of the rule.
	Text string `json:",omitempty"`

	// IP is the host IP.  It is nil unless the rule uses the /etc/hosts syntax
	// or the reason is [FilteredSafeSearch].
	IP netip.Addr `json:",omitzero"`

	// FilterListID is the ID of the rule's filter list.
	FilterListID rulelist.APIID `json:",omitempty"`
}

// NewResultRule converts an URLFilter rule into a *ResultRule.  nr must not be
// nil.
func NewResultRule(r rules.Rule) (rr *ResultRule) {
	return &ResultRule{
		// #nosec G115 -- The overflow is required for backwards
		// compatibility.
		FilterListID: rulelist.APIID(r.GetFilterListID()),
		Text:         r.Text(),
	}
}

// Reason holds an enum detailing why it was filtered or not filtered
type Reason int

const (
	// NotFilteredNotFound: the host was not find in any checks, default value
	// for results.
	NotFilteredNotFound Reason = iota

	// NotFilteredAllowList: the host is explicitly allowed.
	NotFilteredAllowList

	// NotFilteredError is returned when there was an error during checking.
	// Reserved, currently unused.
	NotFilteredError

	// FilteredBlockList: the host was matched to be advertising host.
	FilteredBlockList

	// FilteredSafeBrowsing: the host was matched to be malicious/phishing.
	FilteredSafeBrowsing

	// FilteredParental: the host was matched to be outside of parental control
	// settings.
	FilteredParental

	// FilteredInvalid: the request was invalid and was not processed.
	FilteredInvalid

	// FilteredSafeSearch: the host was replaced with safesearch variant.
	FilteredSafeSearch

	// FilteredBlockedService: the host is blocked by the blocked services
	// feature.
	FilteredBlockedService

	// Rewritten is returned when there was a rewrite by a legacy DNS rewrite
	// rule.
	Rewritten

	// RewrittenAutoHosts is returned when there was a rewrite by /etc/hosts.
	RewrittenAutoHosts

	// RewrittenRule is returned when a $dnsrewrite filter rule was applied.
	//
	// TODO(a.garipov): Remove [Rewritten] and [RewrittenAutoHosts] by merging
	// their functionality into RewrittenRule.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2499.
	RewrittenRule
)

// TODO(a.garipov): Resync with actual code names or replace completely in HTTP
// API v1.
var reasonNames = []string{
	NotFilteredNotFound:  "NotFilteredNotFound",
	NotFilteredAllowList: "NotFilteredWhiteList",
	NotFilteredError:     "NotFilteredError",

	FilteredBlockList:      "FilteredBlackList",
	FilteredSafeBrowsing:   "FilteredSafeBrowsing",
	FilteredParental:       "FilteredParental",
	FilteredInvalid:        "FilteredInvalid",
	FilteredSafeSearch:     "FilteredSafeSearch",
	FilteredBlockedService: "FilteredBlockedService",

	Rewritten:          "Rewrite",
	RewrittenAutoHosts: "RewriteEtcHosts",
	RewrittenRule:      "RewriteRule",
}

// type check
var _ fmt.Stringer = NotFilteredNotFound

// String implements the [fmt.Stringer] interface for Reason.
func (r Reason) String() (s string) {
	if r < 0 || int(r) >= len(reasonNames) {
		return ""
	}

	return reasonNames[r]
}

// In returns true if reasons include r.
func (r Reason) In(reasons ...Reason) (ok bool) { return slices.Contains(reasons, r) }
