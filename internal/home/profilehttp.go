package home

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
)

// Theme is an enum of all allowed UI themes.
type Theme string

// Allowed [Theme] values.
//
// Keep in sync with client/src/helpers/constants.ts.
const (
	ThemeAuto  Theme = "auto"
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
)

// UnmarshalText implements [encoding.TextUnmarshaler] interface for *Theme.
func (t *Theme) UnmarshalText(b []byte) (err error) {
	switch string(b) {
	case "auto":
		*t = ThemeAuto
	case "dark":
		*t = ThemeDark
	case "light":
		*t = ThemeLight
	default:
		return fmt.Errorf("invalid theme %q, supported: %q, %q, %q", b, ThemeAuto, ThemeDark, ThemeLight)
	}

	return nil
}

// profileJSON is an object for /control/profile and /control/profile/update
// endpoints.
type profileJSON struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Theme    Theme  `json:"theme"`
}

// handleGetProfile is the handler for GET /control/profile endpoint.
func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	u := Context.auth.getCurrentUser(r)

	var resp profileJSON
	func() {
		config.RLock()
		defer config.RUnlock()

		resp = profileJSON{
			Name:     u.Name,
			Language: config.Language,
			Theme:    config.Theme,
		}
	}()

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// handlePutProfile is the handler for PUT /control/profile/update endpoint.
func handlePutProfile(w http.ResponseWriter, r *http.Request) {
	if aghhttp.WriteTextPlainDeprecated(w, r) {
		return
	}

	profileReq := &profileJSON{}
	err := json.NewDecoder(r.Body).Decode(profileReq)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	lang := profileReq.Language
	if !allowedLanguages.Has(lang) {
		aghhttp.Error(r, w, http.StatusBadRequest, "unknown language: %q", lang)

		return
	}

	theme := profileReq.Theme

	func() {
		config.Lock()
		defer config.Unlock()

		config.Language = lang
		config.Theme = theme
		log.Printf("home: language is set to %s", lang)
		log.Printf("home: theme is set to %s", theme)
	}()

	onConfigModified()
	aghhttp.OK(w)
}
