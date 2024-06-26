package client

import (
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/errors"
	"golang.org/x/exp/maps"
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

// Index stores all information about persistent clients.
type Index struct {
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

// NewIndex initializes the new instance of client index.
func NewIndex() (ci *Index) {
	return &Index{
		nameToUID:     map[string]UID{},
		clientIDToUID: map[string]UID{},
		ipToUID:       map[netip.Addr]UID{},
		subnetToUID:   aghalg.NewSortedMap[netip.Prefix, UID](subnetCompare),
		macToUID:      map[macKey]UID{},
		uidToClient:   map[UID]*Persistent{},
	}
}

// Add stores information about a persistent client in the index.  c must be
// non-nil, have a UID, and contain at least one identifier.
func (ci *Index) Add(c *Persistent) {
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

// ClashesUID returns existing persistent client with the same UID as c.  Note
// that this is only possible when configuration contains duplicate fields.
func (ci *Index) ClashesUID(c *Persistent) (err error) {
	p, ok := ci.uidToClient[c.UID]
	if ok {
		return fmt.Errorf("another client %q uses the same uid", p.Name)
	}

	return nil
}

// Clashes returns an error if the index contains a different persistent client
// with at least a single identifier contained by c.  c must be non-nil.
func (ci *Index) Clashes(c *Persistent) (err error) {
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
func (ci *Index) clashesName(c *Persistent) (existing *Persistent) {
	existing, ok := ci.FindByName(c.Name)
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
func (ci *Index) clashesIP(c *Persistent) (p *Persistent, ip netip.Addr) {
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
func (ci *Index) clashesSubnet(c *Persistent) (p *Persistent, s netip.Prefix) {
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
func (ci *Index) clashesMAC(c *Persistent) (p *Persistent, mac net.HardwareAddr) {
	for _, mac = range c.MACs {
		k := macToKey(mac)
		existing, ok := ci.macToUID[k]
		if ok && existing != c.UID {
			return ci.uidToClient[existing], mac
		}
	}

	return nil, nil
}

// Find finds persistent client by string representation of the client ID, IP
// address, or MAC.
func (ci *Index) Find(id string) (c *Persistent, ok bool) {
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
		return ci.FindByMAC(mac)
	}

	return nil, false
}

// FindByName finds persistent client by name.
func (ci *Index) FindByName(name string) (c *Persistent, found bool) {
	uid, found := ci.nameToUID[name]
	if found {
		return ci.uidToClient[uid], true
	}

	return nil, false
}

// findByIP finds persistent client by IP address.
func (ci *Index) findByIP(ip netip.Addr) (c *Persistent, found bool) {
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

// FindByMAC finds persistent client by MAC.
func (ci *Index) FindByMAC(mac net.HardwareAddr) (c *Persistent, found bool) {
	k := macToKey(mac)
	uid, found := ci.macToUID[k]
	if found {
		return ci.uidToClient[uid], true
	}

	return nil, false
}

// FindByIPWithoutZone finds a persistent client by IP address without zone.  It
// strips the IPv6 zone index from the stored IP addresses before comparing,
// because querylog entries don't have it.  See TODO on [querylog.logEntry.IP].
//
// Note that multiple clients can have the same IP address with different zones.
// Therefore, the result of this method is indeterminate.
func (ci *Index) FindByIPWithoutZone(ip netip.Addr) (c *Persistent) {
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

// Delete removes information about persistent client from the index.  c must be
// non-nil.
func (ci *Index) Delete(c *Persistent) {
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

// Size returns the number of persistent clients.
func (ci *Index) Size() (n int) {
	return len(ci.uidToClient)
}

// Range calls f for each persistent client, unless cont is false.  The order is
// undefined.
func (ci *Index) Range(f func(c *Persistent) (cont bool)) {
	for _, c := range ci.uidToClient {
		if !f(c) {
			return
		}
	}
}

// RangeByName is like [Index.Range] but sorts the persistent clients by name
// before iterating ensuring a predictable order.
func (ci *Index) RangeByName(f func(c *Persistent) (cont bool)) {
	cs := maps.Values(ci.uidToClient)
	slices.SortFunc(cs, func(a, b *Persistent) (n int) {
		return strings.Compare(a.Name, b.Name)
	})

	for _, c := range cs {
		if !f(c) {
			break
		}
	}
}

// CloseUpstreams closes upstream configurations of persistent clients.
func (ci *Index) CloseUpstreams() (err error) {
	var errs []error
	ci.RangeByName(func(c *Persistent) (cont bool) {
		err = c.CloseUpstreams()
		if err != nil {
			errs = append(errs, err)
		}

		return true
	})

	return errors.Join(errs...)
}
