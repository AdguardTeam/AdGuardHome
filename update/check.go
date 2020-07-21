package update

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

const versionCheckPeriod = 8 * 60 * 60

// VersionInfo - VersionInfo
type VersionInfo struct {
	NewVersion           string
	Announcement         string
	AnnouncementURL      string
	SelfUpdateMinVersion string
	CanAutoUpdate        bool
	PackageURL           string
}

// GetVersionResponse - GetVersionResponse
func (u *Updater) GetVersionResponse(forceRecheck bool) (VersionInfo, error) {
	if !forceRecheck &&
		u.versionCheckLastTime.Unix()+versionCheckPeriod > time.Now().Unix() {
		return u.parseVersionResponse(u.versionJSON)
	}

	resp, err := u.Client.Get(u.VersionURL)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %s", u.VersionURL, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %s", u.VersionURL, err)
	}

	u.versionJSON = body
	u.versionCheckLastTime = time.Now()

	return u.parseVersionResponse(body)
}

func (u *Updater) parseVersionResponse(data []byte) (VersionInfo, error) {
	info := VersionInfo{}
	versionJSON := make(map[string]interface{})
	err := json.Unmarshal(data, &versionJSON)
	if err != nil {
		return info, fmt.Errorf("version.json: %s", err)
	}

	var ok1, ok2, ok3, ok4 bool
	info.NewVersion, ok1 = versionJSON["version"].(string)
	info.Announcement, ok2 = versionJSON["announcement"].(string)
	info.AnnouncementURL, ok3 = versionJSON["announcement_url"].(string)
	info.SelfUpdateMinVersion, ok4 = versionJSON["selfupdate_min_version"].(string)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return info, fmt.Errorf("version.json: invalid data")
	}

	var ok bool
	info.PackageURL, ok = u.getDownloadURL(versionJSON)
	if ok &&
		info.NewVersion != u.VersionString &&
		u.VersionString >= info.SelfUpdateMinVersion {
		info.CanAutoUpdate = true
	}

	return info, nil
}

// Get download URL for the current GOOS/GOARCH/ARMVersion
func (u *Updater) getDownloadURL(json map[string]interface{}) (string, bool) {
	var key string

	if u.Arch == "arm" && u.ARMVersion != "" {
		// the key is:
		// download_linux_armv5 for ARMv5
		// download_linux_armv6 for ARMv6
		// download_linux_armv7 for ARMv7
		key = fmt.Sprintf("download_%s_%sv%s", u.OS, u.Arch, u.ARMVersion)
	}

	val, ok := json[key]
	if !ok {
		// the key is download_linux_arm or download_linux_arm64 for regular ARM versions
		key = fmt.Sprintf("download_%s_%s", u.OS, u.Arch)
		val, ok = json[key]
	}

	if !ok {
		return "", false
	}

	return val.(string), true
}
