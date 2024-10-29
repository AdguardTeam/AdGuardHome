package home

import (
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testIPv4 = netip.AddrFrom4([4]byte{1, 2, 3, 4})

// newStorage is a helper function that returns a client storage filled with
// persistent clients.  It also generates a UID for each client.
func newStorage(tb testing.TB, clients []*client.Persistent) (s *client.Storage) {
	tb.Helper()

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	s, err := client.NewStorage(ctx, &client.StorageConfig{
		Logger: slogutil.NewDiscardLogger(),
	})
	require.NoError(tb, err)

	for _, p := range clients {
		p.UID = client.MustNewUID()
		require.NoError(tb, s.Add(ctx, p))
	}

	return s
}

func TestApplyAdditionalFiltering(t *testing.T) {
	var err error

	Context.filters, err = filtering.New(&filtering.Config{
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
		},
	}, nil)
	require.NoError(t, err)

	Context.clients.storage = newStorage(t, []*client.Persistent{{
		Name:                "default",
		ClientIDs:           []string{"default"},
		UseOwnSettings:      false,
		SafeSearchConf:      filtering.SafeSearchConfig{Enabled: false},
		FilteringEnabled:    false,
		SafeBrowsingEnabled: false,
		ParentalEnabled:     false,
	}, {
		Name:                "custom_filtering",
		ClientIDs:           []string{"custom_filtering"},
		UseOwnSettings:      true,
		SafeSearchConf:      filtering.SafeSearchConfig{Enabled: true},
		FilteringEnabled:    true,
		SafeBrowsingEnabled: true,
		ParentalEnabled:     true,
	}, {
		Name:                "partial_custom_filtering",
		ClientIDs:           []string{"partial_custom_filtering"},
		UseOwnSettings:      true,
		SafeSearchConf:      filtering.SafeSearchConfig{Enabled: true},
		FilteringEnabled:    true,
		SafeBrowsingEnabled: false,
		ParentalEnabled:     false,
	}})

	testCases := []struct {
		name                string
		id                  string
		FilteringEnabled    assert.BoolAssertionFunc
		SafeSearchEnabled   assert.BoolAssertionFunc
		SafeBrowsingEnabled assert.BoolAssertionFunc
		ParentalEnabled     assert.BoolAssertionFunc
	}{{
		name:                "global_settings",
		id:                  "default",
		FilteringEnabled:    assert.False,
		SafeSearchEnabled:   assert.False,
		SafeBrowsingEnabled: assert.False,
		ParentalEnabled:     assert.False,
	}, {
		name:                "custom_settings",
		id:                  "custom_filtering",
		FilteringEnabled:    assert.True,
		SafeSearchEnabled:   assert.True,
		SafeBrowsingEnabled: assert.True,
		ParentalEnabled:     assert.True,
	}, {
		name:                "partial",
		id:                  "partial_custom_filtering",
		FilteringEnabled:    assert.True,
		SafeSearchEnabled:   assert.True,
		SafeBrowsingEnabled: assert.False,
		ParentalEnabled:     assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setts := &filtering.Settings{}

			applyAdditionalFiltering(testIPv4, tc.id, setts)
			tc.FilteringEnabled(t, setts.FilteringEnabled)
			tc.SafeSearchEnabled(t, setts.SafeSearchEnabled)
			tc.SafeBrowsingEnabled(t, setts.SafeBrowsingEnabled)
			tc.ParentalEnabled(t, setts.ParentalEnabled)
		})
	}
}

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

	Context.clients.storage = newStorage(t, []*client.Persistent{{
		Name:                  "default",
		ClientIDs:             []string{"default"},
		UseOwnBlockedServices: false,
	}, {
		Name:      "no_services",
		ClientIDs: []string{"no_services"},
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
		},
		UseOwnBlockedServices: true,
	}, {
		Name:      "services",
		ClientIDs: []string{"services"},
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      clientBlockedServices,
		},
		UseOwnBlockedServices: true,
	}, {
		Name:      "invalid_services",
		ClientIDs: []string{"invalid_services"},
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      invalidBlockedServices,
		},
		UseOwnBlockedServices: true,
	}, {
		Name:      "allow_all",
		ClientIDs: []string{"allow_all"},
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.FullWeekly(),
			IDs:      clientBlockedServices,
		},
		UseOwnBlockedServices: true,
	}})

	testCases := []struct {
		name    string
		id      string
		wantLen int
	}{{
		name:    "global_settings",
		id:      "default",
		wantLen: len(globalBlockedServices),
	}, {
		name:    "custom_settings",
		id:      "no_services",
		wantLen: 0,
	}, {
		name:    "custom_settings_block",
		id:      "services",
		wantLen: len(clientBlockedServices),
	}, {
		name:    "custom_settings_invalid",
		id:      "invalid_services",
		wantLen: 0,
	}, {
		name:    "custom_settings_inactive_schedule",
		id:      "allow_all",
		wantLen: 0,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setts := &filtering.Settings{}

			applyAdditionalFiltering(testIPv4, tc.id, setts)
			require.Len(t, setts.ServicesRules, tc.wantLen)
		})
	}
}
