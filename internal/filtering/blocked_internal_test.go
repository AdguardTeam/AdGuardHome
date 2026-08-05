package filtering

import (
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_CheckHost_blockedServiceDNSRewrite(t *testing.T) {
	const (
		serviceID = "icloud_private_relay"
		host      = "mask.icloud.com"
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	initBlockedServices(ctx, testLogger)

	d, setts := newForTest(t, nil, nil)
	t.Cleanup(d.Close)
	d.ApplyBlockedServicesList(setts, []string{serviceID})

	res, err := d.CheckHost(host, dns.TypeA, setts)
	require.NoError(t, err)

	assert.Equal(t, FilteredBlockedService, res.Reason)
	assert.Equal(t, serviceID, res.ServiceName)

	require.NotNil(t, res.DNSRewriteResult)
	assert.Equal(t, dns.RcodeNameError, res.DNSRewriteResult.RCode)
}

func TestDNSFilter_CheckHost_blockedServicePrecedence(t *testing.T) {
	const (
		serviceID = "facebook"
		host      = "facebook.com"
		rule      = "||facebook.com^"
	)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	initBlockedServices(ctx, testLogger)

	d, setts := newForTest(t, nil, []Filter{{
		ID:   1,
		Data: []byte(rule),
	}})
	t.Cleanup(d.Close)
	d.ApplyBlockedServicesList(setts, []string{serviceID})

	res, err := d.CheckHost(host, dns.TypeA, setts)
	require.NoError(t, err)

	assert.Equal(t, FilteredBlockList, res.Reason)
	assert.Empty(t, res.ServiceName)
	assert.Nil(t, res.DNSRewriteResult)

	require.Len(t, res.Rules, 1)
	assert.Equal(t, rule, res.Rules[0].Text)
}
