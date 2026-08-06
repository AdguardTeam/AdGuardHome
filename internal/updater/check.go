package updater

import (
	"context"
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
	"github.com/AdguardTeam/golibs/validate"
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
func (u *Updater) VersionInfo(ctx context.Context, forceRecheck bool) (vi VersionInfo, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	now := time.Now()
	recheckTime := u.prevCheckTime.Add(versionCheckPeriod)
	if !forceRecheck && now.Before(recheckTime) {
		u.logger.DebugContext(ctx, "version info recheck is not required yet")

		return u.prevCheckResult, u.prevCheckError
	}

	vcu := u.versionCheckURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, vcu, nil)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("constructing request to %s: %w", vcu, err)
	}

	u.logger.DebugContext(ctx, "requesting version data", "url", vcu)

	resp, err := u.client.Do(req)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("sending http request to %s: %w", vcu, err)
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return VersionInfo{}, fmt.Errorf(
			"got status code %d, want %d",
			resp.StatusCode,
			http.StatusOK,
		)
	}

	r := ioutil.LimitReader(resp.Body, maxVersionRespSize.Bytes())

	// This use of ReadAll is safe, because we just limited the appropriate
	// ReadCloser.
	body, err := io.ReadAll(r)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("reading response from %s: %w", vcu, err)
	}

	u.prevCheckTime = now
	u.prevCheckResult, u.prevCheckError = u.parseVersionResponse(ctx, body)

	return u.prevCheckResult, u.prevCheckError
}

// parseVersionResponse parses version-related data and unmarshals it into the
// [VersionInfo] structure.
func (u *Updater) parseVersionResponse(
	ctx context.Context,
	data []byte,
) (vi VersionInfo, err error) {
	info := VersionInfo{
		CanAutoUpdate: aghalg.NBFalse,
	}
	versionJSON := map[string]string{
		"version":          "",
		"announcement":     "",
		"announcement_url": "",
	}
	err = json.Unmarshal(data, &versionJSON)
	if err != nil {
		return info, fmt.Errorf("version.json: %w", err)
	}

	for k, v := range versionJSON {
		err = validate.NotEmpty("version_json_value", v)
		if err != nil {
			return info, fmt.Errorf("bad value for %q key: %w", k, err)
		}
	}

	info.NewVersion = versionJSON["version"]
	info.Announcement = versionJSON["announcement"]
	info.AnnouncementURL = versionJSON["announcement_url"]

	packageURL, key, found := u.downloadURL(ctx, versionJSON)
	if !found {
		return info, fmt.Errorf("version.json: bad key %q: %w", key, errors.ErrNoValue)
	}

	isNewVersion := info.NewVersion != u.version
	if isNewVersion {
		u.logger.InfoContext(
			ctx,
			"a new version is available",
			"current_version", u.version,
			"new_version", info.NewVersion,
		)
	} else {
		u.logger.DebugContext(
			ctx,
			"the current version is up-to-date",
			"current_version", u.version,
		)
	}

	info.CanAutoUpdate = aghalg.BoolToNullBool(isNewVersion)

	u.newVersion = info.NewVersion
	u.packageURL = packageURL

	return info, nil
}

// downloadURL returns the download URL for current build as well as its key in
// versionObj.  If the key is not found, it additionally prints an informative
// log message.
func (u *Updater) downloadURL(
	ctx context.Context,
	versionObj map[string]string,
) (dlURL, key string, ok bool) {
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

	u.logger.ErrorContext(ctx, "key not found", "missing", key, "got", keys)

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
