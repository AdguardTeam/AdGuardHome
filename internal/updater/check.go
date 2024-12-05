package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/log"
	"github.com/c2h5oh/datasize"
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

// maxVersionRespSize is the maximum length in bytes for version information
// response.
const maxVersionRespSize datasize.ByteSize = 64 * datasize.KB

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

	r := ioutil.LimitReader(resp.Body, maxVersionRespSize.Bytes())

	// This use of ReadAll is safe, because we just limited the appropriate
	// ReadCloser.
	body, err := io.ReadAll(r)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("updater: HTTP GET %s: %w", vcu, err)
	}

	u.prevCheckTime = now
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

	for k, v := range versionJSON {
		if v == "" {
			return info, fmt.Errorf("version.json: bad data: value for key %q is empty", k)
		}
	}

	info.NewVersion = versionJSON["version"]
	info.Announcement = versionJSON["announcement"]
	info.AnnouncementURL = versionJSON["announcement_url"]

	packageURL, key, found := u.downloadURL(versionJSON)
	if !found {
		return info, fmt.Errorf("version.json: no package URL: key %q not found in object", key)
	}

	info.CanAutoUpdate = aghalg.BoolToNullBool(info.NewVersion != u.version)

	u.newVersion = info.NewVersion
	u.packageURL = packageURL

	return info, nil
}

// downloadURL returns the download URL for current build as well as its key in
// versionObj.  If the key is not found, it additionally prints an informative
// log message.
func (u *Updater) downloadURL(versionObj map[string]string) (dlURL, key string, ok bool) {
	if u.goarch == "arm" && u.goarm != "" {
		key = fmt.Sprintf("download_%s_%sv%s", u.goos, u.goarch, u.goarm)
	} else if isMIPS(u.goarch) && u.gomips != "" {
		key = fmt.Sprintf("download_%s_%s_%s", u.goos, u.goarch, u.gomips)
	} else {
		key = fmt.Sprintf("download_%s_%s", u.goos, u.goarch)
	}

	dlURL, ok = versionObj[key]
	if ok {
		return dlURL, key, true
	}

	keys := slices.Sorted(maps.Keys(versionObj))

	log.Error("updater: key %q not found; got keys %q", key, keys)

	return "", key, false
}

// isMIPS returns true if arch is any MIPS architecture.
func isMIPS(arch string) (ok bool) {
	switch arch {
	case
		"mips",
		"mips64",
		"mips64le",
		"mipsle":
		return true
	default:
		return false
	}
}
