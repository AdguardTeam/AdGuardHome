package home

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRateLimiter_Cleanup(t *testing.T) {
	const key = "some-key"
	now := time.Now()

	testCases := []struct {
		name    string
		att     failedAuth
		wantExp bool
	}{{
		name: "expired",
		att: failedAuth{
			until: now.Add(-100 * time.Hour),
		},
		wantExp: true,
	}, {
		name: "nope_yet",
		att: failedAuth{
			until: now.Add(failedAuthTTL / 2),
		},
		wantExp: false,
	}, {
		name: "blocked",
		att: failedAuth{
			until: now.Add(100 * time.Hour),
		},
		wantExp: false,
	}}

	for _, tc := range testCases {
		ab := &authRateLimiter{
			failedAuths: map[string]failedAuth{
				key: tc.att,
			},
		}
		t.Run(tc.name, func(t *testing.T) {
			ab.cleanupLocked(now)
			if tc.wantExp {
				assert.Empty(t, ab.failedAuths)

				return
			}

			require.Len(t, ab.failedAuths, 1)

			_, ok := ab.failedAuths[key]
			require.True(t, ok)
		})
	}
}

func TestAuthRateLimiter_Check(t *testing.T) {
	key := string(net.IP{127, 0, 0, 1})
	const maxAtt = 1
	now := time.Now()

	testCases := []struct {
		until   time.Time
		name    string
		num     uint
		wantExp bool
	}{{
		until:   now.Add(-100 * time.Hour),
		name:    "expired",
		num:     0,
		wantExp: true,
	}, {
		until:   now.Add(failedAuthTTL),
		name:    "not_blocked_but_tracked",
		num:     0,
		wantExp: true,
	}, {
		until:   now,
		name:    "expired_but_stayed",
		num:     2,
		wantExp: true,
	}, {
		until:   now.Add(100 * time.Hour),
		name:    "blocked",
		num:     2,
		wantExp: false,
	}}

	for _, tc := range testCases {
		failedAuths := map[string]failedAuth{
			key: {
				num:   tc.num,
				until: tc.until,
			},
		}
		ab := &authRateLimiter{
			maxAttempts: maxAtt,
			failedAuths: failedAuths,
		}
		t.Run(tc.name, func(t *testing.T) {
			until := ab.check(key)

			if tc.wantExp {
				assert.LessOrEqual(t, until, time.Duration(0))
			} else {
				assert.Greater(t, until, time.Duration(0))
			}
		})
	}

	t.Run("non-existent", func(t *testing.T) {
		ab := &authRateLimiter{
			failedAuths: map[string]failedAuth{
				key + "smthng": {},
			},
		}

		until := ab.check(key)

		assert.Zero(t, until)
	})
}

func TestAuthRateLimiter_Inc(t *testing.T) {
	ip := net.IP{127, 0, 0, 1}
	key := string(ip)
	now := time.Now()
	const maxAtt = 2
	const blockDur = 15 * time.Minute

	testCases := []struct {
		until     time.Time
		wantUntil time.Time
		name      string
		num       uint
		wantNum   uint
	}{{
		name:      "only_inc",
		until:     now,
		wantUntil: now,
		num:       maxAtt - 1,
		wantNum:   maxAtt,
	}, {
		name:      "inc_and_block",
		until:     now,
		wantUntil: now.Add(failedAuthTTL),
		num:       maxAtt,
		wantNum:   maxAtt + 1,
	}}

	for _, tc := range testCases {
		failedAuths := map[string]failedAuth{
			key: {
				num:   tc.num,
				until: tc.until,
			},
		}
		ab := &authRateLimiter{
			blockDur:    blockDur,
			maxAttempts: maxAtt,
			failedAuths: failedAuths,
		}
		t.Run(tc.name, func(t *testing.T) {
			ab.inc(key)

			a, ok := ab.failedAuths[key]
			require.True(t, ok)

			assert.Equal(t, tc.wantNum, a.num)
			assert.LessOrEqual(t, tc.wantUntil.Unix(), a.until.Unix())
		})
	}

	t.Run("non-existent", func(t *testing.T) {
		ab := &authRateLimiter{
			blockDur:    blockDur,
			maxAttempts: maxAtt,
			failedAuths: map[string]failedAuth{},
		}

		ab.inc(key)

		a, ok := ab.failedAuths[key]
		require.True(t, ok)
		assert.EqualValues(t, 1, a.num)
	})
}

func TestAuthRateLimiter_Remove(t *testing.T) {
	const key = "some-key"

	failedAuths := map[string]failedAuth{
		key: {},
	}
	ab := &authRateLimiter{
		failedAuths: failedAuths,
	}

	ab.remove(key)

	assert.Empty(t, ab.failedAuths)
}
