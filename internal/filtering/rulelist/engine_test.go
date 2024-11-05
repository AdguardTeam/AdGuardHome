package rulelist_test

import (
	"net/http"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Refresh(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()

	fileURL, srvURL := newFilterLocations(t, cacheDir, testRuleTextBlocked, testRuleTextBlocked2)

	fileFlt := newFilter(t, fileURL, "File Filter")
	httpFlt := newFilter(t, srvURL, "HTTP Filter")

	eng := rulelist.NewEngine(&rulelist.EngineConfig{
		Logger:  slogutil.NewDiscardLogger(),
		Name:    "Engine",
		Filters: []*rulelist.Filter{fileFlt, httpFlt},
	})
	require.NotNil(t, eng)
	testutil.CleanupAndRequireSuccess(t, eng.Close)

	buf := make([]byte, rulelist.DefaultRuleBufSize)
	cli := &http.Client{
		Timeout: testTimeout,
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	err := eng.Refresh(ctx, buf, cli, cacheDir, rulelist.DefaultMaxRuleListSize)
	require.NoError(t, err)

	fltReq := &urlfilter.DNSRequest{
		Hostname: "blocked.example",
		Answer:   false,
		DNSType:  dns.TypeA,
	}

	fltRes, hasMatched := eng.FilterRequest(fltReq)
	assert.True(t, hasMatched)

	require.NotNil(t, fltRes)

	fltReq = &urlfilter.DNSRequest{
		Hostname: "blocked-2.example",
		Answer:   false,
		DNSType:  dns.TypeA,
	}

	fltRes, hasMatched = eng.FilterRequest(fltReq)
	assert.True(t, hasMatched)

	require.NotNil(t, fltRes)
}
