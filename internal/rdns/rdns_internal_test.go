package rdns

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testExchanger func(
	ctx context.Context,
	ip netip.Addr,
) (host string, ttl time.Duration, err error)

// Exchange implements the [Exchanger] interface for testExchanger.
func (f testExchanger) Exchange(
	ctx context.Context,
	ip netip.Addr,
) (host string, ttl time.Duration, err error) {
	return f(ctx, ip)
}

func TestDefault_Process_errorRetry(t *testing.T) {
	const shortCacheTTL = 30 * time.Second

	testCases := []struct {
		err      error
		name     string
		cacheTTL time.Duration
		wantTTL  time.Duration
	}{
		{
			name:     "temporary_failure",
			err:      errors.Error("service unavailable"),
			cacheTTL: time.Hour,
			wantTTL:  maxErrorCacheTTL,
		},
		{
			name:     "temporary_failure_short_cache",
			err:      errors.Error("service unavailable"),
			cacheTTL: shortCacheTTL,
			wantTTL:  shortCacheTTL,
		},
		{
			name:     "no_data",
			err:      ErrNoData,
			cacheTTL: time.Hour,
			wantTTL:  time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				available bool
				hits      int
			)

			wantHost := "router.example"
			exchanger := testExchanger(func(
				_ context.Context,
				_ netip.Addr,
			) (host string, ttl time.Duration, err error) {
				hits++
				if !available {
					return "", 0, tc.err
				}

				return wantHost, tc.cacheTTL, nil
			})

			r := New(&Config{
				Logger:    slogutil.NewDiscardLogger(),
				Exchanger: exchanger,
				CacheSize: 1,
				CacheTTL:  tc.cacheTTL,
			})

			ctx := testutil.ContextWithTimeout(t, time.Second)
			ip := netip.MustParseAddr("192.168.1.1")
			started := time.Now()

			got, changed := r.Process(ctx, ip)
			finished := time.Now()
			require.True(t, changed)
			assert.Empty(t, got)
			require.Equal(t, 1, hits)

			val, err := r.cache.Get(ip)
			require.NoError(t, err)

			item := val.(*cacheItem)
			assert.False(t, item.expiry.Before(started.Add(tc.wantTTL)))
			assert.False(t, item.expiry.After(finished.Add(tc.wantTTL)))

			// The failure is suppressed during the retry window.
			got, changed = r.Process(ctx, ip)
			assert.False(t, changed)
			assert.Empty(t, got)
			assert.Equal(t, 1, hits)

			// Simulate the retry delay passing and the private resolver becoming
			// ready after AdGuard Home startup.
			item.expiry = time.Now().Add(-time.Second)
			available = true

			got, changed = r.Process(ctx, ip)
			require.True(t, changed)
			assert.Equal(t, wantHost, got)
			assert.Equal(t, 2, hits)
		})
	}
}
