package home

import (
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/stretchr/testify/require"
)

func TestApplyAdditionalFiltering_blockedServices(t *testing.T) {
	filtering.InitModule()

	var (
		globalBlockedServices  = []string{"ok"}
		clientBlockedServices  = []string{"ok", "mail_ru", "vk"}
		invalidBlockedServices = []string{"invalid"}

		err error
	)

	Context.filters, err = filtering.New(&filtering.Config{
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      globalBlockedServices,
		},
	}, nil)
	require.NoError(t, err)

	Context.clients.idIndex = map[string]*Client{
		"client_1": {
			UseOwnBlockedServices: false,
		},
		"client_2": {
			UseOwnBlockedServices: true,
		},
		"client_3": {
			BlockedServices:       clientBlockedServices,
			UseOwnBlockedServices: true,
		},
		"client_4": {
			BlockedServices:       invalidBlockedServices,
			UseOwnBlockedServices: true,
		},
	}

	testCases := []struct {
		name    string
		ip      net.IP
		id      string
		setts   *filtering.Settings
		wantLen int
	}{{
		name:    "global_settings",
		id:      "client_1",
		wantLen: len(globalBlockedServices),
	}, {
		name:    "custom_settings",
		id:      "client_2",
		wantLen: 0,
	}, {
		name:    "custom_settings_block",
		id:      "client_3",
		wantLen: len(clientBlockedServices),
	}, {
		name:    "custom_settings_invalid",
		id:      "client_4",
		wantLen: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setts := &filtering.Settings{}

			applyAdditionalFiltering(net.IP{1, 2, 3, 4}, tc.id, setts)
			require.Len(t, setts.ServicesRules, tc.wantLen)
		})
	}
}
