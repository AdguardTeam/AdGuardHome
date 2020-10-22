// +build !linux

package dnsforward

import (
	"github.com/AdguardTeam/golibs/log"
)

type ipsetCtx struct {}

// Convert configuration settings to an internal map and check ipsets
// DOMAIN[,DOMAIN].../IPSET1_NAME[,IPSET2_NAME]...
// config parameter may be nil
func (c *ipsetCtx) init(ipsetConfig []string, config *interface{}) error {
	if len(ipsetConfig) != 0 {
		log.Info("IPSET: ignoring %d ipset configuration lines; " +
			"ipset support is only available on Linux",
			len(ipsetConfig))
	}
	return nil	
}

func (c *ipsetCtx) process(ctx *dnsContext) int {
	return resultDone
}
