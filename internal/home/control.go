package home

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
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

// appendDNSAddrs is a convenient helper for appending a formatted form of DNS
// addresses to a slice of strings.
func appendDNSAddrs(dst []string, addrs ...net.IP) (res []string) {
	for _, addr := range addrs {
		hostport := addr.String()
		if config.DNS.Port != 53 {
			hostport = net.JoinHostPort(hostport, strconv.Itoa(config.DNS.Port))
		}

		dst = append(dst, hostport)
	}

	return dst
}

// appendDNSAddrsWithIfaces formats and appends all DNS addresses from src to
// dst.  It also adds the IP addresses of all network interfaces if src contains
// an unspecified IP addresss.
func appendDNSAddrsWithIfaces(dst []string, src []net.IP) (res []string, err error) {
	ifacesAdded := false
	for _, h := range src {
		if !h.IsUnspecified() {
			dst = appendDNSAddrs(dst, h)

			continue
		} else if ifacesAdded {
			continue
		}

		// Add addresses of all network interfaces for addresses like
		// "0.0.0.0" and "::".
		var ifaces []*aghnet.NetInterface
		ifaces, err = aghnet.GetValidNetInterfacesForWeb()
		if err != nil {
			return nil, fmt.Errorf("cannot get network interfaces: %w", err)
		}

		for _, iface := range ifaces {
			dst = appendDNSAddrs(dst, iface.Addresses...)
		}

		ifacesAdded = true
	}

	return dst, nil
}

// collectDNSAddresses returns the list of DNS addresses the server is listening
// on, including the addresses on all interfaces in cases of unspecified IPs.
func collectDNSAddresses() (addrs []string, err error) {
	if hosts := config.DNS.BindHosts; len(hosts) == 0 {
		addrs = appendDNSAddrs(addrs, net.IP{127, 0, 0, 1})
	} else {
		addrs, err = appendDNSAddrsWithIfaces(addrs, hosts)
		if err != nil {
			return nil, fmt.Errorf("collecting dns addresses: %w", err)
		}
	}

	de := getDNSEncryption()
	if de.https != "" {
		addrs = append(addrs, de.https)
	}

	if de.tls != "" {
		addrs = append(addrs, de.tls)
	}

	if de.quic != "" {
		addrs = append(addrs, de.quic)
	}

	return addrs, nil
}

// statusResponse is a response for /control/status endpoint.
type statusResponse struct {
	DNSAddrs            []string `json:"dns_addresses"`
	DNSPort             int      `json:"dns_port"`
	HTTPPort            int      `json:"http_port"`
	IsProtectionEnabled bool     `json:"protection_enabled"`
	// TODO(e.burkov): Inspect if front-end doesn't requires this field as
	// openapi.yaml declares.
	IsDHCPAvailable bool   `json:"dhcp_available"`
	IsRunning       bool   `json:"running"`
	Version         string `json:"version"`
	Language        string `json:"language"`
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	dnsAddrs, err := collectDNSAddresses()
	if err != nil {
		// Don't add a lot of formatting, since the error is already
		// wrapped by collectDNSAddresses.
		httpError(w, http.StatusInternalServerError, "%s", err)

		return
	}

	resp := statusResponse{
		DNSAddrs:  dnsAddrs,
		DNSPort:   config.DNS.Port,
		HTTPPort:  config.BindPort,
		IsRunning: isRunning(),
		Version:   version.Version(),
		Language:  config.Language,
	}

	var c *dnsforward.FilteringConfig
	if Context.dnsServer != nil {
		c = &dnsforward.FilteringConfig{}
		Context.dnsServer.WriteDiskConfig(c)
		resp.IsProtectionEnabled = c.ProtectionEnabled
	}

	// IsDHCPAvailable field is now false by default for Windows.
	if runtime.GOOS != "windows" {
		resp.IsDHCPAvailable = Context.dhcpServer != nil
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
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
	u := Context.auth.getCurrentUser(r)
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
	Context.mux.HandleFunc("/control/version.json", postInstall(optionalAuth(handleGetVersionJSON)))
	httpRegister(http.MethodPost, "/control/update", handleUpdate)
	httpRegister(http.MethodGet, "/control/profile", handleGetProfile)

	// No auth is necessary for DOH/DOT configurations
	Context.mux.HandleFunc("/apple/doh.mobileconfig", postInstall(handleMobileConfigDOH))
	Context.mux.HandleFunc("/apple/dot.mobileconfig", postInstall(handleMobileConfigDOT))
	RegisterAuthHandlers()
}

func httpRegister(method, url string, handler func(http.ResponseWriter, *http.Request)) {
	if method == "" {
		// "/dns-query" handler doesn't need auth, gzip and isn't restricted by 1 HTTP method
		Context.mux.HandleFunc(url, postInstall(handler))
		return
	}

	Context.mux.Handle(url, postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(ensureHandler(method, handler)))))
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

		if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete {
			Context.controlLock.Lock()
			defer Context.controlLock.Unlock()
		}

		handler(w, r)
	}
}

func ensurePOST(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure(http.MethodPost, handler)
}

func ensureGET(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure(http.MethodGet, handler)
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

const defaultHTTPSPort = 443

// handleHTTPSRedirect redirects the request to HTTPS, if needed.  If ok is
// true, the middleware must continue handling the request.
func handleHTTPSRedirect(w http.ResponseWriter, r *http.Request) (ok bool) {
	web := Context.web
	if web.httpsServer.server == nil {
		return true
	}

	host, err := aghnet.SplitHost(r.Host)
	if err != nil {
		httpError(w, http.StatusBadRequest, "bad host: %s", err)

		return false
	}

	if r.TLS == nil && web.forceHTTPS {
		hostPort := host
		if port := web.conf.PortHTTPS; port != defaultHTTPSPort {
			portStr := strconv.Itoa(port)
			hostPort = net.JoinHostPort(host, portStr)
		}

		httpsURL := &url.URL{
			Scheme:   schemeHTTPS,
			Host:     hostPort,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(w, r, httpsURL.String(), http.StatusTemporaryRedirect)

		return false
	}

	// Allow the frontend from the HTTP origin to send requests to the HTTPS
	// server.  This can happen when the user has just set up HTTPS with
	// redirects.  Prevent cache-related errors by setting the Vary header.
	//
	// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin.
	originURL := &url.URL{
		Scheme: schemeHTTP,
		Host:   r.Host,
	}
	w.Header().Set("Access-Control-Allow-Origin", originURL.String())
	w.Header().Set("Vary", "Origin")

	return true
}

// postInstall lets the handler to run only if firstRun is false.  Otherwise, it
// redirects to /install.html.  It also enforces HTTPS if it is enabled and
// configured and sets appropriate access control headers.
func postInstall(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if Context.firstRun && !strings.HasPrefix(path, "/install.") &&
			!strings.HasPrefix(path, "/assets/") {
			http.Redirect(w, r, "/install.html", http.StatusFound)

			return
		}

		if !handleHTTPSRedirect(w, r) {
			return
		}

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
