package filtering

import (
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// matchSysHosts tries to match the host against the operating system's hosts
// database.  err is always nil.
func (d *DNSFilter) matchSysHosts(
	host string,
	qtype uint16,
	setts *Settings,
) (res Result, err error) {
	// TODO(e.burkov):  Where else is this checked?
	if !setts.FilteringEnabled || d.conf.EtcHosts == nil {
		return Result{}, nil
	}

	vals, rs, matched := hostsRewrites(qtype, host, d.conf.EtcHosts)
	if !matched {
		return Result{}, nil
	}

	return Result{
		DNSRewriteResult: &DNSRewriteResult{
			Response: DNSRewriteResultResponse{
				qtype: vals,
			},
			RCode: dns.RcodeSuccess,
		},
		Rules:  rs,
		Reason: RewrittenAutoHosts,
	}, nil
}

// hostsRewrites returns values and rules matched by qt and host within hs.
func hostsRewrites(
	qtype uint16,
	host string,
	hs hostsfile.Storage,
) (vals []rules.RRValue, rls []*ResultRule, matched bool) {
	var isValidProto func(netip.Addr) (ok bool)
	switch qtype {
	case dns.TypeA:
		isValidProto = netip.Addr.Is4
	case dns.TypeAAAA:
		isValidProto = netip.Addr.Is6
	case dns.TypePTR:
		addr, err := netutil.IPFromReversedAddr(host)
		if err != nil {
			log.Debug("filtering: failed to parse PTR record %q: %s", host, err)

			return nil, nil, false
		}

		names := hs.ByAddr(addr)

		for _, name := range names {
			vals = append(vals, name)
			rls = append(rls, &ResultRule{
				Text:         fmt.Sprintf("%s %s", addr, name),
				FilterListID: rulelist.URLFilterIDEtcHosts,
			})
		}

		return vals, rls, len(names) > 0
	default:
		log.Debug("filtering: unsupported qtype %d", qtype)

		return nil, nil, false
	}

	addrs := hs.ByName(host)
	for _, addr := range addrs {
		if isValidProto(addr) {
			vals = append(vals, addr)
		}
		rls = append(rls, &ResultRule{
			Text:         fmt.Sprintf("%s %s", addr, host),
			FilterListID: rulelist.URLFilterIDEtcHosts,
		})
	}

	return vals, rls, len(addrs) > 0
}
