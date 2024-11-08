package client

import (
	"fmt"
	"maps"
	"net"
	"net/netip"
	"slices"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/errors"
)

// macKey contains MAC as byte array of 6, 8, or 20 bytes.
type macKey any

// macToKey converts mac into key of type macKey, which is used as the key of
// the [clientIndex.macToUID].  mac must be valid MAC address.
func macToKey(mac net.HardwareAddr) (key macKey) {
	switch len(mac) {
	case 6:
		return [6]byte(mac)
	case 8:
		return [8]byte(mac)
	case 20:
		return [20]byte(mac)
	default:
		panic(fmt.Errorf("invalid mac address %#v", mac))
	}
}

// index stores all information about persistent clients.
type index struct {
	// nameToUID maps client name to UID.
	nameToUID map[string]UID

	// clientIDToUID maps client ID to UID.
	clientIDToUID map[string]UID

	// ipToUID maps IP address to UID.
	ipToUID map[netip.Addr]UID

	// macToUID maps MAC address to UID.
	macToUID map[macKey]UID

	// uidToClient maps UID to the persistent client.
	uidToClient map[UID]*Persistent

	// subnetToUID maps subnet to UID.
	subnetToUID aghalg.SortedMap[netip.Prefix, UID]
}

// newIndex initializes the new instance of client index.
func newIndex() (ci *index) {
	return &index{
		nameToUID:     map[string]UID{},
		clientIDToUID: map[string]UID{},
		ipToUID:       map[netip.Addr]UID{},
		subnetToUID:   aghalg.NewSortedMap[netip.Prefix, UID](subnetCompare),
		macToUID:      map[macKey]UID{},
		uidToClient:   map[UID]*Persistent{},
	}
}

// add stores information about a persistent client in the index.  c must be
// non-nil, have a UID, and contain at least one identifier.
func (ci *index) add(c *Persistent) {
	if (c.UID == UID{}) {
		panic("client must contain uid")
	}

	ci.nameToUID[c.Name] = c.UID

	for _, id := range c.ClientIDs {
		ci.clientIDToUID[id] = c.UID
	}

	for _, ip := range c.IPs {
		ci.ipToUID[ip] = c.UID
	}

	for _, pref := range c.Subnets {
		ci.subnetToUID.Set(pref, c.UID)
	}

	for _, mac := range c.MACs {
		k := macToKey(mac)
		ci.macToUID[k] = c.UID
	}

	ci.uidToClient[c.UID] = c
}

// clashesUID returns existing persistent client with the same UID as c.  Note
// that this is only possible when configuration contains duplicate fields.
func (ci *index) clashesUID(c *Persistent) (err error) {
	p, ok := ci.uidToClient[c.UID]
	if ok {
		return fmt.Errorf("another client %q uses the same uid", p.Name)
	}

	return nil
}

// clashes returns an error if the index contains a different persistent client
// with at least a single identifier contained by c.  c must be non-nil.
func (ci *index) clashes(c *Persistent) (err error) {
	if p := ci.clashesName(c); p != nil {
		return fmt.Errorf("another client uses the same name %q", p.Name)
	}

	for _, id := range c.ClientIDs {
		existing, ok := ci.clientIDToUID[id]
		if ok && existing != c.UID {
			p := ci.uidToClient[existing]

			return fmt.Errorf("another client %q uses the same ClientID %q", p.Name, id)
		}
	}

	p, ip := ci.clashesIP(c)
	if p != nil {
		return fmt.Errorf("another client %q uses the same IP %q", p.Name, ip)
	}

	p, s := ci.clashesSubnet(c)
	if p != nil {
		return fmt.Errorf("another client %q uses the same subnet %q", p.Name, s)
	}

	p, mac := ci.clashesMAC(c)
	if p != nil {
		return fmt.Errorf("another client %q uses the same MAC %q", p.Name, mac)
	}

	return nil
}

// clashesName returns existing persistent client with the same name as c or
// nil.  c must be non-nil.
func (ci *index) clashesName(c *Persistent) (existing *Persistent) {
	existing, ok := ci.findByName(c.Name)
	if !ok {
		return nil
	}

	if existing.UID != c.UID {
		return existing
	}

	return nil
}

// clashesIP returns a previous client with the same IP address as c.  c must be
// non-nil.
func (ci *index) clashesIP(c *Persistent) (p *Persistent, ip netip.Addr) {
	for _, ip := range c.IPs {
		existing, ok := ci.ipToUID[ip]
		if ok && existing != c.UID {
			return ci.uidToClient[existing], ip
		}
	}

	return nil, netip.Addr{}
}

