package client_test

import (
	"net"
	"net/netip"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/arpdb"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/faketime"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStorage is a helper function that returns initialized storage.
func newTestStorage(tb testing.TB, clock timeutil.Clock) (s *client.Storage) {
	tb.Helper()

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	s, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger: slogutil.NewDiscardLogger(),
		Clock:  clock,
	})
	require.NoError(tb, err)

	return s
}

// type check
var _ dnsforward.ClientsContainer = (*client.Storage)(nil)

// testHostsContainer is a mock implementation of the [client.HostsContainer]
// interface.
type testHostsContainer struct {
	onUpd func() (updates <-chan *hostsfile.DefaultStorage)
}

// type check
var _ client.HostsContainer = (*testHostsContainer)(nil)

// Upd implements the [client.HostsContainer] interface for *testHostsContainer.
func (c *testHostsContainer) Upd() (updates <-chan *hostsfile.DefaultStorage) {
	return c.onUpd()
}

// Interface stores and refreshes the network neighborhood reported by ARP
// (Address Resolution Protocol).
type Interface interface {
	// Refresh updates the stored data.  It must be safe for concurrent use.
	Refresh() (err error)

	// Neighbors returnes the last set of data reported by ARP.  Both the method
	// and it's result must be safe for concurrent use.
	Neighbors() (ns []arpdb.Neighbor)
}

// testARPDB is a mock implementation of the [arpdb.Interface].
type testARPDB struct {
	onRefresh   func() (err error)
	onNeighbors func() (ns []arpdb.Neighbor)
}

// type check
var _ arpdb.Interface = (*testARPDB)(nil)

// Refresh implements the [arpdb.Interface] interface for *testARP.
func (c *testARPDB) Refresh() (err error) {
	return c.onRefresh()
}

// Neighbors implements the [arpdb.Interface] interface for *testARP.
func (c *testARPDB) Neighbors() (ns []arpdb.Neighbor) {
	return c.onNeighbors()
}

// testDHCP is a mock implementation of the [client.DHCP].
type testDHCP struct {
	OnLeases func() (leases []*dhcpsvc.Lease)
	OnHostBy func(ip netip.Addr) (host string)
	OnMACBy  func(ip netip.Addr) (mac net.HardwareAddr)
}

// type check
var _ client.DHCP = (*testDHCP)(nil)

// Lease implements the [client.DHCP] interface for *testDHCP.
func (t *testDHCP) Leases() (leases []*dhcpsvc.Lease) { return t.OnLeases() }

// HostByIP implements the [client.DHCP] interface for *testDHCP.
func (t *testDHCP) HostByIP(ip netip.Addr) (host string) { return t.OnHostBy(ip) }

// MACByIP implements the [client.DHCP] interface for *testDHCP.
func (t *testDHCP) MACByIP(ip netip.Addr) (mac net.HardwareAddr) { return t.OnMACBy(ip) }

// compareRuntimeInfo is a helper function that returns true if the runtime
// client has provided info.
func compareRuntimeInfo(rc *client.Runtime, src client.Source, host string) (ok bool) {
	s, h := rc.Info()
	if s != src {
		return false
	} else if h != host {
		return false
	}

	return true
}

func TestStorage_Add_hostsfile(t *testing.T) {
	var (
		cliIP1   = netip.MustParseAddr("1.1.1.1")
		cliName1 = "client_one"

		cliIP2   = netip.MustParseAddr("2.2.2.2")
		cliName2 = "client_two"
	)

	hostCh := make(chan *hostsfile.DefaultStorage)
	h := &testHostsContainer{
		onUpd: func() (updates <-chan *hostsfile.DefaultStorage) { return hostCh },
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	storage, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger:                 slogutil.NewDiscardLogger(),
		DHCP:                   client.EmptyDHCP{},
		EtcHosts:               h,
		ARPClientsUpdatePeriod: testTimeout / 10,
	})
	require.NoError(t, err)

	err = storage.Start(testutil.ContextWithTimeout(t, testTimeout))
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return storage.Shutdown(testutil.ContextWithTimeout(t, testTimeout))
	})

	t.Run("add_hosts", func(t *testing.T) {
		var s *hostsfile.DefaultStorage
		s, err = hostsfile.NewDefaultStorage()
		require.NoError(t, err)

		s.Add(&hostsfile.Record{
			Addr:  cliIP1,
			Names: []string{cliName1},
		})

		testutil.RequireSend(t, hostCh, s, testTimeout)

		require.Eventually(t, func() (ok bool) {
			cli1 := storage.ClientRuntime(cliIP1)
			if cli1 == nil {
				return false
			}

			assert.True(t, compareRuntimeInfo(cli1, client.SourceHostsFile, cliName1))

			return true
		}, testTimeout, testTimeout/10)
	})

	t.Run("update_hosts", func(t *testing.T) {
		var s *hostsfile.DefaultStorage
		s, err = hostsfile.NewDefaultStorage()
		require.NoError(t, err)

		s.Add(&hostsfile.Record{
			Addr:  cliIP2,
			Names: []string{cliName2},
		})

		testutil.RequireSend(t, hostCh, s, testTimeout)

		require.Eventually(t, func() (ok bool) {
			cli2 := storage.ClientRuntime(cliIP2)
			if cli2 == nil {
				return false
			}

			assert.True(t, compareRuntimeInfo(cli2, client.SourceHostsFile, cliName2))

			cli1 := storage.ClientRuntime(cliIP1)
			require.Nil(t, cli1)

			return true
		}, testTimeout, testTimeout/10)
	})
}

