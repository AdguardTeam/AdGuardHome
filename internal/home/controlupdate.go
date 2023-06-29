package home

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// temporaryError is the interface for temporary errors from the Go standard
// library.
type temporaryError interface {
	error
	Temporary() (ok bool)
}

// handleVersionJSON is the handler for the POST /control/version.json HTTP API.
//
// TODO(a.garipov): Find out if this API used with a GET method by anyone.
func handleVersionJSON(w http.ResponseWriter, r *http.Request) {
	resp := &versionResponse{}
	if Context.disableUpdate {
		resp.Disabled = true
		_ = aghhttp.WriteJSONResponse(w, r, resp)

		return
	}

	req := &struct {
		Recheck bool `json:"recheck_now"`
	}{}

	var err error
	if r.ContentLength != 0 {
		err = json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "parsing request: %s", err)

			return
		}
	}

	err = requestVersionInfo(resp, req.Recheck)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		aghhttp.Error(r, w, http.StatusBadGateway, "%s", err)

		return
	}

	err = resp.setAllowedToAutoUpdate()
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

// requestVersionInfo sets the VersionInfo field of resp if it can reach the
// update server.
func requestVersionInfo(resp *versionResponse, recheck bool) (err error) {
	for i := 0; i != 3; i++ {
		resp.VersionInfo, err = Context.updater.VersionInfo(recheck)
		if err != nil {
			var terr temporaryError
			if errors.As(err, &terr) && terr.Temporary() {
				// Temporary network error.  This case may happen while we're
				// restarting our DNS server.  Log and sleep for some time.
				//
				// See https://github.com/AdguardTeam/AdGuardHome/issues/934.
				d := time.Duration(i) * time.Second
				log.Info("update: temp net error: %q; sleeping for %s and retrying", err, d)
				time.Sleep(d)

				continue
			}
		}

		break
	}

	if err != nil {
		vcu := Context.updater.VersionCheckURL()

		return fmt.Errorf("getting version info from %s: %w", vcu, err)
	}

	return nil
}

// handleUpdate performs an update to the latest available version procedure.
func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if Context.updater.NewVersion() == "" {
		aghhttp.Error(r, w, http.StatusBadRequest, "/update request isn't allowed now")

		return
	}

	// Retain the current absolute path of the executable, since the updater is
	// likely to change the position current one to the backup directory.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/4735.
	execPath, err := os.Executable()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "getting path: %s", err)

		return
	}

	err = Context.updater.Update(false)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	aghhttp.OK(w)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// The background context is used because the underlying functions wrap it
	// with timeout and shut down the server, which handles current request.  It
	// also should be done in a separate goroutine for the same reason.
	go finishUpdate(context.Background(), execPath)
}

// versionResponse is the response for /control/version.json endpoint.
type versionResponse struct {
	updater.VersionInfo
	Disabled bool `json:"disabled"`
}

// setAllowedToAutoUpdate sets CanAutoUpdate to true if AdGuard Home is actually
// allowed to perform an automatic update by the OS.
func (vr *versionResponse) setAllowedToAutoUpdate() (err error) {
	if vr.CanAutoUpdate != aghalg.NBTrue {
		return nil
	}

	tlsConf := &tlsConfigSettings{}
	Context.tls.WriteDiskConfig(tlsConf)

	canUpdate := true
	if tlsConfUsesPrivilegedPorts(tlsConf) ||
		config.HTTPConfig.Address.Port() < 1024 ||
		config.DNS.Port < 1024 {
		canUpdate, err = aghnet.CanBindPrivilegedPorts()
		if err != nil {
			return fmt.Errorf("checking ability to bind privileged ports: %w", err)
		}
	}

	vr.CanAutoUpdate = aghalg.BoolToNullBool(canUpdate)

	return nil
}

// tlsConfUsesPrivilegedPorts returns true if the provided TLS configuration
// indicates that privileged ports are used.
func tlsConfUsesPrivilegedPorts(c *tlsConfigSettings) (ok bool) {
	return c.Enabled && (c.PortHTTPS < 1024 || c.PortDNSOverTLS < 1024 || c.PortDNSOverQUIC < 1024)
}

// finishUpdate completes an update procedure.
func finishUpdate(ctx context.Context, execPath string) {
	var err error

	log.Info("stopping all tasks")

	cleanup(ctx)
	cleanupAlways()

	if runtime.GOOS == "windows" {
		if Context.runningAsService {
			// NOTE: We can't restart the service via "kardianos/service"
			// package, because it kills the process first we can't start a new
			// instance, because Windows doesn't allow it.
			//
			// TODO(a.garipov): Recheck the claim above.
			cmd := exec.Command("cmd", "/c", "net stop AdGuardHome & net start AdGuardHome")
			err = cmd.Start()
			if err != nil {
				log.Fatalf("restarting: stopping: %s", err)
			}

			os.Exit(0)
		}

		cmd := exec.Command(execPath, os.Args[1:]...)
		log.Info("restarting: %q %q", execPath, os.Args[1:])
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			log.Fatalf("restarting:: %s", err)
		}

		os.Exit(0)
	}

	log.Info("restarting: %q %q", execPath, os.Args[1:])
	err = syscall.Exec(execPath, os.Args, os.Environ())
	if err != nil {
		log.Fatalf("restarting: %s", err)
	}
}
