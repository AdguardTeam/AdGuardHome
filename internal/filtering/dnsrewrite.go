package filtering

import (
	"github.com/AdguardTeam/urlfilter"
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

// processDNSRewrites processes DNS rewrite rules in dnsr.  It returns an empty
// result if dnsr is empty.  Otherwise, the result will have either CanonName or
// DNSRewriteResult set.  dnsr is expected to be non-empty.
func (d *DNSFilter) processDNSRewrites(dnsr []*rules.NetworkRule) (res Result) {
	var rules []*ResultRule
	dnsrr := &DNSRewriteResult{
		Response: DNSRewriteResultResponse{},
	}

	for _, nr := range dnsr {
		dr := nr.DNSRewrite
		if dr.NewCNAME != "" {
			// NewCNAME rules have a higher priority than other rules.
			rules = []*ResultRule{{
				FilterListID: nr.GetFilterListID(),
				Text:         nr.RuleText,
			}}

			return Result{
				Rules:     rules,
				Reason:    RewrittenRule,
				CanonName: dr.NewCNAME,
			}
		}

		switch dr.RCode {
		case dns.RcodeSuccess:
			dnsrr.RCode = dr.RCode
			dnsrr.Response[dr.RRType] = append(dnsrr.Response[dr.RRType], dr.Value)
			rules = append(rules, &ResultRule{
				FilterListID: nr.GetFilterListID(),
				Text:         nr.RuleText,
			})
		default:
			// RcodeRefused and other such codes have higher priority.  Return
			// immediately.
			rules = []*ResultRule{{
				FilterListID: nr.GetFilterListID(),
				Text:         nr.RuleText,
			}}
			dnsrr = &DNSRewriteResult{
				RCode: dr.RCode,
			}

			return Result{
				DNSRewriteResult: dnsrr,
				Rules:            rules,
				Reason:           RewrittenRule,
			}
		}
	}

	return Result{
		DNSRewriteResult: dnsrr,
		Rules:            rules,
		Reason:           RewrittenRule,
	}
}

// processDNSResultRewrites returns an empty Result if there are no dnsrewrite
// rules in dnsres.  Otherwise, it returns the processed Result.
func (d *DNSFilter) processDNSResultRewrites(
	dnsres *urlfilter.DNSResult,
	host string,
) (dnsRWRes Result) {
	dnsr := dnsres.DNSRewrites()
	if len(dnsr) == 0 {
		return Result{}
	}

	res := d.processDNSRewrites(dnsr)
	if res.Reason == RewrittenRule && res.CanonName == host {
		// A rewrite of a host to itself.  Go on and try matching other things.
		return Result{}
	}

	return res
}
