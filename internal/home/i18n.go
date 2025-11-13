package home

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/container"
)

// TODO(a.garipov): Get rid of a global or generate from .twosky.json.
var allowedLanguages = container.NewMapSet(
	"ar",
	"be",
	"bg",
	"cs",
	"da",
	"de",
	"en",
	"es",
	"fa",
	"fi",
	"fr",
	"hr",
	"hu",
	"id",
	"it",
	"ja",
	"ko",
	"nl",
	"no",
	"pl",
	"pt-br",
	"pt-pt",
	"ro",
	"ru",
	"si-lk",
	"sk",
	"sl",
	"sr-cs",
	"sv",
	"th",
	"tr",
	"uk",
	"vi",
	"zh-cn",
	"zh-hk",
	"zh-tw",
)

// languageJSON is the JSON structure for language requests and responses.
type languageJSON struct {
	Language string `json:"language"`
}

// handleI18nCurrentLanguage is the handler for the GET
// /control/i18n/current_language HTTP API.
//
// TODO(d.kolyshev): Deprecated, remove it later.
func (web *webAPI) handleI18nCurrentLanguage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := web.logger

	l.InfoContext(ctx, "current language", "lang", config.Language)

	aghhttp.WriteJSONResponseOK(ctx, l, w, r, &languageJSON{
		Language: config.Language,
	})
}

// handleI18nChangeLanguage is the handler for the POST
// /control/i18n/change_language HTTP API.
//
// TODO(d.kolyshev): Deprecated, remove it later.
func (web *webAPI) handleI18nChangeLanguage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := web.logger

	if aghhttp.WriteTextPlainDeprecated(ctx, l, w, r) {
		return
	}

	langReq := &languageJSON{}
	err := json.NewDecoder(r.Body).Decode(langReq)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusInternalServerError, "reading req: %s", err)

		return
	}

	lang := langReq.Language
	if !allowedLanguages.Has(lang) {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "unknown language: %q", lang)

		return
	}

	func() {
		config.Lock()
		defer config.Unlock()

		config.Language = lang
		l.InfoContext(ctx, "language is updated", "lang", lang)
	}()

	web.confModifier.Apply(ctx)

	aghhttp.OK(ctx, l, w)
}
