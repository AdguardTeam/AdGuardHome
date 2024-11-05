package rulelist_test

import (
	"net/http"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_Refresh(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()

	allowedFileURL, _ := newFilterLocations(t, cacheDir, testRuleTextAllowed, "")
	allowedFlt := newFilter(t, allowedFileURL, "Allowed 1")

	blockedFileURL, _ := newFilterLocations(t, cacheDir, testRuleTextBlocked, "")
	blockedFlt := newFilter(t, blockedFileURL, "Blocked 1")

	strg, err := rulelist.NewStorage(&rulelist.StorageConfig{
		Logger: slogutil.NewDiscardLogger(),
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		CacheDir: cacheDir,
		AllowFilters: []*rulelist.Filter{
			allowedFlt,
		},
		BlockFilters: []*rulelist.Filter{
			blockedFlt,
		},
		CustomRules: []string{
			testRuleTextBlocked2,
		},
		MaxRuleListTextSize: 1 * datasize.KB,
	})
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, strg.Close)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	err = strg.Refresh(ctx)
	assert.NoError(t, err)
}
