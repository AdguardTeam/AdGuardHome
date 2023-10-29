// Package ipset provides ipset functionality.
package ipset

import (
	"net"
)

// Manager is the ipset manager interface.
//
// TODO(a.garipov): Perhaps generalize this into some kind of a NetFilter type,
// since ipset is exclusive to Linux?
type Manager interface {
	Add(host string, ip4s, ip6s []net.IP) (n int, err error)
	Close() (err error)
}

// NewManager returns a new ipset manager.  IPv4 addresses are added to an
// ipset with an ipv4 family; IPv6 addresses, to an ipv6 ipset.  ipset must
// exist.
//
// The syntax of the ipsetConf is:
//
//	DOMAIN[,DOMAIN].../IPSET_NAME[,IPSET_NAME]...
//
// If ipsetConf is empty, msg and err are nil.  The error is of type
// *aghos.UnsupportedError if the OS is not supported.
func NewManager(ipsetConf []string) (mgr Manager, err error) {
	if len(ipsetConf) == 0 {
		return nil, nil
	}

	return newManager(ipsetConf)
}
