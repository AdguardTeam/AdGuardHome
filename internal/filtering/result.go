package filtering

import (
	"net/netip"

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
