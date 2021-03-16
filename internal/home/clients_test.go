package home

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClients(t *testing.T) {
	clients := clientsContainer{}
	clients.testing = true

	clients.Init(nil, nil, nil)

	t.Run("add_success", func(t *testing.T) {
		c := &Client{
			IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa"},
			Name: "client1",
		}

		ok, err := clients.Add(c)
		require.Nil(t, err)
		assert.True(t, ok)

		c = &Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client2",
		}

		ok, err = clients.Add(c)
		require.Nil(t, err)
		assert.True(t, ok)

		c, ok = clients.Find("1.1.1.1")
		require.True(t, ok)
		assert.Equal(t, "client1", c.Name)

		c, ok = clients.Find("1:2:3::4")
		require.True(t, ok)
		assert.Equal(t, "client1", c.Name)

		c, ok = clients.Find("2.2.2.2")
		require.True(t, ok)
		assert.Equal(t, "client2", c.Name)

		assert.False(t, clients.Exists("1.2.3.4", ClientSourceHostsFile))
		assert.True(t, clients.Exists("1.1.1.1", ClientSourceHostsFile))
		assert.True(t, clients.Exists("2.2.2.2", ClientSourceHostsFile))
	})

	t.Run("add_fail_name", func(t *testing.T) {
		ok, err := clients.Add(&Client{
			IDs:  []string{"1.2.3.5"},
			Name: "client1",
		})
		require.Nil(t, err)
		assert.False(t, ok)
	})

	t.Run("add_fail_ip", func(t *testing.T) {
		ok, err := clients.Add(&Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client3",
		})
		require.NotNil(t, err)
		assert.False(t, ok)
	})

	t.Run("update_fail_name", func(t *testing.T) {
		err := clients.Update("client3", &Client{
			IDs:  []string{"1.2.3.0"},
			Name: "client3",
		})
		require.NotNil(t, err)

		err = clients.Update("client3", &Client{
			IDs:  []string{"1.2.3.0"},
			Name: "client2",
		})
		assert.NotNil(t, err)
	})

	t.Run("update_fail_ip", func(t *testing.T) {
		err := clients.Update("client1", &Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client1",
		})
		assert.NotNil(t, err)
	})

	t.Run("update_success", func(t *testing.T) {
		err := clients.Update("client1", &Client{
			IDs:  []string{"1.1.1.2"},
			Name: "client1",
		})
		require.Nil(t, err)

		assert.False(t, clients.Exists("1.1.1.1", ClientSourceHostsFile))
		assert.True(t, clients.Exists("1.1.1.2", ClientSourceHostsFile))

		err = clients.Update("client1", &Client{
			IDs:            []string{"1.1.1.2"},
			Name:           "client1-renamed",
			UseOwnSettings: true,
		})
		require.Nil(t, err)

		c, ok := clients.Find("1.1.1.2")
		require.True(t, ok)
		assert.Equal(t, "client1-renamed", c.Name)
		assert.True(t, c.UseOwnSettings)

		nilCli, ok := clients.list["client1"]
		require.False(t, ok)
		assert.Nil(t, nilCli)

		require.Len(t, c.IDs, 1)
		assert.Equal(t, "1.1.1.2", c.IDs[0])
	})

	t.Run("del_success", func(t *testing.T) {
		ok := clients.Del("client1-renamed")
		require.True(t, ok)
		assert.False(t, clients.Exists("1.1.1.2", ClientSourceHostsFile))
	})

	t.Run("del_fail", func(t *testing.T) {
		ok := clients.Del("client3")
		assert.False(t, ok)
	})

	t.Run("addhost_success", func(t *testing.T) {
		ok, err := clients.AddHost("1.1.1.1", "host", ClientSourceARP)
		require.Nil(t, err)
		assert.True(t, ok)

		ok, err = clients.AddHost("1.1.1.1", "host2", ClientSourceARP)
		require.Nil(t, err)
		assert.True(t, ok)

		ok, err = clients.AddHost("1.1.1.1", "host3", ClientSourceHostsFile)
		require.Nil(t, err)
		assert.True(t, ok)

		assert.True(t, clients.Exists("1.1.1.1", ClientSourceHostsFile))
	})

	t.Run("addhost_fail", func(t *testing.T) {
		ok, err := clients.AddHost("1.1.1.1", "host1", ClientSourceRDNS)
		require.Nil(t, err)
		assert.False(t, ok)
	})
}

