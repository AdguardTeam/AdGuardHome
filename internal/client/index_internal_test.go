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
func newIDIndex(m []*Persistent) (ci *Index) {
	ci = NewIndex()

	for _, c := range m {
		c.UID = MustNewUID()
		ci.Add(c)
	}

	return ci
}

func TestClientIndex(t *testing.T) {
	const (
		cliIPNone = "1.2.3.4"
		cliIP1    = "1.1.1.1"
		cliIP2    = "2.2.2.2"

		cliIPv6 = "1:2:3::4"

		cliSubnet   = "2.2.2.0/24"
		cliSubnetIP = "2.2.2.222"

		cliID  = "client-id"
		cliMAC = "11:11:11:11:11:11"
	)

	clients := []*Persistent{{
		Name: "client1",
		IPs: []netip.Addr{
			netip.MustParseAddr(cliIP1),
			netip.MustParseAddr(cliIPv6),
		},
	}, {
		Name:    "client2",
		IPs:     []netip.Addr{netip.MustParseAddr(cliIP2)},
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
		want *Persistent
		name string
		ids  []string
	}{{
		name: "ipv4_ipv6",
		ids:  []string{cliIP1, cliIPv6},
		want: clients[0],
	}, {
		name: "ipv4_subnet",
		ids:  []string{cliIP2, cliSubnetIP},
		want: clients[1],
	}, {
		name: "mac",
		ids:  []string{cliMAC},
		want: clients[2],
	}, {
		name: "client_id",
		ids:  []string{cliID},
		want: clients[3],
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range tc.ids {
				c, ok := ci.Find(id)
				require.True(t, ok)

				assert.Equal(t, tc.want, c)
			}
		})
	}

	t.Run("not_found", func(t *testing.T) {
		_, ok := ci.Find(cliIPNone)
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

			err := ci.Clashes(clone)
			require.Error(t, err)

			ci.Delete(tc.client)
			err = ci.Clashes(clone)
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
