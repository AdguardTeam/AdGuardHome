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
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/NYTimes/gziphandler"
)

// appendDNSAddrs is a convenient helper for appending a formatted form of DNS
// addresses to a slice of strings.
func appendDNSAddrs(dst []string, addrs ...netip.Addr) (res []string) {
	for _, addr := range addrs {
		hostport := addr.String()
		if p := config.DNS.Port; p != defaultPortDNS {
			hostport = netutil.JoinHostPort(hostport, p)
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
// tlsMgr must not be nil.
func collectDNSAddresses(tlsMgr *tlsManager) (addrs []string, err error) {
	if hosts := config.DNS.BindHosts; len(hosts) == 0 {
		addrs = appendDNSAddrs(addrs, netutil.IPv4Localhost())
	} else {
		addrs, err = appendDNSAddrsWithIfaces(addrs, hosts)
		if err != nil {
			return nil, fmt.Errorf("collecting dns addresses: %w", err)
		}
	}

	de := getDNSEncryption(tlsMgr)
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
	Version  string   `json:"version"`
	Language string   `json:"language"`
	DNSAddrs []string `json:"dns_addresses"`
	DNSPort  uint16   `json:"dns_port"`
	HTTPPort uint16   `json:"http_port"`

	// ProtectionDisabledDuration is the duration of the protection pause in
	// milliseconds.
	ProtectionDisabledDuration int64 `json:"protection_disabled_duration"`

	ProtectionEnabled bool `json:"protection_enabled"`
	// TODO(e.burkov): Inspect if front-end doesn't requires this field as
	// openapi.yaml declares.
	IsDHCPAvailable bool `json:"dhcp_available"`
	IsRunning       bool `json:"running"`
}

func (web *webAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dnsAddrs, err := collectDNSAddresses(web.tlsManager)
	if err != nil {
		// Don't add a lot of formatting, since the error is already
		// wrapped by collectDNSAddresses.
		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	var (
		fltConf           *dnsforward.Config
		protDisabledUntil *time.Time
		protEnabled       bool
	)
	if globalContext.dnsServer != nil {
		fltConf = &dnsforward.Config{}
		globalContext.dnsServer.WriteDiskConfig(fltConf)
		protEnabled, protDisabledUntil = globalContext.dnsServer.UpdatedProtectionStatus(ctx)
	}

	var resp statusResponse
	func() {
		config.RLock()
		defer config.RUnlock()

		var protectionDisabledDuration int64
		if protDisabledUntil != nil {
			// Make sure that we don't send negative numbers to the frontend,
			// since enough time might have passed to make the difference less
			// than zero.
			protectionDisabledDuration = max(0, time.Until(*protDisabledUntil).Milliseconds())
		}

		resp = statusResponse{
			Version:                    version.Version(),
			Language:                   config.Language,
			DNSAddrs:                   dnsAddrs,
			DNSPort:                    config.DNS.Port,
			HTTPPort:                   config.HTTPConfig.Address.Port(),
			ProtectionDisabledDuration: protectionDisabledDuration,
			ProtectionEnabled:          protEnabled,
			IsRunning:                  isRunning(),
		}
	}()

	// IsDHCPAvailable field is now false by default for Windows.
	if runtime.GOOS != "windows" {
		resp.IsDHCPAvailable = globalContext.dhcpServer != nil
	}

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// registerControlHandlers sets up HTTP handlers for various control endpoints.
func (web *webAPI) registerControlHandlers() {
	mux := web.conf.mux

	mux.Handle(
		"/control/version.json",
		web.postInstallHandler(http.HandlerFunc(web.handleVersionJSON)),
	)
	httpRegister(http.MethodPost, "/control/update", web.handleUpdate)

	httpRegister(http.MethodGet, "/control/status", web.handleStatus)
	httpRegister(http.MethodPost, "/control/i18n/change_language", web.handleI18nChangeLanguage)
	httpRegister(http.MethodGet, "/control/i18n/current_language", handleI18nCurrentLanguage)
	httpRegister(http.MethodGet, "/control/profile", web.handleGetProfile)
	httpRegister(http.MethodPut, "/control/profile/update", web.handlePutProfile)

	// No authentication is required for DoH/DoT configuration endpoints.
	mux.Handle(
		"/apple/doh.mobileconfig",
		web.postInstallHandler(http.HandlerFunc(handleMobileConfigDoH)),
	)
	mux.Handle(
		"/apple/dot.mobileconfig",
		web.postInstallHandler(http.HandlerFunc(handleMobileConfigDoT)),
	)

	web.registerAuthHandlers()
}

// httpRegister registers an HTTP handler.
//
// TODO(s.chzhen):  Do not use [globalContext.mux].
func httpRegister(method, url string, handler http.HandlerFunc) {
	if method == "" {
		// "/dns-query" handler doesn't need auth, gzip and isn't restricted by 1 HTTP method
		globalContext.mux.Handle(url, postInstallHandler(handler))
		return
	}

	globalContext.mux.Handle(
		url,
		postInstallHandler(gziphandler.GzipHandler(ensure(method, handler))),
	)
}

// ensure returns a wrapped handler that makes sure that the request has the
// correct method as well as additional method and header checks.
func ensure(
	method string,
	handler func(http.ResponseWriter, *http.Request),
) (wrapped http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request) {
		m := r.Method
		if m != method {
			aghhttp.Error(r, w, http.StatusMethodNotAllowed, "only method %s is allowed", method)

			return
		}

		if modifiesData(m) {
			if !ensureContentType(w, r) {
				return
			}

			globalContext.controlLock.Lock()
			defer globalContext.controlLock.Unlock()
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

	cType := r.Header.Get(httphdr.ContentType)
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

// preInstallHandler lets the handler run only if firstRun is true; it does not
// perform redirects.
func (web *webAPI) preInstallHandler(handler http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !web.conf.firstRun {
			// If it's not first run, do not allow access to install-only routes
			// (for example, /install.html once configuration is complete).
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)

			return
		}

		handler.ServeHTTP(w, r)
	})
}

// handleHTTPSRedirect redirects the request to HTTPS, if needed, and adds some
// HTTPS-related headers.  If proceed is true, the middleware must continue
// handling the request.
func handleHTTPSRedirect(w http.ResponseWriter, r *http.Request) (proceed bool) {
	web := globalContext.web
	if web.httpsServer.server == nil {
		return true
	}

	host, err := netutil.SplitHost(r.Host)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "bad host: %s", err)

		return false
	}

	var (
		forceHTTPS bool
		serveHTTP3 bool
		portHTTPS  uint16
	)
	func() {
		config.RLock()
		defer config.RUnlock()

		serveHTTP3, portHTTPS = config.DNS.ServeHTTP3, config.TLS.PortHTTPS
		forceHTTPS = config.TLS.ForceHTTPS && config.TLS.Enabled && config.TLS.PortHTTPS != 0
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
		respHdr.Set(httphdr.AltSvc, altSvc)
	}

	if forceHTTPS {
		if r.TLS == nil {
			u := httpsURL(r.URL, host, portHTTPS)
			http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)

			return false
		}

		// TODO(a.garipov): Consider adding a configurable max-age.  Currently,
		// the default is 365 days.
		respHdr.Set(httphdr.StrictTransportSecurity, aghhttp.HdrValStrictTransportSecurity)
	}

	// Allow the frontend from the HTTP origin to send requests to the HTTPS
	// server.  This can happen when the user has just set up HTTPS with
	// redirects.  Prevent cache-related errors by setting the Vary header.
	//
	// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin.
	originURL := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   r.Host,
	}

	respHdr.Set(httphdr.AccessControlAllowOrigin, originURL.String())
	respHdr.Set(httphdr.Vary, httphdr.Origin)

	return true
}

