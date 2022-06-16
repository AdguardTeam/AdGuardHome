package aghnet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSystemResolvers(
	t *testing.T,
	hostGenFunc HostGenFunc,
) (sr SystemResolvers) {
	t.Helper()

	var err error
	sr, err = NewSystemResolvers(hostGenFunc)
	require.NoError(t, err)
	require.NotNil(t, sr)

	return sr
}

func TestSystemResolvers_Get(t *testing.T) {
	sr := createTestSystemResolvers(t, nil)

	var rs []string
	require.NotPanics(t, func() {
		rs = sr.Get()
	})

	assert.NotEmpty(t, rs)
}

// TODO(e.burkov): Write tests for refreshWithTicker.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2846.
