// Package client contains types and logic dealing with AdGuard Home's DNS
// clients.
//
// TODO(a.garipov): Expand.
package client

import (
	"encoding"
	"fmt"
	"net/netip"
	"slices"

	"github.com/AdguardTeam/AdGuardHome/internal/whois"
)

// Source represents the source from which the information about the client has
// been obtained.
type Source uint8

// Clients information sources.  The order determines the priority.
const (
	SourceWHOIS Source = iota + 1
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

// Runtime is a client information from different sources.
type Runtime struct {
	// ip is an IP address of a client.
	ip netip.Addr

	// whois is the filtered WHOIS information of a client.
	whois *whois.Info

	// arp is the ARP information of a client.  nil indicates that there is no
	// information from the source.  Empty non-nil slice indicates that the data
	// from the source is present, but empty.
	arp []string

	// rdns is the RDNS information of a client.  nil indicates that there is no
	// information from the source.  Empty non-nil slice indicates that the data
	// from the source is present, but empty.
	rdns []string

	// dhcp is the DHCP information of a client.  nil indicates that there is no
	// information from the source.  Empty non-nil slice indicates that the data
	// from the source is present, but empty.
	dhcp []string

	// hostsFile is the information from the hosts file.  nil indicates that
	// there is no information from the source.  Empty non-nil slice indicates
	// that the data from the source is present, but empty.
	hostsFile []string
}

// NewRuntime constructs a new runtime client.  ip must be valid IP address.
//
// TODO(s.chzhen):  Validate IP address.
func NewRuntime(ip netip.Addr) (r *Runtime) {
	return &Runtime{
		ip: ip,
	}
}

// Info returns a client information from the highest-priority source.
func (r *Runtime) Info() (cs Source, host string) {
	info := []string{}

	switch {
	case r.hostsFile != nil:
		cs, info = SourceHostsFile, r.hostsFile
	case r.dhcp != nil:
		cs, info = SourceDHCP, r.dhcp
	case r.rdns != nil:
		cs, info = SourceRDNS, r.rdns
	case r.arp != nil:
		cs, info = SourceARP, r.arp
	case r.whois != nil:
		cs = SourceWHOIS
	}

	if len(info) == 0 {
		return cs, ""
	}

	// TODO(s.chzhen):  Return the full information.
	return cs, info[0]
}

// setInfo sets a host as a client information from the cs.
func (r *Runtime) setInfo(cs Source, hosts []string) {
	// TODO(s.chzhen):  Use contract where hosts must contain non-empty host.
	if len(hosts) == 1 && hosts[0] == "" {
		hosts = []string{}
	}

	switch cs {
	case SourceARP:
		r.arp = hosts
	case SourceRDNS:
		r.rdns = hosts
	case SourceDHCP:
		r.dhcp = hosts
	case SourceHostsFile:
		r.hostsFile = hosts
	}
}

// WHOIS returns a copy of WHOIS client information.
func (r *Runtime) WHOIS() (info *whois.Info) {
	return r.whois.Clone()
}

// setWHOIS sets a WHOIS client information.  info must be non-nil.
func (r *Runtime) setWHOIS(info *whois.Info) {
	r.whois = info
}

// unset clears a cs information.
func (r *Runtime) unset(cs Source) {
	switch cs {
	case SourceWHOIS:
		r.whois = nil
	case SourceARP:
		r.arp = nil
	case SourceRDNS:
		r.rdns = nil
	case SourceDHCP:
		r.dhcp = nil
	case SourceHostsFile:
		r.hostsFile = nil
	}
}

// isEmpty returns true if there is no information from any source.
func (r *Runtime) isEmpty() (ok bool) {
	return r.whois == nil &&
		r.arp == nil &&
		r.rdns == nil &&
		r.dhcp == nil &&
		r.hostsFile == nil
}

// Addr returns an IP address of the client.
func (r *Runtime) Addr() (ip netip.Addr) {
	return r.ip
}

// clone returns a deep copy of the runtime client.
func (r *Runtime) clone() (c *Runtime) {
	return &Runtime{
		ip:        r.ip,
		whois:     r.whois.Clone(),
		arp:       slices.Clone(r.arp),
		rdns:      slices.Clone(r.rdns),
		dhcp:      slices.Clone(r.dhcp),
		hostsFile: slices.Clone(r.hostsFile),
	}
}