// httpsURL returns a copy of u for redirection to the HTTPS version, taking the
// hostname and the HTTPS port into account.
func httpsURL(u *url.URL, host string, portHTTPS uint16) (redirectURL *url.URL) {
	hostPort := host
	if portHTTPS != defaultPortHTTPS {
		hostPort = netutil.JoinHostPort(host, portHTTPS)
	}

	return &url.URL{
		Scheme:   urlutil.SchemeHTTPS,
		Host:     hostPort,
		Path:     u.Path,
		RawQuery: u.RawQuery,
	}
}

// postInstallHandler lets the handler to run only if firstRun is false.
// Otherwise, it redirects to /install.html.  It also enforces HTTPS if it is
// enabled and configured and sets appropriate access control headers.
//
// TODO(s.chzhen):  Replace with [web.postInstall] after fixing its usage in
// [httpRegister], which is called by [dhcpd.Create] before [web] is
// initialized.
func postInstallHandler(handler http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if globalContext.web == nil {
			aghhttp.Error(r, w, http.StatusTooEarly, "it is not initialized yet")

			return
		}

		path := r.URL.Path
		if globalContext.web.conf.firstRun &&
			!strings.HasPrefix(path, "/install.") &&
			!strings.HasPrefix(path, "/assets/") {
			http.Redirect(w, r, "install.html", http.StatusFound)

			return
		}

		if handleHTTPSRedirect(w, r) {
			handler.ServeHTTP(w, r)
		}
	})
}

// postInstallHandler lets the handler to run only if firstRun is false.
// Otherwise, it redirects to /install.html.  It also enforces HTTPS if it is
// enabled and configured and sets appropriate access control headers.
func (web *webAPI) postInstallHandler(handler http.Handler) (wrapped http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if web.conf.firstRun &&
			!strings.HasPrefix(path, "/install.") &&
			!strings.HasPrefix(path, "/assets/") {
			http.Redirect(w, r, "install.html", http.StatusFound)

			return
		}

		if handleHTTPSRedirect(w, r) {
			handler.ServeHTTP(w, r)
		}
	})
}
