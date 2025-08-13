package home

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// newClientsContainer is a helper that creates a new clients container for
// tests.
func newClientsContainer(t *testing.T) (c *clientsContainer) {
	t.Helper()

	c = &clientsContainer{}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	err := c.Init(
		ctx,
		testLogger,
		nil,
		client.EmptyDHCP{},
		nil,
		nil,
		&filtering.Config{
			Logger: testLogger,
		},
		newSignalHandler(testLogger, nil, nil),
		agh.EmptyConfigModifier{},
	)

	require.NoError(t, err)

	return c
}
