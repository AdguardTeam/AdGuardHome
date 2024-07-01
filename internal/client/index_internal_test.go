package client

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIDIndex is a helper function that returns a client index filled with
// persistent clients from the m.  It also generates a UID for each client.
func newIDIndex(m []*Persistent) (ci *index) {
	ci = newIndex()

	for _, c := range m {
		c.UID = MustNewUID()
		ci.add(c)
	}

	return ci
}

// TODO(s.chzhen):  Remove.
func TestClientIndex_Find(t *testing.T) {
	const (
		cliIPNone = "1.2.3.4"
		cliIP1    = "1.1.1.1"
		cliIP2    = "2.2.2.2"

		cliIPv6 = "1:2:3::4"

		cliSubnet   = "2.2.2.0/24"
		cliSubnetIP = "2.2.2.222"

		cliID  = "client-id"
		cliMAC = "11:11:11:11:11:11"

		linkLocalIP     = "fe80::abcd:abcd:abcd:ab%eth0"
		linkLocalSubnet = "fe80::/16"
	)

	var (
		clientWithBothFams = &Persistent{
			Name: "client1",
			IPs: []netip.Addr{
				netip.MustParseAddr(cliIP1),
				netip.MustParseAddr(cliIPv6),
			},
		}

		clientWithSubnet = &Persistent{
			Name:    "client2",
			IPs:     []netip.Addr{netip.MustParseAddr(cliIP2)},
			Subnets: []netip.Prefix{netip.MustParsePrefix(cliSubnet)},
		}

		clientWithMAC = &Persistent{
			Name: "client_with_mac",
			MACs: []net.HardwareAddr{mustParseMAC(cliMAC)},
		}

		clientWithID = &Persistent{
			Name:      "client_with_id",
			ClientIDs: []string{cliID},
		}

		clientLinkLocal = &Persistent{
			Name:    "client_link_local",
			Subnets: []netip.Prefix{netip.MustParsePrefix(linkLocalSubnet)},
		}
	)

	clients := []*Persistent{
		clientWithBothFams,
		clientWithSubnet,
		clientWithMAC,
		clientWithID,
		clientLinkLocal,
	}
	ci := newIDIndex(clients)

	testCases := []struct {
		want *Persistent
		name string
		ids  []string
	}{{
		name: "ipv4_ipv6",
		ids:  []string{cliIP1, cliIPv6},
		want: clientWithBothFams,
	}, {
		name: "ipv4_subnet",
		ids:  []string{cliIP2, cliSubnetIP},
		want: clientWithSubnet,
	}, {
		name: "mac",
		ids:  []string{cliMAC},
		want: clientWithMAC,
	}, {
		name: "client_id",
		ids:  []string{cliID},
		want: clientWithID,
	}, {
		name: "client_link_local_subnet",
		ids:  []string{linkLocalIP},
		want: clientLinkLocal,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range tc.ids {
				c, ok := ci.find(id)
				require.True(t, ok)

				assert.Equal(t, tc.want, c)
			}
		})
	}

	t.Run("not_found", func(t *testing.T) {
		_, ok := ci.find(cliIPNone)
		assert.False(t, ok)
	})
}

func TestClientIndex_Clashes(t *testing.T) {
	const (
		cliIP1      = "1.1.1.1"
		cliSubnet   = "2.2.2.0/24"
		cliSubnetIP = "2.2.2.222"
		cliID       = "client-id"
		cliMAC      = "11:11:11:11:11:11"
	)

	clients := []*Persistent{{
		Name: "client_with_ip",
		IPs:  []netip.Addr{netip.MustParseAddr(cliIP1)},
	}, {
		Name:    "client_with_subnet",
		Subnets: []netip.Prefix{netip.MustParsePrefix(cliSubnet)},
	}, {
		Name: "client_with_mac",
		MACs: []net.HardwareAddr{mustParseMAC(cliMAC)},
	}, {
		Name:      "client_with_id",
		ClientIDs: []string{cliID},
	}}

	ci := newIDIndex(clients)

	testCases := []struct {
		client *Persistent
		name   string
	}{{
		name:   "ipv4",
		client: clients[0],
	}, {
		name:   "subnet",
		client: clients[1],
	}, {
		name:   "mac",
		client: clients[2],
	}, {
		name:   "client_id",
		client: clients[3],
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clone := tc.client.ShallowClone()
			clone.UID = MustNewUID()

			err := ci.clashes(clone)
			require.Error(t, err)

			ci.remove(tc.client)
			err = ci.clashes(clone)
			require.NoError(t, err)
		})
	}
}

