package aghnet

import (
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
)

// This package is not the best place for this functionality, but we put it here
// since we need to use it in both rDNS (home) and dnsServer (dnsforward).

// NoUpstreamsErr should be returned when there are no upstreams inside
// Exchanger implementation.
const NoUpstreamsErr agherr.Error = "no upstreams specified"

// Exchanger represents an object able to resolve DNS messages.
//
// TODO(e.burkov): Maybe expand with method like ExchangeParallel to be able to
// use user's upstream mode settings.  Also, think about Update method to
// refresh the internal state.
type Exchanger interface {
	Exchange(req *dns.Msg) (resp *dns.Msg, err error)
}

// multiAddrExchanger is the default implementation of Exchanger interface.
type multiAddrExchanger struct {
	ups []upstream.Upstream
}

// NewMultiAddrExchanger creates an Exchanger instance from passed addresses.
// It returns an error if any of addrs failed to become an upstream.
func NewMultiAddrExchanger(addrs []string, timeout time.Duration) (e Exchanger, err error) {
	defer agherr.Annotate("exchanger: %w", &err)

	if len(addrs) == 0 {
		return &multiAddrExchanger{}, nil
	}

	var ups []upstream.Upstream = make([]upstream.Upstream, 0, len(addrs))
	for _, addr := range addrs {
		var u upstream.Upstream
		u, err = upstream.AddressToUpstream(addr, upstream.Options{Timeout: timeout})
		if err != nil {
			return nil, err
		}

		ups = append(ups, u)
	}

	return &multiAddrExchanger{ups: ups}, nil
}

// Exсhange performs a query to each resolver until first response.
func (e *multiAddrExchanger) Exchange(req *dns.Msg) (resp *dns.Msg, err error) {
	defer agherr.Annotate("exchanger: %w", &err)

	// TODO(e.burkov): Maybe prohibit the initialization without upstreams.
	if len(e.ups) == 0 {
		return nil, NoUpstreamsErr
	}

	var errs []error
	for _, u := range e.ups {
		resp, err = u.Exchange(req)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if resp != nil {
			return resp, nil
		}
	}

	return nil, agherr.Many("can't exchange", errs...)
}
