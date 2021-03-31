package aghtest

import (
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
)

// Exchanger is a mock aghnet.Exchanger implementation for tests.
type Exchanger struct {
	Ups upstream.Upstream
}

// Exchange implements aghnet.Exchanger interface for *Exchanger.
func (lr *Exchanger) Exchange(req *dns.Msg) (resp *dns.Msg, err error) {
	if lr.Ups == nil {
		lr.Ups = &TestErrUpstream{}
	}

	return lr.Ups.Exchange(req)
}
