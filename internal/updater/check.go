package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/golibs/errors"
)

// TODO(a.garipov): Make configurable.
const versionCheckPeriod = 8 * time.Hour

// VersionInfo contains information about a new version.
type VersionInfo struct {
	NewVersion      string `json:"new_version,omitempty"`
	Announcement    string `json:"announcement,omitempty"`
	AnnouncementURL string `json:"announcement_url,omitempty"`
	// TODO(a.garipov): See if the frontend actually still cares about
	// nullability.
	CanAutoUpdate aghalg.NullBool `json:"can_autoupdate,omitempty"`
}

// MaxResponseSize is responses on server's requests maximum length in bytes.
const MaxResponseSize = 64 * 1024

// VersionInfo downloads the latest version information.  If forceRecheck is
// false and there are cached results, those results are returned.
func (u *Updater) VersionInfo(forceRecheck bool) (vi VersionInfo, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()
	recheckTime := u.prevCheckTime.Add(versionCheckPeriod)
	if !forceRecheck && now.Before(recheckTime) {
		return u.prevCheckResult, u.prevCheckError
	}

	var resp *http.Response
	vcu := u.versionCheckURL
	resp, err = u.client.Get(vcu)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %w", vcu, err)
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	var r io.Reader
	r, err = aghio.LimitReader(resp.Body, MaxResponseSize)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: LimitReadCloser: %w", err)
	}

	// This use of ReadAll is safe, because we just limited the appropriate
	// ReadCloser.
	body, err := io.ReadAll(r)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %w", vcu, err)
	}

	u.prevCheckTime = time.Now()
	u.prevCheckResult, u.prevCheckError = u.parseVersionResponse(body)

	return u.prevCheckResult, u.prevCheckError
}

func (u *Updater) parseVersionResponse(data []byte) (VersionInfo, error) {
	info := VersionInfo{
		CanAutoUpdate: aghalg.NBFalse,
	}
	versionJSON := map[string]string{
		"version":          "",
		"announcement":     "",
		"announcement_url": "",
	}
	err := json.Unmarshal(data, &versionJSON)
	if err != nil {
		return info, fmt.Errorf("version.json: %w", err)
	}

	for _, v := range versionJSON {
		if v == "" {
			return info, fmt.Errorf("version.json: invalid data")
		}
	}

	info.NewVersion = versionJSON["version"]
	info.Announcement = versionJSON["announcement"]
	info.AnnouncementURL = versionJSON["announcement_url"]

	packageURL, ok := u.downloadURL(versionJSON)
	info.CanAutoUpdate = aghalg.BoolToNullBool(ok && info.NewVersion != u.version)

	u.newVersion = info.NewVersion
	u.packageURL = packageURL

	return info, nil
}

// downloadURL returns the download URL for current build.
func (u *Updater) downloadURL(json map[string]string) (string, bool) {
	var key string

	if u.goarch == "arm" && u.goarm != "" {
		key = fmt.Sprintf("download_%s_%sv%s", u.goos, u.goarch, u.goarm)
	} else if u.goarch == "mips" && u.gomips != "" {
		key = fmt.Sprintf("download_%s_%s_%s", u.goos, u.goarch, u.gomips)
	}

	val, ok := json[key]
	if !ok {
		key = fmt.Sprintf("download_%s_%s", u.goos, u.goarch)
		val, ok = json[key]
	}

	if !ok {
		return "", false
	}

	return val, true
}
