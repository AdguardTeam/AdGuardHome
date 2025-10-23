package home

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// newClientsContainer is a helper that creates a new clients container for
// tests.
func newClientsContainer(tb testing.TB) (c *clientsContainer) {
	tb.Helper()

	c = &clientsContainer{}

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
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
		aghhttp.EmptyRegistrar{},
	)

	require.NoError(tb, err)

	return c
}