// clashesSubnet returns a previous client with the same subnet as c.  c must be
// non-nil.
func (ci *index) clashesSubnet(c *Persistent) (p *Persistent, s netip.Prefix) {
	for _, s = range c.Subnets {
		var existing UID
		var ok bool

		ci.subnetToUID.Range(func(p netip.Prefix, uid UID) (cont bool) {
			if s == p {
				existing = uid
				ok = true

				return false
			}

			return true
		})

		if ok && existing != c.UID {
			return ci.uidToClient[existing], s
		}
	}

	return nil, netip.Prefix{}
}

// clashesMAC returns a previous client with the same MAC address as c.  c must
// be non-nil.
func (ci *index) clashesMAC(c *Persistent) (p *Persistent, mac net.HardwareAddr) {
	for _, mac = range c.MACs {
		k := macToKey(mac)
		existing, ok := ci.macToUID[k]
		if ok && existing != c.UID {
			return ci.uidToClient[existing], mac
		}
	}

	return nil, nil
}

// find finds persistent client by string representation of the client ID, IP
// address, or MAC.
func (ci *index) find(id string) (c *Persistent, ok bool) {
	uid, found := ci.clientIDToUID[id]
	if found {
		return ci.uidToClient[uid], true
	}

	ip, err := netip.ParseAddr(id)
	if err == nil {
		// MAC addresses can be successfully parsed as IP addresses.
		c, found = ci.findByIP(ip)
		if found {
			return c, true
		}
	}

	mac, err := net.ParseMAC(id)
	if err == nil {
		return ci.findByMAC(mac)
	}

	return nil, false
}

// findByName finds persistent client by name.
func (ci *index) findByName(name string) (c *Persistent, found bool) {
	uid, found := ci.nameToUID[name]
	if found {
		return ci.uidToClient[uid], true
	}

	return nil, false
}

// findByIP finds persistent client by IP address.
func (ci *index) findByIP(ip netip.Addr) (c *Persistent, found bool) {
	uid, found := ci.ipToUID[ip]
	if found {
		return ci.uidToClient[uid], true
	}

	ipWithoutZone := ip.WithZone("")
	ci.subnetToUID.Range(func(pref netip.Prefix, id UID) (cont bool) {
		// Remove zone before checking because prefixes strip zones.
		if pref.Contains(ipWithoutZone) {
			uid, found = id, true

			return false
		}

		return true
	})

	if found {
		return ci.uidToClient[uid], true
	}

	return nil, false
}

// findByMAC finds persistent client by MAC.
func (ci *index) findByMAC(mac net.HardwareAddr) (c *Persistent, found bool) {
	k := macToKey(mac)
	uid, found := ci.macToUID[k]
	if found {
		return ci.uidToClient[uid], true
	}

	return nil, false
}

// findByIPWithoutZone finds a persistent client by IP address without zone.  It
// strips the IPv6 zone index from the stored IP addresses before comparing,
// because querylog entries don't have it.  See TODO on [querylog.logEntry.IP].
//
// Note that multiple clients can have the same IP address with different zones.
// Therefore, the result of this method is indeterminate.
func (ci *index) findByIPWithoutZone(ip netip.Addr) (c *Persistent) {
	if (ip == netip.Addr{}) {
		return nil
	}

	for addr, uid := range ci.ipToUID {
		if addr.WithZone("") == ip {
			return ci.uidToClient[uid]
		}
	}

	return nil
}

// remove removes information about persistent client from the index.  c must be
// non-nil.
func (ci *index) remove(c *Persistent) {
	delete(ci.nameToUID, c.Name)

	for _, id := range c.ClientIDs {
		delete(ci.clientIDToUID, id)
	}

	for _, ip := range c.IPs {
		delete(ci.ipToUID, ip)
	}

	for _, pref := range c.Subnets {
		ci.subnetToUID.Del(pref)
	}

	for _, mac := range c.MACs {
		k := macToKey(mac)
		delete(ci.macToUID, k)
	}

	delete(ci.uidToClient, c.UID)
}

// size returns the number of persistent clients.
func (ci *index) size() (n int) {
	return len(ci.uidToClient)
}

// rangeByName is like [Index.Range] but sorts the persistent clients by name
// before iterating ensuring a predictable order.
func (ci *index) rangeByName(f func(c *Persistent) (cont bool)) {
	clients := slices.SortedStableFunc(
		maps.Values(ci.uidToClient),
		func(a, b *Persistent) (res int) {
			return strings.Compare(a.Name, b.Name)
		},
	)

	for _, c := range clients {
		if !f(c) {
			break
		}
	}
}

// closeUpstreams closes upstream configurations of persistent clients.
func (ci *index) closeUpstreams() (err error) {
	var errs []error
	ci.rangeByName(func(c *Persistent) (cont bool) {
		err = c.CloseUpstreams()
		if err != nil {
			errs = append(errs, err)
		}

		return true
	})

	return errors.Join(errs...)
}
