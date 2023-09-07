package aghnet_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/stretchr/testify/require"
)

func TestIgnoreEngine_Has(t *testing.T) {
	hostnames := []string{
		"*.example.com",
		"example.com",
		"|.^",
	}

	engine, err := aghnet.NewIgnoreEngine(hostnames)
	require.NotNil(t, engine)
	require.NoError(t, err)

	testCases := []struct {
		name   string
		host   string
		ignore bool
	}{{
		name:   "basic",
		host:   "example.com",
		ignore: true,
	}, {
		name:   "root",
		host:   ".",
		ignore: true,
	}, {
		name:   "wildcard",
		host:   "www.example.com",
		ignore: true,
	}, {
		name:   "not_ignored",
		host:   "something.com",
		ignore: false,
	}}

	for _, tc := range testCases {
		require.Equal(t, tc.ignore, engine.Has(tc.host))
	}
}
