package home

import (
	"fmt"

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

// validateLang returns a standard error about if lang is an unknown language.
// If allowEmpty is true, the language can also be empty.
func validateLang(lang string, allowEmpty bool) (err error) {
	if allowEmpty && lang == "" {
		return nil
	}

	if !allowedLanguages.Has(lang) {
		return fmt.Errorf("unknown language: %q", lang)
	}

	return nil
}
