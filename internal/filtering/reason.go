package filtering

import "fmt"

// Reason holds an enum detailing why it was filtered, allowed, or rewritten.
type Reason uint8

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

// TODO(a.garipov): Resync with actual code names or replace completely in the
// next version of HTTP API.
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
	if int(r) >= len(reasonNames) {
		return ""
	}

	return reasonNames[r]
}
