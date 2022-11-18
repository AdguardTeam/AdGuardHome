package home

import (
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/NYTimes/gziphandler"
)

// appendDNSAddrs is a convenient helper for appending a formatted form of DNS
// addresses to a slice of strings.
func appendDNSAddrs(dst []string, addrs ...netip.Addr) (res []string) {
	for _, addr := range addrs {
		var hostport string
		if config.DNS.Port != defaultPortDNS {
			hostport = netip.AddrPortFrom(addr, uint16(config.DNS.Port)).String()
		} else {
			hostport = addr.String()
		}

		dst = append(dst, hostport)
	}

	return dst
}

// appendDNSAddrsWithIfaces formats and appends all DNS addresses from src to
// dst.  It also adds the IP addresses of all network interfaces if src contains
// an unspecified IP address.
func appendDNSAddrsWithIfaces(dst []string, src []netip.Addr) (res []string, err error) {
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
		addrs = appendDNSAddrs(addrs, netutil.IPv4Localhost())
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
	Version             string   `json:"version"`
	Language            string   `json:"language"`
	DNSAddrs            []string `json:"dns_addresses"`
	DNSPort             int      `json:"dns_port"`
	HTTPPort            int      `json:"http_port"`
	IsProtectionEnabled bool     `json:"protection_enabled"`
	// TODO(e.burkov): Inspect if front-end doesn't requires this field as
	// openapi.yaml declares.
	IsDHCPAvailable bool `json:"dhcp_available"`
	IsRunning       bool `json:"running"`
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	dnsAddrs, err := collectDNSAddresses()
	if err != nil {
		// Don't add a lot of formatting, since the error is already
		// wrapped by collectDNSAddresses.
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	var resp statusResponse
	func() {
		config.RLock()
		defer config.RUnlock()

		resp = statusResponse{
			Version:   version.Version(),
			DNSAddrs:  dnsAddrs,
			DNSPort:   config.DNS.Port,
			HTTPPort:  config.BindPort,
			Language:  config.Language,
			IsRunning: isRunning(),
		}
	}()

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

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

type profileJSON struct {
	Name string `json:"name"`
}

func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	u := Context.auth.getCurrentUser(r)
	resp := &profileJSON{
		Name: u.Name,
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
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

	// No auth is necessary for DoH/DoT configurations
	Context.mux.HandleFunc("/apple/doh.mobileconfig", postInstall(handleMobileConfigDoH))
	Context.mux.HandleFunc("/apple/dot.mobileconfig", postInstall(handleMobileConfigDoT))
	RegisterAuthHandlers()
}

func httpRegister(method, url string, handler http.HandlerFunc) {
	if method == "" {
		// "/dns-query" handler doesn't need auth, gzip and isn't restricted by 1 HTTP method
		Context.mux.HandleFunc(url, postInstall(handler))
		return
	}

	Context.mux.Handle(url, postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(ensureHandler(method, handler)))))
}

// ensure returns a wrapped handler that makes sure that the request has the
// correct method as well as additional method and header checks.
func ensure(
	method string,
	handler func(http.ResponseWriter, *http.Request),
) (wrapped func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m, u := r.Method, r.URL
		log.Debug("started %s %s %s", m, r.Host, u)
		defer func() { log.Debug("finished %s %s %s in %s", m, r.Host, u, time.Since(start)) }()

		if m != method {
			aghhttp.Error(r, w, http.StatusMethodNotAllowed, "only method %s is allowed", method)

			return
		}

		if modifiesData(m) {
			if !ensureContentType(w, r) {
				return
			}

			Context.controlLock.Lock()
			defer Context.controlLock.Unlock()
		}

		handler(w, r)
	}
}

// modifiesData returns true if m is an HTTP method that can modify data.
func modifiesData(m string) (ok bool) {
	return m == http.MethodPost || m == http.MethodPut || m == http.MethodDelete
}

// ensureContentType makes sure that the content type of a data-modifying
// request is set correctly.  If it is not, ensureContentType writes a response
// to w, and ok is false.
func ensureContentType(w http.ResponseWriter, r *http.Request) (ok bool) {
	const statusUnsup = http.StatusUnsupportedMediaType

	cType := r.Header.Get(aghhttp.HdrNameContentType)
	if r.ContentLength == 0 {
		if cType == "" {
			return true
		}

		// Assume that browsers always send a content type when submitting HTML
		// forms and require no content type for requests with no body to make
		// sure that the request comes from JavaScript.
		aghhttp.Error(r, w, statusUnsup, "empty body with content-type %q not allowed", cType)

		return false

	}

	const wantCType = aghhttp.HdrValApplicationJSON
	if cType == wantCType {
		return true
	}

	aghhttp.Error(r, w, statusUnsup, "only content-type %s is allowed", wantCType)

	return false
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

// handleHTTPSRedirect redirects the request to HTTPS, if needed.  If ok is
// true, the middleware must continue handling the request.
func handleHTTPSRedirect(w http.ResponseWriter, r *http.Request) (ok bool) {
	web := Context.web
	if web.httpsServer.server == nil {
		return true
	}

	host, err := netutil.SplitHost(r.Host)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "bad host: %s", err)

		return false
	}

	var serveHTTP3 bool
	var portHTTPS int
	func() {
		config.RLock()
		defer config.RUnlock()

		serveHTTP3, portHTTPS = config.DNS.ServeHTTP3, config.TLS.PortHTTPS
	}()

	respHdr := w.Header()

	// Let the browser know that server supports HTTP/3.
	//
	// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc.
	//
	// TODO(a.garipov): Consider adding a configurable max-age.  Currently, the
	// default is 24 hours.
	if serveHTTP3 {
		altSvc := fmt.Sprintf(`h3=":%d"`, portHTTPS)
		respHdr.Set(aghhttp.HdrNameAltSvc, altSvc)
	}

	if r.TLS == nil && web.forceHTTPS {
		hostPort := host
		if port := web.conf.PortHTTPS; port != defaultPortHTTPS {
			hostPort = netutil.JoinHostPort(host, port)
		}

		httpsURL := &url.URL{
			Scheme:   aghhttp.SchemeHTTPS,
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
		Scheme: aghhttp.SchemeHTTP,
		Host:   r.Host,
	}

	respHdr.Set(aghhttp.HdrNameAccessControlAllowOrigin, originURL.String())
	respHdr.Set(aghhttp.HdrNameVary, aghhttp.HdrNameOrigin)

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
