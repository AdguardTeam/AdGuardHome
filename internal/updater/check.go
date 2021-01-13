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
	NewVersion           string
	Announcement         string
	AnnouncementURL      string
	SelfUpdateMinVersion string
	CanAutoUpdate        bool
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
	info := VersionInfo{}
	versionJSON := make(map[string]interface{})
	err := json.Unmarshal(data, &versionJSON)
	if err != nil {
		return info, fmt.Errorf("version.json: %w", err)
	}

	var ok1, ok2, ok3, ok4 bool
	info.NewVersion, ok1 = versionJSON["version"].(string)
	info.Announcement, ok2 = versionJSON["announcement"].(string)
	info.AnnouncementURL, ok3 = versionJSON["announcement_url"].(string)
	info.SelfUpdateMinVersion, ok4 = versionJSON["selfupdate_min_version"].(string)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return info, fmt.Errorf("version.json: invalid data")
	}

	packageURL, ok := u.downloadURL(versionJSON)
	if ok &&
		info.NewVersion != u.version &&
		strings.TrimPrefix(u.version, "v") >= strings.TrimPrefix(info.SelfUpdateMinVersion, "v") {
		info.CanAutoUpdate = true
	}

	u.newVersion = info.NewVersion
	u.packageURL = packageURL

	return info, nil
}

// downloadURL returns the download URL for current build.
func (u *Updater) downloadURL(json map[string]interface{}) (string, bool) {
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

	return val.(string), true
}
