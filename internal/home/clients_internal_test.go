package home

import (
	"net"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newClientsContainer is a helper that creates a new clients container for
// tests.
func newClientsContainer(t *testing.T) (c *clientsContainer) {
	t.Helper()

	c = &clientsContainer{
		testing: true,
	}

	require.NoError(t, c.Init(nil, client.EmptyDHCP{}, nil, nil, &filtering.Config{}))

	return c
}

func TestClientsCustomUpstream(t *testing.T) {
	clients := newClientsContainer(t)

	// Add client with upstreams.
	err := clients.storage.Add(&client.Persistent{
		Name: "client1",
		UID:  client.MustNewUID(),
		IPs:  []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1:2:3::4")},
		Upstreams: []string{
			"1.1.1.1",
			"[/example.org/]8.8.8.8",
		},
	})
	require.NoError(t, err)

	upsConf, err := clients.UpstreamConfigByID("1.2.3.4", net.DefaultResolver)
	assert.Nil(t, upsConf)
	assert.NoError(t, err)

	upsConf, err = clients.UpstreamConfigByID("1.1.1.1", net.DefaultResolver)
	require.NotNil(t, upsConf)
	assert.NoError(t, err)
}
