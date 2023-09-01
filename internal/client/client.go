// Package client contains types and logic dealing with AdGuard Home's DNS
// clients.
//
// TODO(a.garipov): Expand.
package client

import (
	"encoding"
	"fmt"
)

// Source represents the source from which the information about the client has
// been obtained.
type Source uint8

// Clients information sources.  The order determines the priority.
const (
	SourceNone Source = iota
	SourceWHOIS
	SourceARP
	SourceRDNS
	SourceDHCP
	SourceHostsFile
	SourcePersistent
)

// type check
var _ fmt.Stringer = Source(0)

// String returns a human-readable name of cs.
func (cs Source) String() (s string) {
	switch cs {
	case SourceWHOIS:
		return "WHOIS"
	case SourceARP:
		return "ARP"
	case SourceRDNS:
		return "rDNS"
	case SourceDHCP:
		return "DHCP"
	case SourceHostsFile:
		return "etc/hosts"
	default:
		return ""
	}
}

// type check
var _ encoding.TextMarshaler = Source(0)

// MarshalText implements encoding.TextMarshaler for the Source.
func (cs Source) MarshalText() (text []byte, err error) {
	return []byte(cs.String()), nil
}
