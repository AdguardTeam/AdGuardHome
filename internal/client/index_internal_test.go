package client

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
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
			MACs: []net.HardwareAddr{errors.Must(net.ParseMAC(cliMAC))},
		}

		clientWithID = &Persistent{
			Name:      "client_with_id",
			ClientIDs: []ClientID{cliID},
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
		MACs: []net.HardwareAddr{errors.Must(net.ParseMAC(cliMAC))},
	}, {
		Name:      "client_with_id",
		ClientIDs: []ClientID{cliID},
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

func TestMACToKey(t *testing.T) {
	testCases := []struct {
		want any
		name string
		in   string
	}{{
		name: "column6",
		in:   "00:00:5e:00:53:01",
		want: [6]byte(errors.Must(net.ParseMAC("00:00:5e:00:53:01"))),
	}, {
		name: "column8",
		in:   "02:00:5e:10:00:00:00:01",
		want: [8]byte(errors.Must(net.ParseMAC("02:00:5e:10:00:00:00:01"))),
	}, {
		name: "column20",
		in:   "00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01",
		want: [20]byte(errors.Must(net.ParseMAC("00:00:00:00:fe:80:00:00:00:00:00:00:02:00:5e:10:00:00:00:01"))),
	}, {
		name: "hyphen6",
		in:   "00-00-5e-00-53-01",
		want: [6]byte(errors.Must(net.ParseMAC("00-00-5e-00-53-01"))),
	}, {
		name: "hyphen8",
		in:   "02-00-5e-10-00-00-00-01",
		want: [8]byte(errors.Must(net.ParseMAC("02-00-5e-10-00-00-00-01"))),
	}, {
		name: "hyphen20",
		in:   "00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01",
		want: [20]byte(errors.Must(net.ParseMAC("00-00-00-00-fe-80-00-00-00-00-00-00-02-00-5e-10-00-00-00-01"))),
	}, {
		name: "dot6",
		in:   "0000.5e00.5301",
		want: [6]byte(errors.Must(net.ParseMAC("0000.5e00.5301"))),
	}, {
		name: "dot8",
		in:   "0200.5e10.0000.0001",
		want: [8]byte(errors.Must(net.ParseMAC("0200.5e10.0000.0001"))),
	}, {
		name: "dot20",
		in:   "0000.0000.fe80.0000.0000.0000.0200.5e10.0000.0001",
		want: [20]byte(errors.Must(net.ParseMAC("0000.0000.fe80.0000.0000.0000.0200.5e10.0000.0001"))),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mac := errors.Must(net.ParseMAC(tc.in))

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
		ClientIDs: []ClientID{"A"},
	}, {
		Name:      "clientB",
		ClientIDs: []ClientID{"B"},
	}, {
		Name:      "clientC",
		ClientIDs: []ClientID{"C"},
	}, {
		Name:      "clientD",
		ClientIDs: []ClientID{"D"},
	}, {
		Name:      "clientE",
		ClientIDs: []ClientID{"E"},
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

func TestIndex_FindByName(t *testing.T) {
	const (
		clientExistingName        = "client_existing"
		clientAnotherExistingName = "client_another_existing"
		nonExistingClientName     = "client_non_existing"
	)

	var (
		clientExisting = &Persistent{
			Name: clientExistingName,
			IPs:  []netip.Addr{netip.MustParseAddr("192.0.2.1")},
		}

		clientAnotherExisting = &Persistent{
			Name: clientAnotherExistingName,
			IPs:  []netip.Addr{netip.MustParseAddr("192.0.2.2")},
		}
	)

	clients := []*Persistent{
		clientExisting,
		clientAnotherExisting,
	}
	ci := newIDIndex(clients)

	testCases := []struct {
		want       *Persistent
		found      assert.BoolAssertionFunc
		name       string
		clientName string
	}{{
		want:       clientExisting,
		found:      assert.True,
		name:       "existing",
		clientName: clientExistingName,
	}, {
		want:       clientAnotherExisting,
		found:      assert.True,
		name:       "another_existing",
		clientName: clientAnotherExistingName,
	}, {
		want:       nil,
		found:      assert.False,
		name:       "non_existing",
		clientName: nonExistingClientName,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := ci.findByName(tc.clientName)
			assert.Equal(t, tc.want, c)
			tc.found(t, ok)
		})
	}
}

func TestIndex_FindByMAC(t *testing.T) {
	var (
		cliMAC               = errors.Must(net.ParseMAC("11:11:11:11:11:11"))
		cliAnotherMAC        = errors.Must(net.ParseMAC("22:22:22:22:22:22"))
		nonExistingClientMAC = errors.Must(net.ParseMAC("33:33:33:33:33:33"))
	)

	var (
		clientExisting = &Persistent{
			Name: "client",
			MACs: []net.HardwareAddr{cliMAC},
		}

		clientAnotherExisting = &Persistent{
			Name: "another_client",
			MACs: []net.HardwareAddr{cliAnotherMAC},
		}
	)

	clients := []*Persistent{
		clientExisting,
		clientAnotherExisting,
	}
	ci := newIDIndex(clients)

	testCases := []struct {
		want      *Persistent
		found     assert.BoolAssertionFunc
		name      string
		clientMAC net.HardwareAddr
	}{{
		want:      clientExisting,
		found:     assert.True,
		name:      "existing",
		clientMAC: cliMAC,
	}, {
		want:      clientAnotherExisting,
		found:     assert.True,
		name:      "another_existing",
		clientMAC: cliAnotherMAC,
	}, {
		want:      nil,
		found:     assert.False,
		name:      "non_existing",
		clientMAC: nonExistingClientMAC,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := ci.findByMAC(tc.clientMAC)
			assert.Equal(t, tc.want, c)
			tc.found(t, ok)
		})
	}
}