func TestStorage_Add_arp(t *testing.T) {
	var (
		mu        sync.Mutex
		neighbors []arpdb.Neighbor

		cliIP1   = netip.MustParseAddr("1.1.1.1")
		cliName1 = "client_one"

		cliIP2   = netip.MustParseAddr("2.2.2.2")
		cliName2 = "client_two"
	)

	a := &testARPDB{
		onRefresh: func() (err error) { return nil },
		onNeighbors: func() (ns []arpdb.Neighbor) {
			mu.Lock()
			defer mu.Unlock()

			return neighbors
		},
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	storage, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger:                 slogutil.NewDiscardLogger(),
		DHCP:                   client.EmptyDHCP{},
		ARPDB:                  a,
		ARPClientsUpdatePeriod: testTimeout / 10,
	})
	require.NoError(t, err)

	err = storage.Start(testutil.ContextWithTimeout(t, testTimeout))
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return storage.Shutdown(testutil.ContextWithTimeout(t, testTimeout))
	})

	t.Run("add_hosts", func(t *testing.T) {
		func() {
			mu.Lock()
			defer mu.Unlock()

			neighbors = []arpdb.Neighbor{{
				Name: cliName1,
				IP:   cliIP1,
			}}
		}()

		require.Eventually(t, func() (ok bool) {
			cli1 := storage.ClientRuntime(cliIP1)
			if cli1 == nil {
				return false
			}

			assert.True(t, compareRuntimeInfo(cli1, client.SourceARP, cliName1))

			return true
		}, testTimeout, testTimeout/10)
	})

	t.Run("update_hosts", func(t *testing.T) {
		func() {
			mu.Lock()
			defer mu.Unlock()

			neighbors = []arpdb.Neighbor{{
				Name: cliName2,
				IP:   cliIP2,
			}}
		}()

		require.Eventually(t, func() (ok bool) {
			cli2 := storage.ClientRuntime(cliIP2)
			if cli2 == nil {
				return false
			}

			assert.True(t, compareRuntimeInfo(cli2, client.SourceARP, cliName2))

			cli1 := storage.ClientRuntime(cliIP1)
			require.Nil(t, cli1)

			return true
		}, testTimeout, testTimeout/10)
	})
}

func TestStorage_Add_whois(t *testing.T) {
	var (
		cliIP1 = netip.MustParseAddr("1.1.1.1")

		cliIP2   = netip.MustParseAddr("2.2.2.2")
		cliName2 = "client_two"

		cliIP3   = netip.MustParseAddr("3.3.3.3")
		cliName3 = "client_three"
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	storage, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger: slogutil.NewDiscardLogger(),
		DHCP:   client.EmptyDHCP{},
	})
	require.NoError(t, err)

	whois := &whois.Info{
		Country: "AU",
		Orgname: "Example Org",
	}

	t.Run("new_client", func(t *testing.T) {
		storage.UpdateAddress(ctx, cliIP1, "", whois)
		cli1 := storage.ClientRuntime(cliIP1)
		require.NotNil(t, cli1)

		assert.Equal(t, whois, cli1.WHOIS())
	})

	t.Run("existing_runtime_client", func(t *testing.T) {
		storage.UpdateAddress(ctx, cliIP2, cliName2, nil)
		storage.UpdateAddress(ctx, cliIP2, "", whois)

		cli2 := storage.ClientRuntime(cliIP2)
		require.NotNil(t, cli2)

		assert.True(t, compareRuntimeInfo(cli2, client.SourceRDNS, cliName2))

		assert.Equal(t, whois, cli2.WHOIS())
	})

	t.Run("can't_set_persistent_client", func(t *testing.T) {
		err = storage.Add(ctx, &client.Persistent{
			Name: cliName3,
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{cliIP3},
		})
		require.NoError(t, err)

		storage.UpdateAddress(ctx, cliIP3, "", whois)
		rc := storage.ClientRuntime(cliIP3)
		require.Nil(t, rc)
	})
}

