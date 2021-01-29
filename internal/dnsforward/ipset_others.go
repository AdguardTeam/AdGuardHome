// +build !linux

package dnsforward

import (
	"github.com/AdguardTeam/golibs/log"
)

type ipsetCtx struct{}

// init initializes the ipset context.
func (c *ipsetCtx) init(ipsetConfig []string) (err error) {
	if len(ipsetConfig) != 0 {
		log.Info("ipset: only available on linux")
	}

	return nil
}

// process adds the resolved IP addresses to the domain's ipsets, if any.
func (c *ipsetCtx) process(_ *dnsContext) (rc resultCode) {
	return resultCodeSuccess
}

// Close closes the Linux Netfilter connections.
func (c *ipsetCtx) Close() (_ error) { return nil }
