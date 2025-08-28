package home

import (
	"sync"
	"time"
)

// failedAuthTTL is the period of time for which the failed attempt will stay in
// cache.
const failedAuthTTL = 1 * time.Minute

// loginRateLimiter is an interface for rate limiting login attempts.
type loginRateLimiter interface {
	// check returns the duration of time left until a user is unblocked.
	// A non-positive result indicates that the user is not blocked.
	check(usrID string) (left time.Duration)

	// inc records a failed login attempt for the specified user.
	inc(usrID string)

	// remove stops tracking and blocking of the specified user.
	remove(usrID string)
}

// emptyRateLimiter is the [loginRateLimiter] interface implementation that does
// nothing.
type emptyRateLimiter struct{}

// type check
var _ emptyRateLimiter = emptyRateLimiter{}

// check implements the [loginRateLimiter] interface for emptyRateLimiter.  It
// always returns zero.
func (rl emptyRateLimiter) check(_ string) (left time.Duration) {
	return 0
}

// inc implements the [loginRateLimiter] interface for emptyRateLimiter.
func (rl emptyRateLimiter) inc(_ string) {}

// remove implements the [loginRateLimiter] interface for emptyRateLimiter.
func (rl emptyRateLimiter) remove(_ string) {}

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

// type check
var _ loginRateLimiter = (*authRateLimiter)(nil)

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

// check implements the [loginRateLimiter] interface for *authRateLimiter.
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

// inc implements the [loginRateLimiter] interface for *authRateLimiter.
func (ab *authRateLimiter) inc(usrID string) {
	now := time.Now()

	ab.failedAuthsLock.Lock()
	defer ab.failedAuthsLock.Unlock()

	ab.incLocked(usrID, now)
}

// remove implements the [loginRateLimiter] interface for *authRateLimiter.
func (ab *authRateLimiter) remove(usrID string) {
	ab.failedAuthsLock.Lock()
	defer ab.failedAuthsLock.Unlock()

	delete(ab.failedAuths, usrID)
}