func TestClientsDHCP(t *testing.T) {
	var (
		cliIP1   = netip.MustParseAddr("1.1.1.1")
		cliName1 = "one.dhcp"

		cliIP2   = netip.MustParseAddr("2.2.2.2")
		cliMAC2  = errors.Must(net.ParseMAC("22:22:22:22:22:22"))
		cliName2 = "two.dhcp"

		cliIP3   = netip.MustParseAddr("3.3.3.3")
		cliMAC3  = errors.Must(net.ParseMAC("33:33:33:33:33:33"))
		cliName3 = "three.dhcp"

		prsCliIP   = netip.MustParseAddr("4.3.2.1")
		prsCliMAC  = errors.Must(net.ParseMAC("AA:AA:AA:AA:AA:AA"))
		prsCliName = "persistent.dhcp"

		otherARPCliName = "other.arp"
		otherARPCliIP   = netip.MustParseAddr("192.0.2.1")
	)

	ipToHost := map[netip.Addr]string{
		cliIP1: cliName1,
	}
	ipToMAC := map[netip.Addr]net.HardwareAddr{
		prsCliIP: prsCliMAC,
	}

	leases := []*dhcpsvc.Lease{{
		IP:       cliIP2,
		Hostname: cliName2,
		HWAddr:   cliMAC2,
	}, {
		IP:       cliIP3,
		Hostname: cliName3,
		HWAddr:   cliMAC3,
	}}

	arpCh := make(chan []arpdb.Neighbor, 1)
	arpDB := &testARPDB{
		onRefresh: func() (err error) { return nil },
		onNeighbors: func() (ns []arpdb.Neighbor) {
			select {
			case ns = <-arpCh:
				return ns
			default:
				return nil
			}
		},
	}

	dhcp := &testDHCP{
		OnLeases: func() (ls []*dhcpsvc.Lease) {
			return leases
		},
		OnHostBy: func(ip netip.Addr) (host string) {
			return ipToHost[ip]
		},
		OnMACBy: func(ip netip.Addr) (mac net.HardwareAddr) {
			return ipToMAC[ip]
		},
	}

	etcHostsCh := make(chan *hostsfile.DefaultStorage, 1)
	etcHosts := &testHostsContainer{
		onUpd: func() (updates <-chan *hostsfile.DefaultStorage) {
			return etcHostsCh
		},
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	storage, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger:                 slogutil.NewDiscardLogger(),
		ARPDB:                  arpDB,
		DHCP:                   dhcp,
		EtcHosts:               etcHosts,
		RuntimeSourceDHCP:      true,
		ARPClientsUpdatePeriod: testTimeout / 10,
	})
	require.NoError(t, err)

	err = storage.Start(testutil.ContextWithTimeout(t, testTimeout))
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return storage.Shutdown(testutil.ContextWithTimeout(t, testTimeout))
	})

	require.True(t, t.Run("find_runtime_lower_priority", func(t *testing.T) {
		// Add a lower-priority client.
		ns := []arpdb.Neighbor{{
			Name: cliName1,
			IP:   cliIP1,
		}}

		testutil.RequireSend(t, arpCh, ns, testTimeout)

		storage.ReloadARP(testutil.ContextWithTimeout(t, testTimeout))

		cli1 := storage.ClientRuntime(cliIP1)
		require.NotNil(t, cli1)

		assert.True(t, compareRuntimeInfo(cli1, client.SourceDHCP, cliName1))

		// Remove the matching client.
		//
		// TODO(a.garipov):  Consider adding ways of explicitly clearing runtime
		// sources by source.
		ns = []arpdb.Neighbor{{
			Name: otherARPCliName,
			IP:   otherARPCliIP,
		}}

		testutil.RequireSend(t, arpCh, ns, testTimeout)

		storage.ReloadARP(testutil.ContextWithTimeout(t, testTimeout))
	}))

	require.True(t, t.Run("find_runtime", func(t *testing.T) {
		cli1 := storage.ClientRuntime(cliIP1)
		require.NotNil(t, cli1)

		assert.True(t, compareRuntimeInfo(cli1, client.SourceDHCP, cliName1))
	}))

	require.True(t, t.Run("find_runtime_higher_priority", func(t *testing.T) {
		// Add a higher-priority client.
		s, strgErr := hostsfile.NewDefaultStorage()
		require.NoError(t, strgErr)

		s.Add(&hostsfile.Record{
			Addr:  cliIP1,
			Names: []string{cliName1},
		})

		testutil.RequireSend(t, etcHostsCh, s, testTimeout)

		cli1 := storage.ClientRuntime(cliIP1)
		require.NotNil(t, cli1)

		require.Eventually(t, func() (ok bool) {
			cli := storage.ClientRuntime(cliIP1)
			if cli == nil {
				return false
			}

			assert.True(t, compareRuntimeInfo(cli, client.SourceHostsFile, cliName1))

			return true
		}, testTimeout, testTimeout/10)

		// Remove the matching client.
		//
		// TODO(a.garipov):  Consider adding ways of explicitly clearing runtime
		// sources by source.
		s, strgErr = hostsfile.NewDefaultStorage()
		require.NoError(t, strgErr)

		testutil.RequireSend(t, etcHostsCh, s, testTimeout)

		require.Eventually(t, func() (ok bool) {
			cli := storage.ClientRuntime(cliIP1)

			return compareRuntimeInfo(cli, client.SourceDHCP, cliName1)
		}, testTimeout, testTimeout/10)
	}))

	require.True(t, t.Run("find_persistent", func(t *testing.T) {
		err = storage.Add(ctx, &client.Persistent{
			Name: prsCliName,
			UID:  client.MustNewUID(),
			MACs: []net.HardwareAddr{prsCliMAC},
		})
		require.NoError(t, err)

		params := &client.FindParams{}
		err = params.Set(prsCliIP.String())
		require.NoError(t, err)

		prsCli, ok := storage.Find(params)
		require.True(t, ok)

		assert.Equal(t, prsCliName, prsCli.Name)
	}))

	require.True(t, t.Run("leases", func(t *testing.T) {
		delete(ipToHost, cliIP1)
		storage.UpdateDHCP(ctx)

		cli1 := storage.ClientRuntime(cliIP1)
		require.Nil(t, cli1)

		for i, l := range leases {
			cli := storage.ClientRuntime(l.IP)
			require.NotNil(t, cli)

			src, host := cli.Info()
			assert.Equal(t, client.SourceDHCP, src)
			assert.Equal(t, leases[i].Hostname, host)
		}
	}))

	require.True(t, t.Run("range", func(t *testing.T) {
		s := 0
		storage.RangeRuntime(func(rc *client.Runtime) (cont bool) {
			if src, _ := rc.Info(); src == client.SourceDHCP {
				s++
			}

			return true
		})

		assert.Equal(t, len(leases), s)
	}))
}

