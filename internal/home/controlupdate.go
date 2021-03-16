package home

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/golibs/log"
)

// temporaryError is the interface for temporary errors from the Go standard
// library.
type temporaryError interface {
	error
	Temporary() (ok bool)
}

// Get the latest available version from the Internet
func handleGetVersionJSON(w http.ResponseWriter, r *http.Request) {
	resp := &versionResponse{}
	if Context.disableUpdate {
		// w.Header().Set("Content-Type", "application/json")
		resp.Disabled = true
		_ = json.NewEncoder(w).Encode(resp)
		// TODO(e.burkov): Add error handling and deal with headers.
		return
	}

	req := &struct {
		Recheck bool `json:"recheck_now"`
	}{}

	var err error
	if r.ContentLength != 0 {
		err = json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			httpError(w, http.StatusBadRequest, "JSON parse: %s", err)
			return
		}
	}

	for i := 0; i != 3; i++ {
		func() {
			Context.controlLock.Lock()
			defer Context.controlLock.Unlock()

			resp.VersionInfo, err = Context.updater.VersionInfo(req.Recheck)
		}()

		if err != nil {
			var terr temporaryError
			if errors.As(err, &terr) && terr.Temporary() {
				// Temporary network error.  This case may happen while
				// we're restarting our DNS server.  Log and sleep for
				// some time.
				//
				// See https://github.com/AdguardTeam/AdGuardHome/issues/934.
				d := time.Duration(i) * time.Second
				log.Info("temp net error: %q; sleeping for %s and retrying", err, d)
				time.Sleep(d)

				continue
			}
		}

		break
	}
	if err != nil {
		vcu := Context.updater.VersionCheckURL()
		// TODO(a.garipov): Figure out the purpose of %T verb.
		httpError(w, http.StatusBadGateway, "Couldn't get version check json from %s: %T %s\n", vcu, err, err)

		return
	}

	resp.confirmAutoUpdate()

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

// handleUpdate performs an update to the latest available version procedure.
func handleUpdate(w http.ResponseWriter, _ *http.Request) {
	if Context.updater.NewVersion() == "" {
		httpError(w, http.StatusBadRequest, "/update request isn't allowed now")
		return
	}

	err := Context.updater.Update()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}

	returnOK(w)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// The background context is used because the underlying functions wrap
	// it with timeout and shut down the server, which handles current
	// request. It also should be done in a separate goroutine due to the
	// same reason.
	go func() {
		finishUpdate(context.Background())
	}()
}

// versionResponse is the response for /control/version.json endpoint.
type versionResponse struct {
	Disabled bool `json:"disabled"`
	updater.VersionInfo
}

// confirmAutoUpdate checks the real possibility of auto update.
func (vr *versionResponse) confirmAutoUpdate() {
	if vr.CanAutoUpdate != nil && *vr.CanAutoUpdate {
		canUpdate := true

		var tlsConf *tlsConfigSettings
		if runtime.GOOS != "windows" {
			tlsConf = &tlsConfigSettings{}
			Context.tls.WriteDiskConfig(tlsConf)
		}

		if tlsConf != nil &&
			((tlsConf.Enabled && (tlsConf.PortHTTPS < 1024 ||
				tlsConf.PortDNSOverTLS < 1024 ||
				tlsConf.PortDNSOverQUIC < 1024)) ||
				config.BindPort < 1024 ||
				config.DNS.Port < 1024) {
			canUpdate, _ = aghos.CanBindPrivilegedPorts()
		}
		vr.CanAutoUpdate = &canUpdate
	}
}

// finishUpdate completes an update procedure.
func finishUpdate(ctx context.Context) {
	log.Info("Stopping all tasks")
	cleanup(ctx)
	cleanupAlways()

	exeName := "AdGuardHome"
	if runtime.GOOS == "windows" {
		exeName = "AdGuardHome.exe"
	}
	curBinName := filepath.Join(Context.workDir, exeName)

	if runtime.GOOS == "windows" {
		if Context.runningAsService {
			// Note:
			// we can't restart the service via "kardianos/service" package - it kills the process first
			// we can't start a new instance - Windows doesn't allow it
			cmd := exec.Command("cmd", "/c", "net stop AdGuardHome & net start AdGuardHome")
			err := cmd.Start()
			if err != nil {
				log.Fatalf("exec.Command() failed: %s", err)
			}
			os.Exit(0)
		}

		cmd := exec.Command(curBinName, os.Args[1:]...)
		log.Info("Restarting: %v", cmd.Args)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			log.Fatalf("exec.Command() failed: %s", err)
		}
		os.Exit(0)
	} else {
		log.Info("Restarting: %v", os.Args)
		err := syscall.Exec(curBinName, os.Args, os.Environ())
		if err != nil {
			log.Fatalf("syscall.Exec() failed: %s", err)
		}
		// Unreachable code
	}
}
