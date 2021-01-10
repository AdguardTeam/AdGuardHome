package home

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/AdguardTeam/golibs/log"
)

// --------------------
// internationalization
// --------------------
var allowedLanguages = map[string]bool{
	"be":    true,
	"bg":    true,
	"cs":    true,
	"da":    true,
	"de":    true,
	"en":    true,
	"es":    true,
	"fa":    true,
	"fr":    true,
	"hr":    true,
	"hu":    true,
	"id":    true,
	"it":    true,
	"ja":    true,
	"ko":    true,
	"nl":    true,
	"no":    true,
	"pl":    true,
	"pt-br": true,
	"pt-pt": true,
	"ro":    true,
	"ru":    true,
	"si-lk": true,
	"sk":    true,
	"sl":    true,
	"sr-cs": true,
	"sv":    true,
	"th":    true,
	"tr":    true,
	"vi":    true,
	"zh-cn": true,
	"zh-hk": true,
	"zh-tw": true,
}

func isLanguageAllowed(language string) bool {
	l := strings.ToLower(language)
	return allowedLanguages[l]
}

func handleI18nCurrentLanguage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	log.Printf("config.Language is %s", config.Language)
	_, err := fmt.Fprintf(w, "%s\n", config.Language)
	if err != nil {
		msg := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}

func handleI18nChangeLanguage(w http.ResponseWriter, r *http.Request) {
	// This use of ReadAll is safe, because request's body is now limited.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body: %s", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	language := strings.TrimSpace(string(body))
	if language == "" {
		msg := "empty language specified"
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if !isLanguageAllowed(language) {
		msg := fmt.Sprintf("unknown language specified: %s", language)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	config.Language = language
	onConfigModified()
	returnOK(w)
}