func TestClientsAddExisting(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	t.Run("simple", func(t *testing.T) {
		storage, err := client.NewStorage(ctx, &client.StorageConfig{
			Logger: slogutil.NewDiscardLogger(),
			DHCP:   client.EmptyDHCP{},
		})
		require.NoError(t, err)

		ip := netip.MustParseAddr("1.1.1.1")

		// Add a client.
		err = storage.Add(ctx, &client.Persistent{
			Name:    "client1",
			UID:     client.MustNewUID(),
			IPs:     []netip.Addr{ip, netip.MustParseAddr("1:2:3::4")},
			Subnets: []netip.Prefix{netip.MustParsePrefix("2.2.2.0/24")},
			MACs:    []net.HardwareAddr{{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}},
		})
		require.NoError(t, err)

		// Now add an auto-client with the same IP.
		storage.UpdateAddress(ctx, ip, "test", nil)
		rc := storage.ClientRuntime(ip)
		assert.True(t, compareRuntimeInfo(rc, client.SourceRDNS, "test"))
	})

	t.Run("complicated", func(t *testing.T) {
		// TODO(a.garipov): Properly decouple the DHCP server from the client
		// storage.
		if runtime.GOOS == "windows" {
			t.Skip("skipping dhcp test on windows")
		}

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

		storage, err := client.NewStorage(ctx, &client.StorageConfig{
			Logger: slogutil.NewDiscardLogger(),
			DHCP:   dhcpServer,
		})
		require.NoError(t, err)

		ip := netip.MustParseAddr("1.2.3.4")

		err = dhcpServer.AddStaticLease(&dhcpsvc.Lease{
			HWAddr:   net.HardwareAddr{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			IP:       ip,
			Hostname: "testhost",
			Expiry:   time.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		// Add a new client with the same IP as for a client with MAC.
		err = storage.Add(ctx, &client.Persistent{
			Name: "client2",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{ip},
		})
		require.NoError(t, err)

		// Add a new client with the IP from the first client's IP range.
		err = storage.Add(ctx, &client.Persistent{
			Name: "client3",
			UID:  client.MustNewUID(),
			IPs:  []netip.Addr{netip.MustParseAddr("2.2.2.2")},
		})
		require.NoError(t, err)
	})
}

// newStorage is a helper function that returns a client storage filled with
// persistent clients from the m.  It also generates a UID for each client.
func newStorage(tb testing.TB, m []*client.Persistent) (s *client.Storage) {
	tb.Helper()

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	s, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger: slogutil.NewDiscardLogger(),
		DHCP:   client.EmptyDHCP{},
	})
	require.NoError(tb, err)

	for _, c := range m {
		c.UID = client.MustNewUID()
		require.NoError(tb, s.Add(ctx, c))
	}

	require.Equal(tb, len(m), s.Size())

	return s
}

func TestStorage_Add(t *testing.T) {
	const (
		existingName     = "existing_name"
		existingClientID = "existing_client_id"

		allowedTag    = "user_admin"
		notAllowedTag = "not_allowed_tag"
	)

	var (
		existingClientUID = client.MustNewUID()
		existingIP        = netip.MustParseAddr("1.2.3.4")
		existingSubnet    = netip.MustParsePrefix("1.2.3.0/24")
	)

	existingClient := &client.Persistent{
		Name:      existingName,
		IPs:       []netip.Addr{existingIP},
		Subnets:   []netip.Prefix{existingSubnet},
		ClientIDs: []client.ClientID{existingClientID},
		UID:       existingClientUID,
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	s := newTestStorage(t, timeutil.SystemClock{})
	tags := s.AllowedTags()
	require.NotZero(t, len(tags))
	require.True(t, slices.IsSorted(tags))

	_, ok := slices.BinarySearch(tags, allowedTag)
	require.True(t, ok)

	_, ok = slices.BinarySearch(tags, notAllowedTag)
	require.False(t, ok)

	err := s.Add(ctx, existingClient)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		cli        *client.Persistent
		wantErrMsg string
	}{{
		name: "basic",
		cli: &client.Persistent{
			Name: "basic",
			IPs:  []netip.Addr{netip.MustParseAddr("1.1.1.1")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: "",
	}, {
		name: "duplicate_uid",
		cli: &client.Persistent{
			Name: "no_uid",
			IPs:  []netip.Addr{netip.MustParseAddr("2.2.2.2")},
			UID:  existingClientUID,
		},
		wantErrMsg: `adding client: another client "existing_name" uses the same uid`,
	}, {
		name: "duplicate_name",
		cli: &client.Persistent{
			Name: existingName,
			IPs:  []netip.Addr{netip.MustParseAddr("3.3.3.3")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: `adding client: another client uses the same name "existing_name"`,
	}, {
		name: "duplicate_ip",
		cli: &client.Persistent{
			Name: "duplicate_ip",
			IPs:  []netip.Addr{existingIP},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: `adding client: another client "existing_name" uses the same IP "1.2.3.4"`,
	}, {
		name: "duplicate_subnet",
		cli: &client.Persistent{
			Name:    "duplicate_subnet",
			Subnets: []netip.Prefix{existingSubnet},
			UID:     client.MustNewUID(),
		},
		wantErrMsg: `adding client: another client "existing_name" ` +
			`uses the same subnet "1.2.3.0/24"`,
	}, {
		name: "duplicate_client_id",
		cli: &client.Persistent{
			Name:      "duplicate_client_id",
			ClientIDs: []client.ClientID{existingClientID},
			UID:       client.MustNewUID(),
		},
		wantErrMsg: `adding client: another client "existing_name" ` +
			`uses the same ClientID "existing_client_id"`,
	}, {
		name: "not_allowed_tag",
		cli: &client.Persistent{
			Name: "not_allowed_tag",
			Tags: []string{notAllowedTag},
			IPs:  []netip.Addr{netip.MustParseAddr("4.4.4.4")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: `adding client: invalid tag: "not_allowed_tag"`,
	}, {
		name: "allowed_tag",
		cli: &client.Persistent{
			Name: "allowed_tag",
			Tags: []string{allowedTag},
			IPs:  []netip.Addr{netip.MustParseAddr("5.5.5.5")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: "",
	}, {
		name: "",
		cli: &client.Persistent{
			Name: "",
			IPs:  []netip.Addr{netip.MustParseAddr("6.6.6.6")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: "adding client: empty name",
	}, {
		name: "no_id",
		cli: &client.Persistent{
			Name: "no_id",
			UID:  client.MustNewUID(),
		},
		wantErrMsg: "adding client: id required",
	}, {
		name: "no_uid",
		cli: &client.Persistent{
			Name: "no_uid",
			IPs:  []netip.Addr{netip.MustParseAddr("7.7.7.7")},
		},
		wantErrMsg: "adding client: uid required",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = s.Add(ctx, tc.cli)

			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestStorage_RemoveByName(t *testing.T) {
	const (
		existingName = "existing_name"
	)

	existingClient := &client.Persistent{
		Name: existingName,
		IPs:  []netip.Addr{netip.MustParseAddr("1.2.3.4")},
		UID:  client.MustNewUID(),
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	s := newTestStorage(t, timeutil.SystemClock{})
	err := s.Add(ctx, existingClient)
	require.NoError(t, err)

	testCases := []struct {
		want    assert.BoolAssertionFunc
		name    string
		cliName string
	}{{
		name:    "existing_client",
		cliName: existingName,
		want:    assert.True,
	}, {
		name:    "non_existing_client",
		cliName: "non_existing_client",
		want:    assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.want(t, s.RemoveByName(ctx, tc.cliName))
		})
	}

	t.Run("duplicate_remove", func(t *testing.T) {
		s = newTestStorage(t, timeutil.SystemClock{})
		err = s.Add(ctx, existingClient)
		require.NoError(t, err)

		assert.True(t, s.RemoveByName(ctx, existingName))
		assert.False(t, s.RemoveByName(ctx, existingName))
	})
}

func TestStorage_Find(t *testing.T) {
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
		clientWithBothFams = &client.Persistent{
			Name: "client1",
			IPs: []netip.Addr{
				netip.MustParseAddr(cliIP1),
				netip.MustParseAddr(cliIPv6),
			},
		}

		clientWithSubnet = &client.Persistent{
			Name:    "client2",
			IPs:     []netip.Addr{netip.MustParseAddr(cliIP2)},
			Subnets: []netip.Prefix{netip.MustParsePrefix(cliSubnet)},
		}

		clientWithMAC = &client.Persistent{
			Name: "client_with_mac",
			MACs: []net.HardwareAddr{errors.Must(net.ParseMAC(cliMAC))},
		}

		clientWithID = &client.Persistent{
			Name:      "client_with_id",
			ClientIDs: []client.ClientID{cliID},
		}

		clientLinkLocal = &client.Persistent{
			Name:    "client_link_local",
			Subnets: []netip.Prefix{netip.MustParsePrefix(linkLocalSubnet)},
		}
	)

	clients := []*client.Persistent{
		clientWithBothFams,
		clientWithSubnet,
		clientWithMAC,
		clientWithID,
		clientLinkLocal,
	}
	s := newStorage(t, clients)

	testCases := []struct {
		want *client.Persistent
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
				params := &client.FindParams{}
				err := params.Set(id)
				require.NoError(t, err)

				c, ok := s.Find(params)
				require.True(t, ok)

				assert.Equal(t, tc.want, c)
			}
		})
	}

	t.Run("not_found", func(t *testing.T) {
		params := &client.FindParams{}
		err := params.Set(cliIPNone)
		require.NoError(t, err)

		_, ok := s.Find(params)
		assert.False(t, ok)
	})
}

func TestStorage_FindLoose(t *testing.T) {
	const (
		nonExistingClientID = "client_id"
	)

	var (
		ip         = netip.MustParseAddr("fe80::a098:7654:32ef:ff1")
		ipWithZone = netip.MustParseAddr("fe80::1ff:fe23:4567:890a%eth2")
	)

	var (
		clientNoZone = &client.Persistent{
			Name: "client",
			IPs:  []netip.Addr{ip},
		}

		clientWithZone = &client.Persistent{
			Name: "client_with_zone",
			IPs:  []netip.Addr{ipWithZone},
		}
	)

	s := newStorage(
		t,
		[]*client.Persistent{
			clientNoZone,
			clientWithZone,
		},
	)

	testCases := []struct {
		ip      netip.Addr
		want    assert.BoolAssertionFunc
		wantCli *client.Persistent
		name    string
	}{{
		name:    "without_zone",
		ip:      ip,
		wantCli: clientNoZone,
		want:    assert.True,
	}, {
		name:    "with_zone",
		ip:      ipWithZone,
		wantCli: clientWithZone,
		want:    assert.True,
	}, {
		name:    "zero_address",
		ip:      netip.Addr{},
		wantCli: nil,
		want:    assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := s.FindLoose(tc.ip.WithZone(""), nonExistingClientID)
			assert.Equal(t, tc.wantCli, c)
			tc.want(t, ok)
		})
	}
}

func TestStorage_Update(t *testing.T) {
	const (
		clientName          = "client_name"
		obstructingName     = "obstructing_name"
		obstructingClientID = "obstructing_client_id"
	)

	var (
		obstructingIP     = netip.MustParseAddr("1.2.3.4")
		obstructingSubnet = netip.MustParsePrefix("1.2.3.0/24")
	)

	obstructingClient := &client.Persistent{
		Name:      obstructingName,
		IPs:       []netip.Addr{obstructingIP},
		Subnets:   []netip.Prefix{obstructingSubnet},
		ClientIDs: []client.ClientID{obstructingClientID},
	}

	clientToUpdate := &client.Persistent{
		Name: clientName,
		IPs:  []netip.Addr{netip.MustParseAddr("1.1.1.1")},
	}

	testCases := []struct {
		name       string
		cli        *client.Persistent
		wantErrMsg string
	}{{
		name: "basic",
		cli: &client.Persistent{
			Name: "basic",
			IPs:  []netip.Addr{netip.MustParseAddr("1.1.1.1")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: "",
	}, {
		name: "duplicate_name",
		cli: &client.Persistent{
			Name: obstructingName,
			IPs:  []netip.Addr{netip.MustParseAddr("3.3.3.3")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: `updating client: another client uses the same name "obstructing_name"`,
	}, {
		name: "duplicate_ip",
		cli: &client.Persistent{
			Name: "duplicate_ip",
			IPs:  []netip.Addr{obstructingIP},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: `updating client: another client "obstructing_name" uses the same IP "1.2.3.4"`,
	}, {
		name: "duplicate_subnet",
		cli: &client.Persistent{
			Name:    "duplicate_subnet",
			Subnets: []netip.Prefix{obstructingSubnet},
			UID:     client.MustNewUID(),
		},
		wantErrMsg: `updating client: another client "obstructing_name" ` +
			`uses the same subnet "1.2.3.0/24"`,
	}, {
		name: "duplicate_client_id",
		cli: &client.Persistent{
			Name:      "duplicate_client_id",
			ClientIDs: []client.ClientID{obstructingClientID},
			UID:       client.MustNewUID(),
		},
		wantErrMsg: `updating client: another client "obstructing_name" ` +
			`uses the same ClientID "obstructing_client_id"`,
	}}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStorage(
				t,
				[]*client.Persistent{
					clientToUpdate,
					obstructingClient,
				},
			)

			err := s.Update(ctx, clientName, tc.cli)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestStorage_RangeByName(t *testing.T) {
	sortedClients := []*client.Persistent{{
		Name:      "clientA",
		ClientIDs: []client.ClientID{"A"},
	}, {
		Name:      "clientB",
		ClientIDs: []client.ClientID{"B"},
	}, {
		Name:      "clientC",
		ClientIDs: []client.ClientID{"C"},
	}, {
		Name:      "clientD",
		ClientIDs: []client.ClientID{"D"},
	}, {
		Name:      "clientE",
		ClientIDs: []client.ClientID{"E"},
	}}

	testCases := []struct {
		name string
		want []*client.Persistent
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
			s := newStorage(t, tc.want)

			var got []*client.Persistent
			s.RangeByName(func(c *client.Persistent) (cont bool) {
				got = append(got, c)

				return true
			})

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStorage_CustomUpstreamConfig(t *testing.T) {
	const (
		existingName     = "existing_name"
		existingClientID = "existing_client_id"

		nonExistingClientID = "non_existing_client_id"
	)

	var (
		existingClientUID = client.MustNewUID()
		existingIP        = netip.MustParseAddr("192.0.2.1")

		nonExistingIP = netip.MustParseAddr("192.0.2.255")

		testUpstreamTimeout = time.Second
	)

	existingClient := &client.Persistent{
		Name:      existingName,
		IPs:       []netip.Addr{existingIP},
		ClientIDs: []client.ClientID{existingClientID},
		UID:       existingClientUID,
		Upstreams: []string{"192.0.2.0"},
	}

	date := time.Now()
	clock := &faketime.Clock{
		OnNow: func() (now time.Time) {
			date = date.Add(time.Second)

			return date
		},
	}

	s := newTestStorage(t, clock)
	s.UpdateCommonUpstreamConfig(&client.CommonUpstreamConfig{
		UpstreamTimeout: testUpstreamTimeout,
	})

	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return s.Shutdown(testutil.ContextWithTimeout(t, testTimeout))
	})

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	err := s.Add(ctx, existingClient)
	require.NoError(t, err)

	testCases := []struct {
		cliAddr     netip.Addr
		wantNilConf assert.ValueAssertionFunc
		name        string
		cliID       string
	}{{
		name:        "client_id",
		cliID:       existingClientID,
		cliAddr:     netip.Addr{},
		wantNilConf: assert.NotNil,
	}, {
		name:        "client_addr",
		cliID:       "",
		cliAddr:     existingIP,
		wantNilConf: assert.NotNil,
	}, {
		name:        "non_existing_client_id",
		cliID:       nonExistingClientID,
		cliAddr:     netip.Addr{},
		wantNilConf: assert.Nil,
	}, {
		name:        "non_existing_client_addr",
		cliID:       "",
		cliAddr:     nonExistingIP,
		wantNilConf: assert.Nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conf := s.CustomUpstreamConfig(tc.cliID, tc.cliAddr)
			tc.wantNilConf(t, conf)
		})
	}

	t.Run("update_common_config", func(t *testing.T) {
		conf := s.CustomUpstreamConfig(existingClientID, existingIP)
		require.NotNil(t, conf)

		s.UpdateCommonUpstreamConfig(&client.CommonUpstreamConfig{
			UpstreamTimeout: testUpstreamTimeout * 2,
		})

		updConf := s.CustomUpstreamConfig(existingClientID, existingIP)
		require.NotNil(t, updConf)

		assert.NotEqual(t, conf, updConf)
	})
}

func BenchmarkFindParams_Set(b *testing.B) {
	const (
		testIPStr    = "192.0.2.1"
		testCIDRStr  = "192.0.2.0/24"
		testMACStr   = "02:00:00:00:00:00"
		testClientID = "clientid"
	)

	benchCases := []struct {
		wantErr error
		params  *client.FindParams
		name    string
		id      string
	}{{
		wantErr: nil,
		params: &client.FindParams{
			ClientID: testClientID,
		},
		name: "client_id",
		id:   testClientID,
	}, {
		wantErr: nil,
		params: &client.FindParams{
			RemoteIP: netip.MustParseAddr(testIPStr),
		},
		name: "ip_address",
		id:   testIPStr,
	}, {
		wantErr: nil,
		params: &client.FindParams{
			Subnet: netip.MustParsePrefix(testCIDRStr),
		},
		name: "subnet",
		id:   testCIDRStr,
	}, {
		wantErr: nil,
		params: &client.FindParams{
			MAC: errors.Must(net.ParseMAC(testMACStr)),
		},
		name: "mac_address",
		id:   testMACStr,
	}, {
		wantErr: client.ErrBadIdentifier,
		params:  &client.FindParams{},
		name:    "bad_id",
		id:      "!@#$%^&*()_+",
	}}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			params := &client.FindParams{}
			var err error

			b.ReportAllocs()
			for b.Loop() {
				err = params.Set(bc.id)
			}

			assert.ErrorIs(b, err, bc.wantErr)
			assert.Equal(b, bc.params, params)
		})
	}

	// Most recent results:
	//
	//	goos: linux
	//	goarch: amd64
	//	pkg: github.com/AdguardTeam/AdGuardHome/internal/client
	//	cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
	//	BenchmarkFindParams_Set/client_id-8         	49463488	        24.27 ns/op	       0 B/op	       0 allocs/op
	//	BenchmarkFindParams_Set/ip_address-8        	18740977	        62.22 ns/op	       0 B/op	       0 allocs/op
	//	BenchmarkFindParams_Set/subnet-8            	10848192	       110.0 ns/op	       0 B/op	       0 allocs/op
	//	BenchmarkFindParams_Set/mac_address-8       	 8148494	       133.2 ns/op	       8 B/op	       1 allocs/op
	//	BenchmarkFindParams_Set/bad_id-8            	73894278	        16.29 ns/op	       0 B/op	       0 allocs/op
}

func BenchmarkStorage_Find(b *testing.B) {
	const (
		cliID  = "cid"
		cliMAC = "02:00:00:00:00:00"
	)

	const (
		cliNameWithID   = "client_with_id"
		cliNameWithIP   = "client_with_ip"
		cliNameWithCIDR = "client_with_cidr"
		cliNameWithMAC  = "client_with_mac"
	)

	var (
		cliIP   = netip.MustParseAddr("192.0.2.1")
		cliCIDR = netip.MustParsePrefix("192.0.2.0/24")
	)

	var (
		clientWithID = &client.Persistent{
			Name:      cliNameWithID,
			ClientIDs: []client.ClientID{cliID},
		}
		clientWithIP = &client.Persistent{
			Name: cliNameWithIP,
			IPs:  []netip.Addr{cliIP},
		}
		clientWithCIDR = &client.Persistent{
			Name:    cliNameWithCIDR,
			Subnets: []netip.Prefix{cliCIDR},
		}
		clientWithMAC = &client.Persistent{
			Name: cliNameWithMAC,
			MACs: []net.HardwareAddr{errors.Must(net.ParseMAC(cliMAC))},
		}
	)

	clients := []*client.Persistent{
		clientWithID,
		clientWithIP,
		clientWithCIDR,
		clientWithMAC,
	}
	s := newStorage(b, clients)

	benchCases := []struct {
		params   *client.FindParams
		name     string
		wantName string
	}{{
		params: &client.FindParams{
			ClientID: cliID,
		},
		name:     "client_id",
		wantName: cliNameWithID,
	}, {
		params: &client.FindParams{
			RemoteIP: cliIP,
		},
		name:     "ip_address",
		wantName: cliNameWithIP,
	}, {
		params: &client.FindParams{
			Subnet: cliCIDR,
		},
		name:     "subnet",
		wantName: cliNameWithCIDR,
	}, {
		params: &client.FindParams{
			MAC: errors.Must(net.ParseMAC(cliMAC)),
		},
		name:     "mac_address",
		wantName: cliNameWithMAC,
	}}

	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			var p *client.Persistent
			var ok bool

			b.ReportAllocs()
			for b.Loop() {
				p, ok = s.Find(bc.params)
			}

			assert.True(b, ok)
			assert.NotNil(b, p)
			assert.Equal(b, bc.wantName, p.Name)
		})
	}

	// Most recent results:
	//
	//	goos: linux
	//	goarch: amd64
	//	pkg: github.com/AdguardTeam/AdGuardHome/internal/client
	//	cpu: Intel(R) Core(TM) i7-10510U CPU @ 1.80GHz
	//	BenchmarkStorage_Find/client_id-8         	 7070107	       154.4 ns/op	     240 B/op	       2 allocs/op
	//	BenchmarkStorage_Find/ip_address-8        	 6831823	       168.6 ns/op	     248 B/op	       2 allocs/op
	//	BenchmarkStorage_Find/subnet-8            	 7209050	       167.5 ns/op	     256 B/op	       2 allocs/op
	//	BenchmarkStorage_Find/mac_address-8       	 5776131	       199.7 ns/op	     256 B/op	       3 allocs/op
}
