package home

import (
	"sync"
	"time"
)

// failedAuthTTL is the period of time for which the failed attempt will stay in
// cache.
const failedAuthTTL = 1 * time.Minute

// failedAuth is an entry of authRateLimiter's cache.
type failedAuth struct {
	until time.Time
	num   uint
}

// authRateLimiter used to cache failed authentication attempts.
type authRateLimiter struct {
	failedAuths map[string]failedAuth
	// failedAuthsLock protects failedAuths.
	failedAuthsLock sync.Mutex
	blockDur        time.Duration
	maxAttempts     uint
}

// newAuthRateLimiter returns properly initialized *authRateLimiter.
func newAuthRateLimiter(blockDur time.Duration, maxAttempts uint) (ab *authRateLimiter) {
	return &authRateLimiter{
		failedAuths: make(map[string]failedAuth),
		blockDur:    blockDur,
		maxAttempts: maxAttempts,
	}
}

// cleanupLocked checks each blocked users removing ones with expired TTL.  For
// internal use only.
func (ab *authRateLimiter) cleanupLocked(now time.Time) {
	for k, v := range ab.failedAuths {
		if now.After(v.until) {
			delete(ab.failedAuths, k)
		}
	}
}

// checkLocked checks the attempter for it's state.  For internal use only.
func (ab *authRateLimiter) checkLocked(usrID string, now time.Time) (left time.Duration) {
	a, ok := ab.failedAuths[usrID]
	if !ok {
		return 0
	}

	if a.num < ab.maxAttempts {
		return 0
	}

	return a.until.Sub(now)
}

// check returns the time left until unblocking.  The nonpositive result should
// be interpreted as not blocked attempter.
func (ab *authRateLimiter) check(usrID string) (left time.Duration) {
	now := time.Now()

	ab.failedAuthsLock.Lock()
	defer ab.failedAuthsLock.Unlock()

	ab.cleanupLocked(now)

	return ab.checkLocked(usrID, now)
}

// incLocked increments the number of unsuccessful attempts for attempter with
// usrID and updates it's blocking moment if needed.  For internal use only.
func (ab *authRateLimiter) incLocked(usrID string, now time.Time) {
	until := now.Add(failedAuthTTL)
	var attNum uint = 1

	a, ok := ab.failedAuths[usrID]
	if ok {
		until = a.until
		attNum = a.num + 1
	}
	if attNum >= ab.maxAttempts {
		until = now.Add(ab.blockDur)
	}

	ab.failedAuths[usrID] = failedAuth{
		num:   attNum,
		until: until,
	}
}

// inc updates the failed attempt in cache.
func (ab *authRateLimiter) inc(usrID string) {
	now := time.Now()

	ab.failedAuthsLock.Lock()
	defer ab.failedAuthsLock.Unlock()

	ab.incLocked(usrID, now)
}

// remove stops any tracking and any blocking of the user.
func (ab *authRateLimiter) remove(usrID string) {
	ab.failedAuthsLock.Lock()
	defer ab.failedAuthsLock.Unlock()

	delete(ab.failedAuths, usrID)
}
