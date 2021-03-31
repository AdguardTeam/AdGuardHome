// +build !windows

package aghnet

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSystemResolversImp(
	t *testing.T,
	refreshDur time.Duration,
	hostGenFunc HostGenFunc,
) (imp *systemResolvers) {
	t.Helper()

	sr := createTestSystemResolvers(t, refreshDur, hostGenFunc)

	var ok bool
	imp, ok = sr.(*systemResolvers)
	require.True(t, ok)

	return imp
}

func TestSystemResolvers_Refresh(t *testing.T) {
	t.Run("expected_error", func(t *testing.T) {
		sr := createTestSystemResolvers(t, 0, nil)

		assert.NoError(t, sr.refresh())
	})

	t.Run("unexpected_error", func(t *testing.T) {
		_, err := NewSystemResolvers(0, func() string {
			return "127.0.0.1::123"
		})
		assert.Error(t, err)
	})
}

func TestSystemResolvers_DialFunc(t *testing.T) {
	imp := createTestSystemResolversImp(t, 0, nil)

	testCases := []struct {
		name    string
		address string
		want    error
	}{{
		name:    "valid",
		address: "127.0.0.1",
		want:    fakeDialErr,
	}, {
		name:    "invalid_split_host",
		address: "127.0.0.1::123",
		want:    badAddrPassedErr,
	}, {
		name:    "invalid_parse_ip",
		address: "not-ip",
		want:    badAddrPassedErr,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := imp.dialFunc(context.Background(), "", tc.address)

			require.Nil(t, conn)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
