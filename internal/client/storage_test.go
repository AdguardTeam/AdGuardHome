package client_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStorage is a helper function that returns a client storage filled with
// persistent clients from the m.  It also generates a UID for each client.
func newStorage(tb testing.TB, m []*client.Persistent) (s *client.Storage) {
	tb.Helper()

	s = client.NewStorage(&client.Config{
		AllowedTags: nil,
	})

	for _, c := range m {
		c.UID = client.MustNewUID()
		require.NoError(tb, s.Add(c))
	}

	require.Equal(tb, len(m), s.Size())

	return s
}

// newRuntimeClient is a helper function that returns a new runtime client.
func newRuntimeClient(ip netip.Addr, source client.Source, host string) (rc *client.Runtime) {
	rc = client.NewRuntime(ip)
	rc.SetInfo(source, []string{host})

	return rc
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

func TestStorage_Add(t *testing.T) {
	const (
		existingName     = "existing_name"
		existingClientID = "existing_client_id"

		allowedTag    = "tag"
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
		ClientIDs: []string{existingClientID},
		UID:       existingClientUID,
	}

	s := client.NewStorage(&client.Config{
		AllowedTags: []string{allowedTag},
	})
	err := s.Add(existingClient)
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
			ClientIDs: []string{existingClientID},
			UID:       client.MustNewUID(),
		},
		wantErrMsg: `adding client: another client "existing_name" ` +
			`uses the same ClientID "existing_client_id"`,
	}, {
		name: "not_allowed_tag",
		cli: &client.Persistent{
			Name: "nont_allowed_tag",
			Tags: []string{notAllowedTag},
			IPs:  []netip.Addr{netip.MustParseAddr("4.4.4.4")},
			UID:  client.MustNewUID(),
		},
		wantErrMsg: `adding client: invalid tag: "not_allowed_tag"`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = s.Add(tc.cli)

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

	s := client.NewStorage(&client.Config{
		AllowedTags: nil,
	})
	err := s.Add(existingClient)
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
			tc.want(t, s.RemoveByName(tc.cliName))
		})
	}

	t.Run("duplicate_remove", func(t *testing.T) {
		s = client.NewStorage(&client.Config{
			AllowedTags: nil,
		})
		err = s.Add(existingClient)
		require.NoError(t, err)

		assert.True(t, s.RemoveByName(existingName))
		assert.False(t, s.RemoveByName(existingName))
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
			MACs: []net.HardwareAddr{mustParseMAC(cliMAC)},
		}

		clientWithID = &client.Persistent{
			Name:      "client_with_id",
			ClientIDs: []string{cliID},
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
				c, ok := s.Find(id)
				require.True(t, ok)

				assert.Equal(t, tc.want, c)
			}
		})
	}

	t.Run("not_found", func(t *testing.T) {
		_, ok := s.Find(cliIPNone)
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

func TestStorage_FindByName(t *testing.T) {
	const (
		cliIP1 = "1.1.1.1"
		cliIP2 = "2.2.2.2"
	)

	const (
		clientExistingName        = "client_existing"
		clientAnotherExistingName = "client_another_existing"
		nonExistingClientName     = "client_non_existing"
	)

	var (
		clientExisting = &client.Persistent{
			Name: clientExistingName,
			IPs:  []netip.Addr{netip.MustParseAddr(cliIP1)},
		}

		clientAnotherExisting = &client.Persistent{
			Name: clientAnotherExistingName,
			IPs:  []netip.Addr{netip.MustParseAddr(cliIP2)},
		}
	)

	clients := []*client.Persistent{
		clientExisting,
		clientAnotherExisting,
	}
	s := newStorage(t, clients)

	testCases := []struct {
		want       *client.Persistent
		name       string
		clientName string
	}{{
		name:       "existing",
		clientName: clientExistingName,
		want:       clientExisting,
	}, {
		name:       "another_existing",
		clientName: clientAnotherExistingName,
		want:       clientAnotherExisting,
	}, {
		name:       "non_existing",
		clientName: nonExistingClientName,
		want:       nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := s.FindByName(tc.clientName)
			if tc.want == nil {
				assert.False(t, ok)

				return
			}

			assert.True(t, ok)
			assert.Equal(t, tc.want, c)
		})
	}
}

func TestStorage_FindByMAC(t *testing.T) {
	var (
		cliMAC               = mustParseMAC("11:11:11:11:11:11")
		cliAnotherMAC        = mustParseMAC("22:22:22:22:22:22")
		nonExistingClientMAC = mustParseMAC("33:33:33:33:33:33")
	)

	var (
		clientExisting = &client.Persistent{
			Name: "client",
			MACs: []net.HardwareAddr{cliMAC},
		}

		clientAnotherExisting = &client.Persistent{
			Name: "another_client",
			MACs: []net.HardwareAddr{cliAnotherMAC},
		}
	)

	clients := []*client.Persistent{
		clientExisting,
		clientAnotherExisting,
	}
	s := newStorage(t, clients)

	testCases := []struct {
		want      *client.Persistent
		name      string
		clientMAC net.HardwareAddr
	}{{
		name:      "existing",
		clientMAC: cliMAC,
		want:      clientExisting,
	}, {
		name:      "another_existing",
		clientMAC: cliAnotherMAC,
		want:      clientAnotherExisting,
	}, {
		name:      "non_existing",
		clientMAC: nonExistingClientMAC,
		want:      nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := s.FindByMAC(tc.clientMAC)
			if tc.want == nil {
				assert.False(t, ok)

				return
			}

			assert.True(t, ok)
			assert.Equal(t, tc.want, c)
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
		ClientIDs: []string{obstructingClientID},
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
			ClientIDs: []string{obstructingClientID},
			UID:       client.MustNewUID(),
		},
		wantErrMsg: `updating client: another client "obstructing_name" ` +
			`uses the same ClientID "obstructing_client_id"`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStorage(
				t,
				[]*client.Persistent{
					clientToUpdate,
					obstructingClient,
				},
			)

			err := s.Update(clientName, tc.cli)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestStorage_RangeByName(t *testing.T) {
	sortedClients := []*client.Persistent{{
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

func TestStorage_UpdateRuntime(t *testing.T) {
	const (
		addedARP       = "added_arp"
		addedSecondARP = "added_arp"

		updatedARP = "updated_arp"

		cliCity    = "City"
		cliCountry = "Country"
		cliOrgname = "Orgname"
	)

	var (
		ip  = netip.MustParseAddr("1.1.1.1")
		ip2 = netip.MustParseAddr("2.2.2.2")
	)

	updated := client.NewRuntime(ip)
	updated.SetInfo(client.SourceARP, []string{updatedARP})

	info := &whois.Info{
		City:    cliCity,
		Country: cliCountry,
		Orgname: cliOrgname,
	}
	updated.SetWHOIS(info)

	s := client.NewStorage(&client.Config{
		AllowedTags: nil,
	})

	t.Run("add_arp_client", func(t *testing.T) {
		added := client.NewRuntime(ip)
		added.SetInfo(client.SourceARP, []string{addedARP})

		require.True(t, s.UpdateRuntime(added))
		require.Equal(t, 1, s.SizeRuntime())

		got := s.ClientRuntime(ip)
		source, host := got.Info()
		assert.Equal(t, client.SourceARP, source)
		assert.Equal(t, addedARP, host)
	})

	t.Run("add_second_arp_client", func(t *testing.T) {
		added := client.NewRuntime(ip2)
		added.SetInfo(client.SourceARP, []string{addedSecondARP})

		require.True(t, s.UpdateRuntime(added))
		require.Equal(t, 2, s.SizeRuntime())

		got := s.ClientRuntime(ip2)
		source, host := got.Info()
		assert.Equal(t, client.SourceARP, source)
		assert.Equal(t, addedSecondARP, host)
	})

	t.Run("update_first_client", func(t *testing.T) {
		require.False(t, s.UpdateRuntime(updated))
		got := s.ClientRuntime(ip)
		require.Equal(t, 2, s.SizeRuntime())

		source, host := got.Info()
		assert.Equal(t, client.SourceARP, source)
		assert.Equal(t, updatedARP, host)
	})

	t.Run("remove_arp_info", func(t *testing.T) {
		n := s.DeleteBySource(client.SourceARP)
		require.Equal(t, 1, n)
		require.Equal(t, 1, s.SizeRuntime())

		got := s.ClientRuntime(ip)
		source, _ := got.Info()
		assert.Equal(t, client.SourceWHOIS, source)
		assert.Equal(t, info, got.WHOIS())
	})

	t.Run("remove_whois_info", func(t *testing.T) {
		n := s.DeleteBySource(client.SourceWHOIS)
		require.Equal(t, 1, n)
		require.Equal(t, 0, s.SizeRuntime())
	})
}

func TestStorage_BatchUpdateBySource(t *testing.T) {
	const (
		defSrc = client.SourceARP

		cliFirstHost1   = "host1"
		cliFirstHost2   = "host2"
		cliUpdatedHost3 = "host3"
		cliUpdatedHost4 = "host4"
		cliUpdatedHost5 = "host5"
	)

	var (
		cliFirstIP1   = netip.MustParseAddr("1.1.1.1")
		cliFirstIP2   = netip.MustParseAddr("2.2.2.2")
		cliUpdatedIP3 = netip.MustParseAddr("3.3.3.3")
		cliUpdatedIP4 = netip.MustParseAddr("4.4.4.4")
		cliUpdatedIP5 = netip.MustParseAddr("5.5.5.5")
	)

	firstClients := []*client.Runtime{
		newRuntimeClient(cliFirstIP1, defSrc, cliFirstHost1),
		newRuntimeClient(cliFirstIP2, defSrc, cliFirstHost2),
	}

	updatedClients := []*client.Runtime{
		newRuntimeClient(cliUpdatedIP3, defSrc, cliUpdatedHost3),
		newRuntimeClient(cliUpdatedIP4, defSrc, cliUpdatedHost4),
		newRuntimeClient(cliUpdatedIP5, defSrc, cliUpdatedHost5),
	}

	s := client.NewStorage(&client.Config{
		AllowedTags: nil,
	})

	t.Run("populate_storage_with_first_clients", func(t *testing.T) {
		added, removed := s.BatchUpdateBySource(defSrc, firstClients)
		require.Equal(t, len(firstClients), added)
		require.Equal(t, 0, removed)
		require.Equal(t, len(firstClients), s.SizeRuntime())

		rc := s.ClientRuntime(cliFirstIP1)
		src, host := rc.Info()
		assert.Equal(t, defSrc, src)
		assert.Equal(t, cliFirstHost1, host)
	})

	t.Run("update_storage", func(t *testing.T) {
		added, removed := s.BatchUpdateBySource(defSrc, updatedClients)
		require.Equal(t, len(updatedClients), added)
		require.Equal(t, len(firstClients), removed)
		require.Equal(t, len(updatedClients), s.SizeRuntime())

		rc := s.ClientRuntime(cliUpdatedIP3)
		src, host := rc.Info()
		assert.Equal(t, defSrc, src)
		assert.Equal(t, cliUpdatedHost3, host)

		rc = s.ClientRuntime(cliFirstIP1)
		assert.Nil(t, rc)
	})

	t.Run("remove_all", func(t *testing.T) {
		added, removed := s.BatchUpdateBySource(defSrc, []*client.Runtime{})
		require.Equal(t, 0, added)
		require.Equal(t, len(updatedClients), removed)
		require.Equal(t, 0, s.SizeRuntime())
	})
}
