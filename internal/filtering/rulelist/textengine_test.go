package rulelist_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTextEngine(t *testing.T) {
	t.Parallel()

	eng, err := rulelist.NewTextEngine(&rulelist.TextEngineConfig{
		Name: "RulesEngine",
		Rules: []string{
			testRuleTextTitle,
			testRuleTextBlocked,
		},
		ID: testURLFilterID,
	})
	require.NoError(t, err)
	require.NotNil(t, eng)
	testutil.CleanupAndRequireSuccess(t, eng.Close)

	fltReq := &urlfilter.DNSRequest{
		Hostname: "blocked.example",
		Answer:   false,
		DNSType:  dns.TypeA,
	}

	fltRes, hasMatched := eng.FilterRequest(fltReq)
	assert.True(t, hasMatched)

	require.NotNil(t, fltRes)
	require.NotNil(t, fltRes.NetworkRule)

	assert.Equal(t, fltRes.NetworkRule.FilterListID, testURLFilterID)
}
