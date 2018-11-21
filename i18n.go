package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// --------------------
// internationalization
// --------------------
var allowedLanguages = map[string]bool{
	"en": true,
	"ru": true,
	"vi": true,
}

func isLanguageAllowed(language string) bool {
	l := strings.ToLower(language)
	if allowedLanguages[l] {
		return true
	}
	return false
}

func handleI18nCurrentLanguage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := fmt.Fprintf(w, config.Language)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
}

func handleI18nChangeLanguage(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to read request body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	language := strings.TrimSpace(string(body))
	if language == "" {
		errorText := fmt.Sprintf("empty language specified")
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}
	if !isLanguageAllowed(language) {
		errorText := fmt.Sprintf("unknown language specified: %s", language)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	httpUpdateConfigReloadDNSReturnOK(w, r)
}
