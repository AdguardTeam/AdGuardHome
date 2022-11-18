package home

import (
	"net"
	"net/netip"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/golibs/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClients(t *testing.T) {
	clients := clientsContainer{}
	clients.testing = true

	clients.Init(nil, nil, nil, nil)

	t.Run("add_success", func(t *testing.T) {
		var (
			cliNone = "1.2.3.4"
			cli1    = "1.1.1.1"
			cli2    = "2.2.2.2"

			cliNoneIP = netip.MustParseAddr(cliNone)
			cli1IP    = netip.MustParseAddr(cli1)
			cli2IP    = netip.MustParseAddr(cli2)
		)

		c := &Client{
			IDs:  []string{cli1, "1:2:3::4", "aa:aa:aa:aa:aa:aa"},
			Name: "client1",
		}

		ok, err := clients.Add(c)
		require.NoError(t, err)

		assert.True(t, ok)

		c = &Client{
			IDs:  []string{cli2},
			Name: "client2",
		}

		ok, err = clients.Add(c)
		require.NoError(t, err)

		assert.True(t, ok)

		c, ok = clients.Find(cli1)
		require.True(t, ok)

		assert.Equal(t, "client1", c.Name)

		c, ok = clients.Find("1:2:3::4")
		require.True(t, ok)

		assert.Equal(t, "client1", c.Name)

		c, ok = clients.Find(cli2)
		require.True(t, ok)

		assert.Equal(t, "client2", c.Name)

		assert.False(t, clients.exists(cliNoneIP, ClientSourceHostsFile))
		assert.True(t, clients.exists(cli1IP, ClientSourceHostsFile))
		assert.True(t, clients.exists(cli2IP, ClientSourceHostsFile))
	})

	t.Run("add_fail_name", func(t *testing.T) {
		ok, err := clients.Add(&Client{
			IDs:  []string{"1.2.3.5"},
			Name: "client1",
		})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("add_fail_ip", func(t *testing.T) {
		ok, err := clients.Add(&Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client3",
		})
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("update_fail_name", func(t *testing.T) {
		err := clients.Update("client3", &Client{
			IDs:  []string{"1.2.3.0"},
			Name: "client3",
		})
		require.Error(t, err)

		err = clients.Update("client3", &Client{
			IDs:  []string{"1.2.3.0"},
			Name: "client2",
		})
		assert.Error(t, err)
	})

	t.Run("update_fail_ip", func(t *testing.T) {
		err := clients.Update("client1", &Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client1",
		})
		assert.Error(t, err)
	})

	t.Run("update_success", func(t *testing.T) {
		var (
			cliOld = "1.1.1.1"
			cliNew = "1.1.1.2"

			cliOldIP = netip.MustParseAddr(cliOld)
			cliNewIP = netip.MustParseAddr(cliNew)
		)

		err := clients.Update("client1", &Client{
			IDs:  []string{cliNew},
			Name: "client1",
		})
		require.NoError(t, err)

		assert.False(t, clients.exists(cliOldIP, ClientSourceHostsFile))
		assert.True(t, clients.exists(cliNewIP, ClientSourceHostsFile))

		err = clients.Update("client1", &Client{
			IDs:            []string{cliNew},
			Name:           "client1-renamed",
			UseOwnSettings: true,
		})
		require.NoError(t, err)

		c, ok := clients.Find(cliNew)
		require.True(t, ok)

		assert.Equal(t, "client1-renamed", c.Name)
		assert.True(t, c.UseOwnSettings)

		nilCli, ok := clients.list["client1"]
		require.False(t, ok)

		assert.Nil(t, nilCli)

		require.Len(t, c.IDs, 1)

		assert.Equal(t, cliNew, c.IDs[0])
	})

	t.Run("del_success", func(t *testing.T) {
		ok := clients.Del("client1-renamed")
		require.True(t, ok)

		assert.False(t, clients.exists(netip.MustParseAddr("1.1.1.2"), ClientSourceHostsFile))
	})

	t.Run("del_fail", func(t *testing.T) {
		ok := clients.Del("client3")
		assert.False(t, ok)
	})

	t.Run("addhost_success", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")
		ok := clients.AddHost(ip, "host", ClientSourceARP)
		assert.True(t, ok)

		ok = clients.AddHost(ip, "host2", ClientSourceARP)
		assert.True(t, ok)

		ok = clients.AddHost(ip, "host3", ClientSourceHostsFile)
		assert.True(t, ok)

		assert.True(t, clients.exists(ip, ClientSourceHostsFile))
	})

	t.Run("dhcp_replaces_arp", func(t *testing.T) {
		ip := netip.MustParseAddr("1.2.3.4")
		ok := clients.AddHost(ip, "from_arp", ClientSourceARP)
		assert.True(t, ok)
		assert.True(t, clients.exists(ip, ClientSourceARP))

		ok = clients.AddHost(ip, "from_dhcp", ClientSourceDHCP)
		assert.True(t, ok)
		assert.True(t, clients.exists(ip, ClientSourceDHCP))
	})

	t.Run("addhost_fail", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")
		ok := clients.AddHost(ip, "host1", ClientSourceRDNS)
		assert.False(t, ok)
	})
}

func TestClientsWHOIS(t *testing.T) {
	clients := clientsContainer{
		testing: true,
	}
	clients.Init(nil, nil, nil, nil)
	whois := &RuntimeClientWHOISInfo{
		Country: "AU",
		Orgname: "Example Org",
	}

	t.Run("new_client", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.255")
		clients.setWHOISInfo(ip, whois)
		rc := clients.ipToRC[ip]
		require.NotNil(t, rc)

		assert.Equal(t, rc.WHOISInfo, whois)
	})

	t.Run("existing_auto-client", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")
		ok := clients.AddHost(ip, "host", ClientSourceRDNS)
		assert.True(t, ok)

		clients.setWHOISInfo(ip, whois)
		rc := clients.ipToRC[ip]
		require.NotNil(t, rc)

		assert.Equal(t, rc.WHOISInfo, whois)
	})

	t.Run("can't_set_manually-added", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.2")

		ok, err := clients.Add(&Client{
			IDs:  []string{"1.1.1.2"},
			Name: "client1",
		})
		require.NoError(t, err)
		assert.True(t, ok)

		clients.setWHOISInfo(ip, whois)
		rc := clients.ipToRC[ip]
		require.Nil(t, rc)

		assert.True(t, clients.Del("client1"))
	})
}

