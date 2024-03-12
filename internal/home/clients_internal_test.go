package home

import (
	"net"
	"net/netip"
	"runtime"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testDHCP struct {
	OnLeases func() (leases []*dhcpsvc.Lease)
	OnHostBy func(ip netip.Addr) (host string)
	OnMACBy  func(ip netip.Addr) (mac net.HardwareAddr)
}

// Lease implements the [DHCP] interface for testDHCP.
func (t *testDHCP) Leases() (leases []*dhcpsvc.Lease) { return t.OnLeases() }

// HostByIP implements the [DHCP] interface for testDHCP.
func (t *testDHCP) HostByIP(ip netip.Addr) (host string) { return t.OnHostBy(ip) }

// MACByIP implements the [DHCP] interface for testDHCP.
func (t *testDHCP) MACByIP(ip netip.Addr) (mac net.HardwareAddr) { return t.OnMACBy(ip) }

// newClientsContainer is a helper that creates a new clients container for
// tests.
func newClientsContainer(t *testing.T) (c *clientsContainer) {
	t.Helper()

	c = &clientsContainer{
		testing: true,
	}

	dhcp := &testDHCP{
		OnLeases: func() (leases []*dhcpsvc.Lease) { panic("not implemented") },
		OnHostBy: func(ip netip.Addr) (host string) { return "" },
		OnMACBy:  func(ip netip.Addr) (mac net.HardwareAddr) { return nil },
	}

	require.NoError(t, c.Init(nil, dhcp, nil, nil, &filtering.Config{}))

	return c
}

func TestClients(t *testing.T) {
	clients := newClientsContainer(t)

	t.Run("add_success", func(t *testing.T) {
		var (
			cliNone = "1.2.3.4"
			cli1    = "1.1.1.1"
			cli2    = "2.2.2.2"

			cli1IP = netip.MustParseAddr(cli1)
			cli2IP = netip.MustParseAddr(cli2)

			cliIPv6 = netip.MustParseAddr("1:2:3::4")
		)

		c := &client.Persistent{
			Name: "client1",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{cli1IP, cliIPv6},
		}

		ok, err := clients.add(c)
		require.NoError(t, err)

		assert.True(t, ok)

		c = &client.Persistent{
			Name: "client2",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{cli2IP},
		}

		ok, err = clients.add(c)
		require.NoError(t, err)

		assert.True(t, ok)

		c, ok = clients.find(cli1)
		require.True(t, ok)

		assert.Equal(t, "client1", c.Name)

		c, ok = clients.find("1:2:3::4")
		require.True(t, ok)

		assert.Equal(t, "client1", c.Name)

		c, ok = clients.find(cli2)
		require.True(t, ok)

		assert.Equal(t, "client2", c.Name)

		_, ok = clients.find(cliNone)
		assert.False(t, ok)

		assert.Equal(t, clients.clientSource(cli1IP), client.SourcePersistent)
		assert.Equal(t, clients.clientSource(cli2IP), client.SourcePersistent)
	})

	t.Run("add_fail_name", func(t *testing.T) {
		ok, err := clients.add(&client.Persistent{
			Name: "client1",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{netip.MustParseAddr("1.2.3.5")},
		})
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("add_fail_ip", func(t *testing.T) {
		ok, err := clients.add(&client.Persistent{
			Name: "client3",
			UID:  client.MustNewUID(),
		})
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("update_fail_ip", func(t *testing.T) {
		err := clients.update(&client.Persistent{Name: "client1"}, &client.Persistent{
			Name: "client1",
			UID:  client.MustNewUID(),
		})
		assert.Error(t, err)
	})

	t.Run("update_success", func(t *testing.T) {
		var (
			cliOld = "1.1.1.1"
			cliNew = "1.1.1.2"

			cliNewIP = netip.MustParseAddr(cliNew)
		)

		prev, ok := clients.list["client1"]
		require.True(t, ok)

		err := clients.update(prev, &client.Persistent{
			Name: "client1",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{cliNewIP},
		})
		require.NoError(t, err)

		_, ok = clients.find(cliOld)
		assert.False(t, ok)

		assert.Equal(t, clients.clientSource(cliNewIP), client.SourcePersistent)

		prev, ok = clients.list["client1"]
		require.True(t, ok)

		err = clients.update(prev, &client.Persistent{
			Name:           "client1-renamed",
			UID:            client.MustNewUID(),
			IPs:            []netip.Addr{cliNewIP},
			UseOwnSettings: true,
		})
		require.NoError(t, err)

		c, ok := clients.find(cliNew)
		require.True(t, ok)

		assert.Equal(t, "client1-renamed", c.Name)
		assert.True(t, c.UseOwnSettings)

		nilCli, ok := clients.list["client1"]
		require.False(t, ok)

		assert.Nil(t, nilCli)

		require.Len(t, c.IDs(), 1)

		assert.Equal(t, cliNewIP, c.IPs[0])
	})

	t.Run("del_success", func(t *testing.T) {
		ok := clients.remove("client1-renamed")
		require.True(t, ok)

		_, ok = clients.find("1.1.1.2")
		assert.False(t, ok)
	})

	t.Run("del_fail", func(t *testing.T) {
		ok := clients.remove("client3")
		assert.False(t, ok)
	})

	t.Run("addhost_success", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")
		ok := clients.addHost(ip, "host", client.SourceARP)
		assert.True(t, ok)

		ok = clients.addHost(ip, "host2", client.SourceARP)
		assert.True(t, ok)

		ok = clients.addHost(ip, "host3", client.SourceHostsFile)
		assert.True(t, ok)

		assert.Equal(t, clients.clientSource(ip), client.SourceHostsFile)
	})

	t.Run("dhcp_replaces_arp", func(t *testing.T) {
		ip := netip.MustParseAddr("1.2.3.4")
		ok := clients.addHost(ip, "from_arp", client.SourceARP)
		assert.True(t, ok)
		assert.Equal(t, clients.clientSource(ip), client.SourceARP)

		ok = clients.addHost(ip, "from_dhcp", client.SourceDHCP)
		assert.True(t, ok)
		assert.Equal(t, clients.clientSource(ip), client.SourceDHCP)
	})

	t.Run("addhost_priority", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")
		ok := clients.addHost(ip, "host1", client.SourceRDNS)
		assert.True(t, ok)

		assert.Equal(t, client.SourceHostsFile, clients.clientSource(ip))
	})
}

func TestClientsWHOIS(t *testing.T) {
	clients := newClientsContainer(t)
	whois := &whois.Info{
		Country: "AU",
		Orgname: "Example Org",
	}

	t.Run("new_client", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.255")
		clients.setWHOISInfo(ip, whois)
		rc := clients.ipToRC[ip]
		require.NotNil(t, rc)

		assert.Equal(t, whois, rc.WHOIS())
	})

	t.Run("existing_auto-client", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")
		ok := clients.addHost(ip, "host", client.SourceRDNS)
		assert.True(t, ok)

		clients.setWHOISInfo(ip, whois)
		rc := clients.ipToRC[ip]
		require.NotNil(t, rc)

		assert.Equal(t, whois, rc.WHOIS())
	})

	t.Run("can't_set_manually-added", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.2")

		ok, err := clients.add(&client.Persistent{
			Name: "client1",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{netip.MustParseAddr("1.1.1.2")},
		})
		require.NoError(t, err)
		assert.True(t, ok)

		clients.setWHOISInfo(ip, whois)
		rc := clients.ipToRC[ip]
		require.Nil(t, rc)

		assert.True(t, clients.remove("client1"))
	})
}

