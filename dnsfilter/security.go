// Parental Control, Safe Browsing, Safe Search

package dnsfilter

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/log"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info("DNSFilter: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

func (d *Dnsfilter) handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeBrowsingEnabled = true
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeBrowsingEnabled = false
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": d.Config.SafeBrowsingEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

func parseParametersFromBody(r io.Reader) (map[string]string, error) {
	parameters := map[string]string{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			// skip empty lines
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return parameters, errors.New("Got invalid request body")
		}
		parameters[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return parameters, nil
}

func (d *Dnsfilter) handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "failed to parse parameters from body: %s", err)
		return
	}

	sensitivity, ok := parameters["sensitivity"]
	if !ok {
		http.Error(w, "Sensitivity parameter was not specified", 400)
		return
	}

	switch sensitivity {
	case "3":
		break
	case "EARLY_CHILDHOOD":
		sensitivity = "3"
	case "10":
		break
	case "YOUNG":
		sensitivity = "10"
	case "13":
		break
	case "TEEN":
		sensitivity = "13"
	case "17":
		break
	case "MATURE":
		sensitivity = "17"
	default:
		http.Error(w, "Sensitivity must be set to valid value", 400)
		return
	}
	i, err := strconv.Atoi(sensitivity)
	if err != nil {
		http.Error(w, "Sensitivity must be set to valid value", 400)
		return
	}
	d.Config.ParentalSensitivity = i
	d.Config.ParentalEnabled = true
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleParentalDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.ParentalEnabled = false
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": d.Config.ParentalEnabled,
	}
	if d.Config.ParentalEnabled {
		data["sensitivity"] = d.Config.ParentalSensitivity
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

func (d *Dnsfilter) handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeSearchEnabled = true
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeSearchEnabled = false
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": d.Config.SafeSearchEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

func (d *Dnsfilter) registerSecurityHandlers() {
	d.Config.HTTPRegister("POST", "/control/safebrowsing/enable", d.handleSafeBrowsingEnable)
	d.Config.HTTPRegister("POST", "/control/safebrowsing/disable", d.handleSafeBrowsingDisable)
	d.Config.HTTPRegister("GET", "/control/safebrowsing/status", d.handleSafeBrowsingStatus)
	d.Config.HTTPRegister("POST", "/control/parental/enable", d.handleParentalEnable)
	d.Config.HTTPRegister("POST", "/control/parental/disable", d.handleParentalDisable)
	d.Config.HTTPRegister("GET", "/control/parental/status", d.handleParentalStatus)
	d.Config.HTTPRegister("POST", "/control/safesearch/enable", d.handleSafeSearchEnable)
	d.Config.HTTPRegister("POST", "/control/safesearch/disable", d.handleSafeSearchDisable)
	d.Config.HTTPRegister("GET", "/control/safesearch/status", d.handleSafeSearchStatus)
}
