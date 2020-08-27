package home

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/update"
	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
)

type getVersionJSONRequest struct {
	RecheckNow bool `json:"recheck_now"`
}

// Get the latest available version from the Internet
func handleGetVersionJSON(w http.ResponseWriter, r *http.Request) {
	if Context.disableUpdate {
		resp := make(map[string]interface{})
		resp["disabled"] = true
		d, _ := json.Marshal(resp)
		_, _ = w.Write(d)
		return
	}

	req := getVersionJSONRequest{}
	var err error
	if r.ContentLength != 0 {
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			httpError(w, http.StatusBadRequest, "JSON parse: %s", err)
			return
		}
	}

	var info update.VersionInfo
	for i := 0; i != 3; i++ {
		Context.controlLock.Lock()
		info, err = Context.updater.GetVersionResponse(req.RecheckNow)
		Context.controlLock.Unlock()
		if err != nil && strings.HasSuffix(err.Error(), "i/o timeout") {
			// This case may happen while we're restarting DNS server
			// https://github.com/AdguardTeam/AdGuardHome/issues/934
			continue
		}
		break
	}
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't get version check json from %s: %T %s\n", versionCheckURL, err, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(getVersionResp(info))
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

// Perform an update procedure to the latest available version
func handleUpdate(w http.ResponseWriter, _ *http.Request) {
	if len(Context.updater.NewVersion) == 0 {
		httpError(w, http.StatusBadRequest, "/update request isn't allowed now")
		return
	}

	err := Context.updater.DoUpdate()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}

	returnOK(w)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	go finishUpdate()
}

// Convert version.json data to our JSON response
func getVersionResp(info update.VersionInfo) []byte {
	ret := make(map[string]interface{})
	ret["can_autoupdate"] = false
	ret["new_version"] = info.NewVersion
	ret["announcement"] = info.Announcement
	ret["announcement_url"] = info.AnnouncementURL

	if info.CanAutoUpdate {
		canUpdate := true

		tlsConf := tlsConfigSettings{}
		Context.tls.WriteDiskConfig(&tlsConf)

		if runtime.GOOS != "windows" &&
			((tlsConf.Enabled && (tlsConf.PortHTTPS < 1024 ||
				tlsConf.PortDNSOverTLS < 1024 ||
				tlsConf.PortDNSOverQUIC < 1024)) ||
				config.BindPort < 1024 ||
				config.DNS.Port < 1024) {
			// On UNIX, if we're running under a regular user,
			//  but with CAP_NET_BIND_SERVICE set on a binary file,
			//  and we're listening on ports <1024,
			//  we won't be able to restart after we replace the binary file,
			//  because we'll lose CAP_NET_BIND_SERVICE capability.
			canUpdate, _ = util.HaveAdminRights()
		}
		ret["can_autoupdate"] = canUpdate
	}

	d, _ := json.Marshal(ret)
	return d
}

// Complete an update procedure
func finishUpdate() {
	log.Info("Stopping all tasks")
	cleanup()
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
