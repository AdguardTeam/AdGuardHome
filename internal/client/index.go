package client

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
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
		clientIDToUID: map[string]UID{},
		ipToUID:       map[netip.Addr]UID{},
		subnetToUID:   aghalg.NewSortedMap[netip.Prefix, UID](subnetCompare),
		macToUID:      map[macKey]UID{},
		uidToClient:   map[UID]*Persistent{},
	}
}

// Add stores information about a persistent client in the index.  c must be
// non-nil and contain UID.
func (ci *Index) Add(c *Persistent) {
	if (c.UID == UID{}) {
		panic("client must contain uid")
	}

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

// Clashes returns an error if the index contains a different persistent client
// with at least a single identifier contained by c.  c must be non-nil.
func (ci *Index) Clashes(c *Persistent) (err error) {
	for _, id := range c.ClientIDs {
		existing, ok := ci.clientIDToUID[id]
		if ok && existing != c.UID {
			p := ci.uidToClient[existing]

			return fmt.Errorf("another client %q uses the same ID %q", p.Name, id)
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
		return ci.findByMAC(mac)
	}

	return nil, false
}

// find finds persistent client by IP address.
func (ci *Index) findByIP(ip netip.Addr) (c *Persistent, found bool) {
	uid, found := ci.ipToUID[ip]
	if found {
		return ci.uidToClient[uid], true
	}

	ci.subnetToUID.Range(func(pref netip.Prefix, id UID) (cont bool) {
		if pref.Contains(ip) {
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

// find finds persistent client by MAC.
func (ci *Index) findByMAC(mac net.HardwareAddr) (c *Persistent, found bool) {
	k := macToKey(mac)
	uid, found := ci.macToUID[k]
	if found {
		return ci.uidToClient[uid], true
	}

	return nil, false
}

// Delete removes information about persistent client from the index.  c must be
// non-nil.
func (ci *Index) Delete(c *Persistent) {
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