func TestClientsWhois(t *testing.T) {
	clients := clientsContainer{
		testing: true,
	}
	clients.Init(nil, nil, nil)
	whois := [][]string{{"orgname", "orgname-val"}, {"country", "country-val"}}

	t.Run("new_client", func(t *testing.T) {
		clients.SetWhoisInfo("1.1.1.255", whois)

		require.NotNil(t, clients.ipHost["1.1.1.255"])
		h := clients.ipHost["1.1.1.255"]

		require.Len(t, h.WhoisInfo, 2)
		require.Len(t, h.WhoisInfo[0], 2)
		assert.Equal(t, "orgname-val", h.WhoisInfo[0][1])
	})

	t.Run("existing_auto-client", func(t *testing.T) {
		ok, err := clients.AddHost("1.1.1.1", "host", ClientSourceRDNS)
		require.Nil(t, err)
		assert.True(t, ok)

		clients.SetWhoisInfo("1.1.1.1", whois)

		require.NotNil(t, clients.ipHost["1.1.1.1"])
		h := clients.ipHost["1.1.1.1"]

		require.Len(t, h.WhoisInfo, 2)
		require.Len(t, h.WhoisInfo[0], 2)
		assert.Equal(t, "orgname-val", h.WhoisInfo[0][1])
	})

	t.Run("can't_set_manually-added", func(t *testing.T) {
		ok, err := clients.Add(&Client{
			IDs:  []string{"1.1.1.2"},
			Name: "client1",
		})
		require.Nil(t, err)
		assert.True(t, ok)

		clients.SetWhoisInfo("1.1.1.2", whois)
		require.Nil(t, clients.ipHost["1.1.1.2"])
		assert.True(t, clients.Del("client1"))
	})
}

func TestClientsAddExisting(t *testing.T) {
	clients := clientsContainer{
		testing: true,
	}
	clients.Init(nil, nil, nil)

	t.Run("simple", func(t *testing.T) {
		// Add a client.
		ok, err := clients.Add(&Client{
			IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa", "2.2.2.0/24"},
			Name: "client1",
		})
		require.Nil(t, err)
		assert.True(t, ok)

		// Now add an auto-client with the same IP.
		ok, err = clients.AddHost("1.1.1.1", "test", ClientSourceRDNS)
		require.Nil(t, err)
		assert.True(t, ok)
	})

	t.Run("complicated", func(t *testing.T) {
		testIP := net.IP{1, 2, 3, 4}

		// First, init a DHCP server with a single static lease.
		config := dhcpd.ServerConfig{
			Enabled:    true,
			DBFilePath: "leases.db",
			Conf4: dhcpd.V4ServerConf{
				Enabled:    true,
				GatewayIP:  net.IP{1, 2, 3, 1},
				SubnetMask: net.IP{255, 255, 255, 0},
				RangeStart: net.IP{1, 2, 3, 2},
				RangeEnd:   net.IP{1, 2, 3, 10},
			},
		}

		clients.dhcpServer = dhcpd.Create(config)
		t.Cleanup(func() { _ = os.Remove("leases.db") })

		err := clients.dhcpServer.AddStaticLease(dhcpd.Lease{
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       testIP,
			Hostname: "testhost",
			Expiry:   time.Now().Add(time.Hour),
		})
		require.Nil(t, err)

		// Add a new client with the same IP as for a client with MAC.
		ok, err := clients.Add(&Client{
			IDs:  []string{testIP.String()},
			Name: "client2",
		})
		require.Nil(t, err)
		assert.True(t, ok)

		// Add a new client with the IP from the first client's IP
		// range.
		ok, err = clients.Add(&Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client3",
		})
		require.Nil(t, err)
		assert.True(t, ok)
	})
}

func TestClientsCustomUpstream(t *testing.T) {
	clients := clientsContainer{
		testing: true,
	}
	clients.Init(nil, nil, nil)

	// Add client with upstreams.
	ok, err := clients.Add(&Client{
		IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa"},
		Name: "client1",
		Upstreams: []string{
			"1.1.1.1",
			"[/example.org/]8.8.8.8",
		},
	})
	require.Nil(t, err)
	assert.True(t, ok)

	config := clients.FindUpstreams("1.2.3.4")
	assert.Nil(t, config)

	config = clients.FindUpstreams("1.1.1.1")
	require.NotNil(t, config)
	assert.Len(t, config.Upstreams, 1)
	assert.Len(t, config.DomainReservedUpstreams, 1)
}
