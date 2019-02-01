package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hmage/golibs/log"
)

// ----------------------------------
// helper functions for working with files
// ----------------------------------

// Writes data first to a temporary file and then renames it to what's specified in path
func safeWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	err = ioutil.WriteFile(tmpPath, data, 0644)
	if err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// ----------------------------------
// helper functions for HTTP handlers
// ----------------------------------
func ensure(method string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "This request must be "+method, http.StatusMethodNotAllowed)
			return
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

func ensurePUT(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("PUT", handler)
}

func ensureDELETE(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("DELETE", handler)
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
		configfile = filepath.Join(config.ourBinaryDir, config.ourConfigFilename)
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

// preInstallStruct wraps preInstall into a struct that can be returned as an interface where neccessary
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
func postInstall(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.firstRun && !strings.HasPrefix(r.URL.Path, "/install.") {
			http.Redirect(w, r, "/install.html", http.StatusSeeOther) // should not be cacheable
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
}

// getValidNetInterfaces() returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is either a loopback, ppp interface, or the one that doesn't allow broadcasts
func getValidNetInterfaces() ([]netInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get list of interfaces: %s", err)
	}

	netIfaces := []netInterface{}

	for i := range ifaces {
		if ifaces[i].Flags&net.FlagLoopback != 0 {
			// it's a loopback, skip it
			continue
		}
		if ifaces[i].Flags&net.FlagBroadcast == 0 {
			// this interface doesn't support broadcast, skip it
			continue
		}
		if ifaces[i].Flags&net.FlagPointToPoint != 0 {
			// this interface is ppp, don't do dhcp over it
			continue
		}

		iface := netInterface{
			Name:         ifaces[i].Name,
			MTU:          ifaces[i].MTU,
			HardwareAddr: ifaces[i].HardwareAddr.String(),
		}

		addrs, err := ifaces[i].Addrs()
		if err != nil {
			return nil, fmt.Errorf("Failed to get addresses for interface %v: %s", ifaces[i].Name, err)
		}
		for _, addr := range addrs {
			iface.Addresses = append(iface.Addresses, addr.String())
		}
		if len(iface.Addresses) == 0 {
			// this interface has no addresses, skip it
			continue
		}
		netIfaces = append(netIfaces, iface)
	}

	return netIfaces, nil
}

func findIPv4IfaceAddr(ifaces []netInterface) string {
	for _, iface := range ifaces {
		for _, addr := range iface.Addresses {
			ip, _, err := net.ParseCIDR(addr)
			if err != nil {
				log.Printf("SHOULD NOT HAPPEN: got iface.Addresses element that's not a parseable CIDR: %s", addr)
				continue
			}
			if ip.To4() == nil {
				log.Tracef("Ignoring IP that isn't IPv4: %s", ip)
				continue
			}
			return ip.To4().String()
		}
	}
	return ""
}

// checkPortAvailable is not a cheap test to see if the port is bindable, because it's actually doing the bind momentarily
func checkPortAvailable(host string, port int) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func checkPacketPortAvailable(host string, port int) bool {
	ln, err := net.ListenPacket("udp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	ln.Close()
	return true
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
