package home

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/log"
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

// TODO(d.kolyshev): Deprecated, remove it later.
func handleI18nCurrentLanguage(w http.ResponseWriter, r *http.Request) {
	log.Printf("home: language is %s", config.Language)

	aghhttp.WriteJSONResponseOK(w, r, &languageJSON{
		Language: config.Language,
	})
}

// TODO(d.kolyshev): Deprecated, remove it later.
func handleI18nChangeLanguage(w http.ResponseWriter, r *http.Request) {
	if aghhttp.WriteTextPlainDeprecated(w, r) {
		return
	}

	langReq := &languageJSON{}
	err := json.NewDecoder(r.Body).Decode(langReq)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "reading req: %s", err)

		return
	}

	lang := langReq.Language
	if !allowedLanguages.Has(lang) {
		aghhttp.Error(r, w, http.StatusBadRequest, "unknown language: %q", lang)

		return
	}

	func() {
		config.Lock()
		defer config.Unlock()

		config.Language = lang
		log.Printf("home: language is set to %s", lang)
	}()

	onConfigModified()
	aghhttp.OK(w)
}
