package updater

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
)

// TODO(a.garipov): Make configurable.
const versionCheckPeriod = 8 * time.Hour

// VersionInfo contains information about a new version.
type VersionInfo struct {
	NewVersion           string `json:"new_version,omitempty"`
	Announcement         string `json:"announcement,omitempty"`
	AnnouncementURL      string `json:"announcement_url,omitempty"`
	SelfUpdateMinVersion string `json:"-"`
	CanAutoUpdate        *bool  `json:"can_autoupdate,omitempty"`
}

// MaxResponseSize is responses on server's requests maximum length in bytes.
const MaxResponseSize = 64 * 1024

// VersionInfo downloads the latest version information.  If forceRecheck is
// false and there are cached results, those results are returned.
func (u *Updater) VersionInfo(forceRecheck bool) (VersionInfo, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()
	recheckTime := u.prevCheckTime.Add(versionCheckPeriod)
	if !forceRecheck && now.Before(recheckTime) {
		return u.prevCheckResult, u.prevCheckError
	}

	vcu := u.versionCheckURL
	resp, err := u.client.Get(vcu)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %w", vcu, err)
	}
	defer resp.Body.Close()

	resp.Body, err = aghio.LimitReadCloser(resp.Body, MaxResponseSize)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: LimitReadCloser: %w", err)
	}
	defer resp.Body.Close()

	// This use of ReadAll is safe, because we just limited the appropriate
	// ReadCloser.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %w", vcu, err)
	}

	u.prevCheckTime = time.Now()
	u.prevCheckResult, u.prevCheckError = u.parseVersionResponse(body)

	return u.prevCheckResult, u.prevCheckError
}

func (u *Updater) parseVersionResponse(data []byte) (VersionInfo, error) {
	var canAutoUpdate bool
	info := VersionInfo{
		CanAutoUpdate: &canAutoUpdate,
	}
	versionJSON := map[string]string{
		"version":                "",
		"announcement":           "",
		"announcement_url":       "",
		"selfupdate_min_version": "",
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
	info.SelfUpdateMinVersion = versionJSON["selfupdate_min_version"]

	packageURL, ok := u.downloadURL(versionJSON)
	if ok &&
		info.NewVersion != u.version &&
		strings.TrimPrefix(u.version, "v") >= strings.TrimPrefix(info.SelfUpdateMinVersion, "v") {
		canAutoUpdate = true
	}

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
