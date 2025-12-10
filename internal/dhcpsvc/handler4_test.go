package dhcpsvc_test

import (
	"context"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil/servicetest"
)

func TestDHCPServer_ServeEther4(t *testing.T) {
	t.Parallel()

	ndMgr := &testNetworkDeviceManager{
		onOpen: func(
			ctx context.Context,
			conf *dhcpsvc.NetworkDeviceConfig,
		) (nd dhcpsvc.NetworkDevice, err error) {
			return &testNetworkDevice{
				// TODO(e.burkov):  !! implement ReadPacketData, WritePacketData, and LinkType
			}, nil
		},
	}

	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		NetworkDeviceManager: ndMgr,
		Enabled:              true,
	})
	servicetest.RequireRun(t, srv, testTimeout)

	testCases := []struct {
		name string
		// TODO(e.burkov):  !! define other fields.
	}{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO(e.burkov):  !! implement a test
		})
	}
}
