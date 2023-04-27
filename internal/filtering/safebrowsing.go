package filtering

import (
	"net/http"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
)

// Safe browsing and parental control methods.

// TODO(a.garipov): Unify with checkParental.
func (d *DNSFilter) checkSafeBrowsing(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled || !setts.SafeBrowsingEnabled {
		return Result{}, nil
	}

	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("safebrowsing lookup for %q", host)
	}

	res = Result{
		Rules: []*ResultRule{{
			Text:         "adguard-malware-shavar",
			FilterListID: SafeBrowsingListID,
		}},
		Reason:     FilteredSafeBrowsing,
		IsFiltered: true,
	}

	block, err := d.safeBrowsingChecker.Check(host)
	if !block || err != nil {
		return Result{}, err
	}

	return res, nil
}

// TODO(a.garipov): Unify with checkSafeBrowsing.
func (d *DNSFilter) checkParental(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled || !setts.ParentalEnabled {
		return Result{}, nil
	}

	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("parental lookup for %q", host)
	}

	res = Result{
		Rules: []*ResultRule{{
			Text:         "parental CATEGORY_BLACKLISTED",
			FilterListID: ParentalListID,
		}},
		Reason:     FilteredParental,
		IsFiltered: true,
	}

	block, err := d.parentalControlChecker.Check(host)
	if !block || err != nil {
		return Result{}, err
	}

	return res, nil
}

// setProtectedBool sets the value of a boolean pointer under a lock.  l must
// protect the value under ptr.
//
// TODO(e.burkov):  Make it generic?
func setProtectedBool(mu *sync.RWMutex, ptr *bool, val bool) {
	mu.Lock()
	defer mu.Unlock()

	*ptr = val
}

// protectedBool gets the value of a boolean pointer under a read lock.  l must
// protect the value under ptr.
//
// TODO(e.burkov):  Make it generic?
func protectedBool(mu *sync.RWMutex, ptr *bool) (val bool) {
	mu.RLock()
	defer mu.RUnlock()

	return *ptr
}

func (d *DNSFilter) handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.SafeBrowsingEnabled, true)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.SafeBrowsingEnabled, false)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	resp := &struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: protectedBool(&d.confLock, &d.Config.SafeBrowsingEnabled),
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

func (d *DNSFilter) handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.ParentalEnabled, true)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleParentalDisable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.ParentalEnabled, false)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	resp := &struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: protectedBool(&d.confLock, &d.Config.ParentalEnabled),
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}
