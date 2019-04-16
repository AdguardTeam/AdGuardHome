package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joomcode/errorx"
)

// ----------------------------------
// helper functions for HTTP handlers
// ----------------------------------
func ensure(method string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "This request must be "+method, http.StatusMethodNotAllowed)
			return
		}

		if method == "POST" || method == "PUT" || method == "DELETE" {
			controlLock.Lock()
			defer controlLock.Unlock()
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

func ensureGETHandler(handler func(http.ResponseWriter, *http.Request)) http.Handler {
	h := httpHandler{}
	h.handler = ensure("GET", handler)
	return &h
}

func optionalAuth(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.AuthName == "" || config.AuthPass == "" {
			handler(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != config.AuthName || pass != config.AuthPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="dnsfilter"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorised.\n"))
			return
		}
		handler(w, r)
	}
}

type authHandler struct {
	handler http.Handler
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	optionalAuth(a.handler.ServeHTTP)(w, r)
}

func optionalAuthHandler(handler http.Handler) http.Handler {
	return &authHandler{handler}
}

// -------------------
// first run / install
// -------------------
func detectFirstRun() bool {
	configfile := config.ourConfigFilename
	if !filepath.IsAbs(configfile) {
		configfile = filepath.Join(config.ourWorkingDir, config.ourConfigFilename)
	}
	_, err := os.Stat(configfile)
	if !os.IsNotExist(err) {
		// do nothing, file exists
		return false
	}
	return true
}

// preInstall lets the handler run only if firstRun is true, no redirects
func preInstall(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !config.firstRun {
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
		if config.firstRun && !strings.HasPrefix(r.URL.Path, "/install.") {
			http.Redirect(w, r, "/install.html", http.StatusSeeOther) // should not be cacheable
			return
		}
		// enforce https?
		if config.TLS.ForceHTTPS && r.TLS == nil && config.TLS.Enabled && config.TLS.PortHTTPS != 0 && httpsServer.server != nil {
			// yes, and we want host from host:port
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				// no port in host
				host = r.Host
			}
			// construct new URL to redirect to
			newURL := url.URL{
				Scheme:   "https",
				Host:     net.JoinHostPort(host, strconv.Itoa(config.TLS.PortHTTPS)),
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

// -------------------------------------------------
// helper functions for parsing parameters from body
// -------------------------------------------------
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

// ------------------
// network interfaces
// ------------------
type netInterface struct {
	Name         string   `json:"name"`
	MTU          int      `json:"mtu"`
	HardwareAddr string   `json:"hardware_address"`
	Addresses    []string `json:"ip_addresses"`
	Flags        string   `json:"flags"`
}

// getValidNetInterfaces returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is a ppp interface or the one that doesn't allow broadcasts
func getValidNetInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get list of interfaces: %s", err)
	}

	netIfaces := []net.Interface{}

	for i := range ifaces {
		if ifaces[i].Flags&net.FlagPointToPoint != 0 {
			// this interface is ppp, we're not interested in this one
			continue
		}

		iface := ifaces[i]
		netIfaces = append(netIfaces, iface)
	}

	return netIfaces, nil
}

// getValidNetInterfacesMap returns interfaces that are eligible for DNS and WEB only
// we do not return link-local addresses here
func getValidNetInterfacesForWeb() ([]netInterface, error) {
	ifaces, err := getValidNetInterfaces()
	if err != nil {
		return nil, errorx.Decorate(err, "Couldn't get interfaces")
	}
	if len(ifaces) == 0 {
		return nil, errors.New("couldn't find any legible interface")
	}

	var netInterfaces []netInterface

	for _, iface := range ifaces {
		addrs, e := iface.Addrs()
		if e != nil {
			return nil, errorx.Decorate(e, "Failed to get addresses for interface %s", iface.Name)
		}

		netIface := netInterface{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr.String(),
		}

		if iface.Flags != 0 {
			netIface.Flags = iface.Flags.String()
		}

		// we don't want link-local addresses in json, so skip them
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				// not an IPNet, should not happen
				return nil, fmt.Errorf("SHOULD NOT HAPPEN: got iface.Addrs() element %s that is not net.IPNet, it is %T", addr, addr)
			}
			// ignore link-local
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			netIface.Addresses = append(netIface.Addresses, ipnet.IP.String())
		}
		if len(netIface.Addresses) != 0 {
			netInterfaces = append(netInterfaces, netIface)
		}
	}

	return netInterfaces, nil
}

// checkPortAvailable is not a cheap test to see if the port is bindable, because it's actually doing the bind momentarily
func checkPortAvailable(host string, port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

func checkPacketPortAvailable(host string, port int) error {
	ln, err := net.ListenPacket("udp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return err
	}
	ln.Close()
	return err
}

// ---------------------
// debug logging helpers
// ---------------------
func _Func() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}