// mustParseMAC is wrapper around [net.ParseMAC] that panics if there is an
// error.
func mustParseMAC(s string) (mac net.HardwareAddr) {
	mac, err := net.ParseMAC(s)
	if err != nil {
		panic(err)
	}

	return mac
}

func TestMACToKey(t *testing.T) {
	testCases := []struct {
		want any
		name string
		in   string
	}{{
		name: "column6",
		in:   "00:00:5e:00:53:01",
		want: [6]byte(mustParseMAC("00:00:5e:00:53:01")),
	}, {
		name: "column8",
		in:   "02:00:5e:10:00:00:00:01",
		want: [8]byte(mustParseMAC("02:00:5e:10:00:00:00:01")),
	}, {
		name: "column20",
		in:   "00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01",
		want: [20]byte(mustParseMAC("00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01")),
	}, {
		name: "hyphen6",
		in:   "00-00-5e-00-53-01",
		want: [6]byte(mustParseMAC("00-00-5e-00-53-01")),
	}, {
		name: "hyphen8",
		in:   "02-00-5e-10-00-00-00-01",
		want: [8]byte(mustParseMAC("02-00-5e-10-00-00-00-01")),
	}, {
		name: "hyphen20",
		in:   "00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01",
		want: [20]byte(mustParseMAC("00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01")),
	}, {
		name: "dot6",
		in:   "0000.5e00.5301",
		want: [6]byte(mustParseMAC("0000.5e00.5301")),
	}, {
		name: "dot8",
		in:   "0200.5e10.0000.0001",
		want: [8]byte(mustParseMAC("0200.5e10.0000.0001")),
	}, {
		name: "dot20",
		in:   "0000.0000.fe80.0000.0000.0000.0200.5e10.0000.0001",
		want: [20]byte(mustParseMAC("0000.0000.fe80.0000.0000.0000.0200.5e10.0000.0001")),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mac := mustParseMAC(tc.in)

			key := macToKey(mac)
			assert.Equal(t, tc.want, key)
		})
	}

	assert.Panics(t, func() {
		mac := net.HardwareAddr([]byte{1, 2, 3})
		_ = macToKey(mac)
	})
}

func TestIndex_FindByIPWithoutZone(t *testing.T) {
	var (
		ip         = netip.MustParseAddr("fe80::a098:7654:32ef:ff1")
		ipWithZone = netip.MustParseAddr("fe80::1ff:fe23:4567:890a%eth2")
	)

	var (
		clientNoZone = &Persistent{
			Name: "client",
			IPs:  []netip.Addr{ip},
		}

		clientWithZone = &Persistent{
			Name: "client_with_zone",
			IPs:  []netip.Addr{ipWithZone},
		}
	)

	ci := newIDIndex([]*Persistent{
		clientNoZone,
		clientWithZone,
	})

	testCases := []struct {
		ip   netip.Addr
		want *Persistent
		name string
	}{{
		name: "without_zone",
		ip:   ip,
		want: clientNoZone,
	}, {
		name: "with_zone",
		ip:   ipWithZone,
		want: clientWithZone,
	}, {
		name: "zero_address",
		ip:   netip.Addr{},
		want: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := ci.findByIPWithoutZone(tc.ip.WithZone(""))
			require.Equal(t, tc.want, c)
		})
	}
}

func TestClientIndex_RangeByName(t *testing.T) {
	sortedClients := []*Persistent{{
		Name:      "clientA",
		ClientIDs: []string{"A"},
	}, {
		Name:      "clientB",
		ClientIDs: []string{"B"},
	}, {
		Name:      "clientC",
		ClientIDs: []string{"C"},
	}, {
		Name:      "clientD",
		ClientIDs: []string{"D"},
	}, {
		Name:      "clientE",
		ClientIDs: []string{"E"},
	}}

	testCases := []struct {
		name string
		want []*Persistent
	}{{
		name: "basic",
		want: sortedClients,
	}, {
		name: "nil",
		want: nil,
	}, {
		name: "one_element",
		want: sortedClients[:1],
	}, {
		name: "two_elements",
		want: sortedClients[:2],
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ci := newIDIndex(tc.want)

			var got []*Persistent
			ci.rangeByName(func(c *Persistent) (cont bool) {
				got = append(got, c)

				return true
			})

			assert.Equal(t, tc.want, got)
		})
	}
}
