package home

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

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

// -------------------
// first run / install
// -------------------
func detectFirstRun() bool {
	configfile := Context.configFilename
	if !filepath.IsAbs(configfile) {
		configfile = filepath.Join(Context.workDir, Context.configFilename)
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
			http.Redirect(w, r, "/install.html", http.StatusSeeOther) // should not be cacheable
			return
		}
		// enforce https?
		if config.TLS.ForceHTTPS && r.TLS == nil && config.TLS.Enabled && config.TLS.PortHTTPS != 0 && Context.httpsServer.server != nil {
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

// Connect to a remote server resolving hostname using our own DNS server
func customDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	log.Tracef("network:%v  addr:%v", network, addr)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: time.Minute * 5,
	}

	if net.ParseIP(host) != nil || config.DNS.Port == 0 {
		con, err := dialer.DialContext(ctx, network, addr)
		return con, err
	}

	addrs, e := Context.dnsServer.Resolve(host)
	log.Debug("dnsServer.Resolve: %s: %v", host, addrs)
	if e != nil {
		return nil, e
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("couldn't lookup host: %s", host)
	}

	var dialErrs []error
	for _, a := range addrs {
		addr = net.JoinHostPort(a.String(), port)
		con, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			dialErrs = append(dialErrs, err)
			continue
		}
		return con, err
	}
	return nil, errorx.DecorateMany(fmt.Sprintf("couldn't dial to %s", addr), dialErrs...)
}

// ---------------------
// general helpers
// ---------------------

// fileExists returns TRUE if file exists
func fileExists(fn string) bool {
	_, err := os.Stat(fn)
	if err != nil {
		return false
	}
	return true
}

// runCommand runs shell command
func runCommand(command string, arguments ...string) (int, string, error) {
	cmd := exec.Command(command, arguments...)
	out, err := cmd.Output()
	if err != nil {
		return 1, "", fmt.Errorf("exec.Command(%s) failed: %s", command, err)
	}

	return cmd.ProcessState.ExitCode(), string(out), nil
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

// SplitNext - split string by a byte and return the first chunk
// Whitespace is trimmed
func SplitNext(str *string, splitBy byte) string {
	i := strings.IndexByte(*str, splitBy)
	s := ""
	if i != -1 {
		s = (*str)[0:i]
		*str = (*str)[i+1:]
	} else {
		s = *str
		*str = ""
	}
	return strings.TrimSpace(s)
}
