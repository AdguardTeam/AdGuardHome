package home

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/golibs/log"
	"github.com/NYTimes/gziphandler"
)

// ----------------
// helper functions
// ----------------

func returnOK(w http.ResponseWriter) {
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func httpError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info(text)
	http.Error(w, text, code)
}

// ---------------
// dns run control
// ---------------
func addDNSAddress(dnsAddresses *[]string, addr string) {
	if config.DNS.Port != 53 {
		addr = fmt.Sprintf("%s:%d", addr, config.DNS.Port)
	}
	*dnsAddresses = append(*dnsAddresses, addr)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	c := dnsforward.FilteringConfig{}
	if Context.dnsServer != nil {
		Context.dnsServer.WriteDiskConfig(&c)
	}
	data := map[string]interface{}{
		"dns_addresses": getDNSAddresses(),
		"http_port":     config.BindPort,
		"dns_port":      config.DNS.Port,
		"running":       isRunning(),
		"version":       versionString,
		"language":      config.Language,

		"protection_enabled": c.ProtectionEnabled,
	}

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

type profileJSON struct {
	Name string `json:"name"`
}

func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	pj := profileJSON{}
	u := Context.auth.GetCurrentUser(r)
	pj.Name = u.Name

	data, err := json.Marshal(pj)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json.Marshal: %s", err)
		return
	}
	_, _ = w.Write(data)
}

// ------------------------
// registration of handlers
// ------------------------
func registerControlHandlers() {
	httpRegister(http.MethodGet, "/control/status", handleStatus)
	httpRegister(http.MethodPost, "/control/i18n/change_language", handleI18nChangeLanguage)
	httpRegister(http.MethodGet, "/control/i18n/current_language", handleI18nCurrentLanguage)
	http.HandleFunc("/control/version.json", postInstall(optionalAuth(handleGetVersionJSON)))
	httpRegister(http.MethodPost, "/control/update", handleUpdate)

	httpRegister("GET", "/control/profile", handleGetProfile)
	RegisterAuthHandlers()
}

func httpRegister(method string, url string, handler func(http.ResponseWriter, *http.Request)) {
	if len(method) == 0 {
		// "/dns-query" handler doesn't need auth, gzip and isn't restricted by 1 HTTP method
		http.HandleFunc(url, postInstall(handler))
		return
	}

	http.Handle(url, postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(ensureHandler(method, handler)))))
}

// ----------------------------------
// helper functions for HTTP handlers
// ----------------------------------
func ensure(method string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug("%s %v", r.Method, r.URL)

		if r.Method != method {
			http.Error(w, "This request must be "+method, http.StatusMethodNotAllowed)
			return
		}

		if method == "POST" || method == "PUT" || method == "DELETE" {
			Context.controlLock.Lock()
			defer Context.controlLock.Unlock()
		}

		handler(w, r)
	}
}

func ensurePOST(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("POST", handler)
}

func ensureGET(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("GET", handler)
}

// Bridge between http.Handler object and Go function
type httpHandler struct {
	handler func(http.ResponseWriter, *http.Request)
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler(w, r)
}

func ensureHandler(method string, handler func(http.ResponseWriter, *http.Request)) http.Handler {
	h := httpHandler{}
	h.handler = ensure(method, handler)
	return &h
}

// preInstall lets the handler run only if firstRun is true, no redirects
func preInstall(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !Context.firstRun {
			// if it's not first run, don't let users access it (for example /install.html when configuration is done)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		handler(w, r)
	}
}

// preInstallStruct wraps preInstall into a struct that can be returned as an interface where necessary
type preInstallHandlerStruct struct {
	handler http.Handler
}

func (p *preInstallHandlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	preInstall(p.handler.ServeHTTP)(w, r)
}

// preInstallHandler returns http.Handler interface for preInstall wrapper
func preInstallHandler(handler http.Handler) http.Handler {
	return &preInstallHandlerStruct{handler}
}

// postInstall lets the handler run only if firstRun is false, and redirects to /install.html otherwise
// it also enforces HTTPS if it is enabled and configured
func postInstall(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if Context.firstRun &&
			!strings.HasPrefix(r.URL.Path, "/install.") &&
			r.URL.Path != "/favicon.png" {
			http.Redirect(w, r, "/install.html", http.StatusFound)
			return
		}

		// enforce https?
		if r.TLS == nil && Context.web.forceHTTPS && Context.web.httpsServer.server != nil {
			// yes, and we want host from host:port
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				// no port in host
				host = r.Host
			}
			// construct new URL to redirect to
			newURL := url.URL{
				Scheme:   "https",
				Host:     net.JoinHostPort(host, strconv.Itoa(Context.web.portHTTPS)),
				Path:     r.URL.Path,
				RawQuery: r.URL.RawQuery,
			}
			http.Redirect(w, r, newURL.String(), http.StatusTemporaryRedirect)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		handler(w, r)
	}
}

type postInstallHandlerStruct struct {
	handler http.Handler
}

func (p *postInstallHandlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	postInstall(p.handler.ServeHTTP)(w, r)
}

func postInstallHandler(handler http.Handler) http.Handler {
	return &postInstallHandlerStruct{handler}
}
