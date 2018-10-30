package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// ----------------------------------
// helper functions for working with files
// ----------------------------------

// Writes data first to a temporary file and then renames it to what's specified in path
func writeFileSafe(path string, data []byte) error {

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
	err = os.Rename(tmpPath, path)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------
// helper functions for HTTP handlers
// ----------------------------------
func ensure(method string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "This request must be "+method, 405)
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
	if config.AuthName == "" || config.AuthPass == "" {
		a.handler.ServeHTTP(w, r)
		return
	}
	user, pass, ok := r.BasicAuth()
	if !ok || user != config.AuthName || pass != config.AuthPass {
		w.Header().Set("WWW-Authenticate", `Basic realm="dnsfilter"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorised.\n"))
		return
	}
	a.handler.ServeHTTP(w, r)
}

func optionalAuthHandler(handler http.Handler) http.Handler {
	return &authHandler{handler}
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

// ---------------------
// debug logging helpers
// ---------------------
func trace(format string, args ...interface{}) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%s(): ", path.Base(f.Name())))
	text := fmt.Sprintf(format, args...)
	buf.WriteString(text)
	if len(text) == 0 || text[len(text)-1] != '\n' {
		buf.WriteRune('\n')
	}
	fmt.Fprint(os.Stderr, buf.String())
}