func TestClientsAddExisting(t *testing.T) {
	clients := newClientsContainer(t)

	t.Run("simple", func(t *testing.T) {
		ip := netip.MustParseAddr("1.1.1.1")

		// Add a client.
		ok, err := clients.add(&client.Persistent{
			Name:    "client1",
			UID:     client.MustNewUID(),
			IPs:     []netip.Addr{ip, netip.MustParseAddr("1:2:3::4")},
			Subnets: []netip.Prefix{netip.MustParsePrefix("2.2.2.0/24")},
			MACs:    []net.HardwareAddr{{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}},
		})
		require.NoError(t, err)
		assert.True(t, ok)

		// Now add an auto-client with the same IP.
		ok = clients.addHost(ip, "test", client.SourceRDNS)
		assert.True(t, ok)
	})

	t.Run("complicated", func(t *testing.T) {
		// TODO(a.garipov): Properly decouple the DHCP server from the client
		// storage.
		if runtime.GOOS == "windows" {
			t.Skip("skipping dhcp test on windows")
		}

		ip := netip.MustParseAddr("1.2.3.4")

		// First, init a DHCP server with a single static lease.
		config := &dhcpd.ServerConfig{
			Enabled: true,
			DataDir: t.TempDir(),
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

		clients.dhcp = dhcpServer

		err = dhcpServer.AddStaticLease(&dhcpsvc.Lease{
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       ip,
			Hostname: "testhost",
			Expiry:   time.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		// Add a new client with the same IP as for a client with MAC.
		ok, err := clients.add(&client.Persistent{
			Name: "client2",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{ip},
		})
		require.NoError(t, err)
		assert.True(t, ok)

		// Add a new client with the IP from the first client's IP range.
		ok, err = clients.add(&client.Persistent{
			Name: "client3",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{netip.MustParseAddr("2.2.2.2")},
		})
		require.NoError(t, err)
		assert.True(t, ok)
	})
}

func TestClientsCustomUpstream(t *testing.T) {
	clients := newClientsContainer(t)

	// Add client with upstreams.
	ok, err := clients.add(&client.Persistent{
		Name: "client1",
		UID:  client.MustNewUID(),
		IPs:  []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1:2:3::4")},
		Upstreams: []string{
			"1.1.1.1",
			"[/example.org/]8.8.8.8",
		},
	})
	require.NoError(t, err)
	assert.True(t, ok)

	upsConf, err := clients.UpstreamConfigByID("1.2.3.4", net.DefaultResolver)
	assert.Nil(t, upsConf)
	assert.NoError(t, err)

	upsConf, err = clients.UpstreamConfigByID("1.1.1.1", net.DefaultResolver)
	require.NotNil(t, upsConf)
	assert.NoError(t, err)
}
