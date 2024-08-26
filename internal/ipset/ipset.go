// Package ipset provides ipset functionality.
package ipset

import (
	"context"
	"log/slog"
	"net"
)

// Manager is the ipset manager interface.
//
// TODO(a.garipov): Perhaps generalize this into some kind of a NetFilter type,
// since ipset is exclusive to Linux?
type Manager interface {
	Add(ctx context.Context, host string, ip4s, ip6s []net.IP) (n int, err error)
	Close() (err error)
}

// Config is the configuration structure for the ipset manager.
type Config struct {
	// Logger is used for logging the operation of the ipset manager.  It must
	// not be nil.
	Logger *slog.Logger

	// Lines is the ipset configuration with the following syntax:
	//
	//	DOMAIN[,DOMAIN].../IPSET_NAME[,IPSET_NAME]...
	//
	// Lines must not contain any blank lines or comments.
	Lines []string
}

// NewManager returns a new ipset manager.  IPv4 addresses are added to an ipset
// with an ipv4 family; IPv6 addresses, to an ipv6 ipset.  ipset must exist.
//
// If conf.Lines is empty, mgr and err are nil.  The error's chain contains
// [errors.ErrUnsupported] if current OS is not supported.
func NewManager(ctx context.Context, conf *Config) (mgr Manager, err error) {
	if len(conf.Lines) == 0 {
		return nil, nil
	}

	return newManager(ctx, conf)
}