func TestClientsAddExisting(t *testing.T) {
	clients := clientsContainer{
		testing: true,
	}
	clients.Init(nil, nil, nil, nil)

	t.Run("simple", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")

		// Add a client.
		ok, err := clients.Add(&Client{
			IDs:  []string{ip.String(), "1:2:3::4", "aa:aa:aa:aa:aa:aa", "2.2.2.0/24"},
			Name: "client1",
		})
		require.NoError(t, err)
		assert.True(t, ok)

		// Now add an auto-client with the same IP.
		ok = clients.AddHost(ip, "test", ClientSourceRDNS)
		assert.True(t, ok)
	})

	t.Run("complicated", func(t *testing.T) {
		// TODO(a.garipov): Properly decouple the DHCP server from the client
		// storage.
		if runtime.GOOS == "windows" {
			t.Skip("skipping dhcp test on windows")
		}

		ip := net.IP{1, 2, 3, 4}

		// First, init a DHCP server with a single static lease.
		config := &dhcpd.ServerConfig{
			Enabled:    true,
			DBFilePath: "leases.db",
			Conf4: dhcpd.V4ServerConf{
				Enabled:    true,
				GatewayIP:  netip.MustParseAddr("1.2.3.1"),
				SubnetMask: netip.MustParseAddr("255.255.255.0"),
				RangeStart: netip.MustParseAddr("1.2.3.2"),
				RangeEnd:   netip.MustParseAddr("1.2.3.10"),
			},
		}

		dhcpServer, err := dhcpd.Create(config)
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, func() (err error) {
			return os.Remove("leases.db")
		})

		clients.dhcpServer = dhcpServer

		err = dhcpServer.AddStaticLease(&dhcpd.Lease{
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       ip,
			Hostname: "testhost",
			Expiry:   time.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		// Add a new client with the same IP as for a client with MAC.
		ok, err := clients.Add(&Client{
			IDs:  []string{ip.String()},
			Name: "client2",
		})
		require.NoError(t, err)
		assert.True(t, ok)

		// Add a new client with the IP from the first client's IP range.
		ok, err = clients.Add(&Client{
			IDs:  []string{"2.2.2.2"},
			Name: "client3",
		})
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestClientsCustomUpstream(t *testing.T) {
	clients := clientsContainer{
		testing: true,
	}
	clients.Init(nil, nil, nil, nil)

	// Add client with upstreams.
	ok, err := clients.Add(&Client{
		IDs:  []string{"1.1.1.1", "1:2:3::4", "aa:aa:aa:aa:aa:aa"},
		Name: "client1",
		Upstreams: []string{
			"1.1.1.1",
			"[/example.org/]8.8.8.8",
		},
	})
	require.NoError(t, err)
	assert.True(t, ok)

	config, err := clients.findUpstreams("1.2.3.4")
	assert.Nil(t, config)
	assert.NoError(t, err)

	config, err = clients.findUpstreams("1.1.1.1")
	require.NotNil(t, config)
	assert.NoError(t, err)
	assert.Len(t, config.Upstreams, 1)
	assert.Len(t, config.DomainReservedUpstreams, 1)
}
