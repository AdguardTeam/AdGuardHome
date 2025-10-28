package home

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
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
func (web *webAPI) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var name string

	if !web.auth.isGLiNet && !web.auth.isUserless {
		u, ok := webUserFromContext(ctx)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		name = string(u.Login)
	}

	var resp profileJSON
	func() {
		config.RLock()
		defer config.RUnlock()

		resp = profileJSON{
			Name:     name,
			Language: config.Language,
			Theme:    config.Theme,
		}
	}()

	aghhttp.WriteJSONResponseOK(ctx, web.logger, w, r, resp)
}

// handlePutProfile is the handler for PUT /control/profile/update endpoint.
func (web *webAPI) handlePutProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := web.logger

	if aghhttp.WriteTextPlainDeprecated(ctx, l, w, r) {
		return
	}

	profileReq := &profileJSON{}
	err := json.NewDecoder(r.Body).Decode(profileReq)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	lang := profileReq.Language
	if !allowedLanguages.Has(lang) {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "unknown language: %q", lang)

		return
	}

	theme := profileReq.Theme

	changed := false
	func() {
		config.Lock()
		defer config.Unlock()

		if config.Language == lang && config.Theme == theme {
			l.DebugContext(ctx, "updating profile; no changes")

			return
		}

		changed = true
		config.Language = lang
		config.Theme = theme
		l.InfoContext(ctx, "profile updated", "lang", lang, "theme", theme)
	}()

	if changed {
		web.confModifier.Apply(ctx)
	}

	aghhttp.OK(ctx, l, w)
}
