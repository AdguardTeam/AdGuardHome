package aghnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSystemResolvers(
	t *testing.T,
	refreshDur time.Duration,
	hostGenFunc HostGenFunc,
) (sr SystemResolvers) {
	t.Helper()

	var err error
	sr, err = NewSystemResolvers(refreshDur, hostGenFunc)
	require.NoError(t, err)
	require.NotNil(t, sr)

	return sr
}

func TestSystemResolvers_Get(t *testing.T) {
	sr := createTestSystemResolvers(t, 0, nil)
	assert.NotEmpty(t, sr.Get())
}

// TODO(e.burkov): Write tests for refreshWithTicker.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2846.
