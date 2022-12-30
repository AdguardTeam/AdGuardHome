package filtering

import (
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/rules"
)

// RewriteStorage is a storage for rewrite rules.
type RewriteStorage interface {
	// MatchRequest returns matching dnsrewrites for the specified request.
	MatchRequest(dReq *urlfilter.DNSRequest) (rws []*rules.DNSRewrite)

	// Add adds item to the storage.
	Add(item *RewriteItem) (err error)

	// Remove deletes item from the storage.
	Remove(item *RewriteItem) (err error)

	// List returns all items from the storage.
	List() (items []*RewriteItem)
}

// RewriteItem is a single DNS rewrite record.
type RewriteItem struct {
	// Domain is the domain pattern for which this rewrite should work.
	Domain string `yaml:"domain" json:"domain"`

	// Answer is the IP address, canonical name, or one of the special
	// values: "A" or "AAAA".
	Answer string `yaml:"answer" json:"answer"`
}

// Equal returns true if rw is Equal to other.
func (rw *RewriteItem) Equal(other *RewriteItem) (ok bool) {
	if rw == nil {
		return other == nil
	} else if other == nil {
		return false
	}

	return *rw == *other
}
