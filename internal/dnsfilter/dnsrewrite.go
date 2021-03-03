package dnsfilter

import (
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// DNSRewriteResult is the result of application of $dnsrewrite rules.
type DNSRewriteResult struct {
	Response DNSRewriteResultResponse `json:",omitempty"`
	RCode    rules.RCode              `json:",omitempty"`
}

// DNSRewriteResultResponse is the collection of DNS response records
// the server returns.
type DNSRewriteResultResponse map[rules.RRType][]rules.RRValue

// processDNSRewrites processes DNS rewrite rules in dnsr.  It returns
// an empty result if dnsr is empty.  Otherwise, the result will have
// either CanonName or DNSRewriteResult set.
func (d *DNSFilter) processDNSRewrites(dnsr []*rules.NetworkRule) (res Result) {
	if len(dnsr) == 0 {
		return Result{}
	}

	var rules []*ResultRule
	dnsrr := &DNSRewriteResult{
		Response: DNSRewriteResultResponse{},
	}

	for _, nr := range dnsr {
		dr := nr.DNSRewrite
		if dr.NewCNAME != "" {
			// NewCNAME rules have a higher priority than
			// the other rules.
			rules = []*ResultRule{{
				FilterListID: int64(nr.GetFilterListID()),
				Text:         nr.RuleText,
			}}

			return Result{
				Reason:    RewrittenRule,
				Rules:     rules,
				CanonName: dr.NewCNAME,
			}
		}

		switch dr.RCode {
		case dns.RcodeSuccess:
			dnsrr.RCode = dr.RCode
			dnsrr.Response[dr.RRType] = append(dnsrr.Response[dr.RRType], dr.Value)
			rules = append(rules, &ResultRule{
				FilterListID: int64(nr.GetFilterListID()),
				Text:         nr.RuleText,
			})
		default:
			// RcodeRefused and other such codes have higher
			// priority.  Return immediately.
			rules = []*ResultRule{{
				FilterListID: int64(nr.GetFilterListID()),
				Text:         nr.RuleText,
			}}
			dnsrr = &DNSRewriteResult{
				RCode: dr.RCode,
			}

			return Result{
				Reason:           RewrittenRule,
				Rules:            rules,
				DNSRewriteResult: dnsrr,
			}
		}
	}

	return Result{
		Reason:           RewrittenRule,
		Rules:            rules,
		DNSRewriteResult: dnsrr,
	}
}
