package home

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// newClientsContainer is a helper that creates a new clients container for
// tests.
func newClientsContainer(t *testing.T) (c *clientsContainer) {
	t.Helper()

	c = &clientsContainer{
		testing: true,
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	err := c.Init(
		ctx,
		slogutil.NewDiscardLogger(),
		nil,
		client.EmptyDHCP{},
		nil,
		nil,
		&filtering.Config{},
		newSignalHandler(nil, nil),
	)

	require.NoError(t, err)

	return c
}
