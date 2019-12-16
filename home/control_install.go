package home

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/AdguardTeam/golibs/log"
)

type firstRunData struct {
	WebPort    int                    `json:"web_port"`
	DNSPort    int                    `json:"dns_port"`
	Interfaces map[string]interface{} `json:"interfaces"`
}

// Get initial installation settings
func handleInstallGetAddresses(w http.ResponseWriter, r *http.Request) {
	data := firstRunData{}
	data.WebPort = 80
	data.DNSPort = 53

	ifaces, err := getValidNetInterfacesForWeb()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)
		return
	}

	data.Interfaces = make(map[string]interface{})
	for _, iface := range ifaces {
		data.Interfaces[iface.Name] = iface
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal default addresses to json: %s", err)
		return
	}
}

type checkConfigReqEnt struct {
	Port    int    `json:"port"`
	IP      string `json:"ip"`
	Autofix bool   `json:"autofix"`
}
type checkConfigReq struct {
	Web checkConfigReqEnt `json:"web"`
	DNS checkConfigReqEnt `json:"dns"`
}

type checkConfigRespEnt struct {
	Status     string `json:"status"`
	CanAutofix bool   `json:"can_autofix"`
}
type checkConfigResp struct {
	Web checkConfigRespEnt `json:"web"`
	DNS checkConfigRespEnt `json:"dns"`
}

// Check if ports are available, respond with results
func handleInstallCheckConfig(w http.ResponseWriter, r *http.Request) {
	reqData := checkConfigReq{}
	respData := checkConfigResp{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse 'check_config' JSON data: %s", err)
		return
	}

	if reqData.Web.Port != 0 && reqData.Web.Port != config.BindPort {
		err = checkPortAvailable(reqData.Web.IP, reqData.Web.Port)
		if err != nil {
			respData.Web.Status = fmt.Sprintf("%v", err)
		}
	}

	if reqData.DNS.Port != 0 {
		err = checkPacketPortAvailable(reqData.DNS.IP, reqData.DNS.Port)

		if errorIsAddrInUse(err) {
			canAutofix := checkDNSStubListener()
			if canAutofix && reqData.DNS.Autofix {

				err = disableDNSStubListener()
				if err != nil {
					log.Error("Couldn't disable DNSStubListener: %s", err)
				}

				err = checkPacketPortAvailable(reqData.DNS.IP, reqData.DNS.Port)
				canAutofix = false
			}

			respData.DNS.CanAutofix = canAutofix
		}

		if err == nil {
			err = checkPortAvailable(reqData.DNS.IP, reqData.DNS.Port)
		}

		if err != nil {
			respData.DNS.Status = fmt.Sprintf("%v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(respData)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal JSON: %s", err)
		return
	}
}

// Check if DNSStubListener is active
func checkDNSStubListener() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	cmd := exec.Command("systemctl", "is-enabled", "systemd-resolved")
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	_, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		log.Error("command %s has failed: %v code:%d",
			cmd.Path, err, cmd.ProcessState.ExitCode())
		return false
	}

	cmd = exec.Command("grep", "-E", "#?DNSStubListener=yes", "/etc/systemd/resolved.conf")
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	_, err = cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		log.Error("command %s has failed: %v code:%d",
			cmd.Path, err, cmd.ProcessState.ExitCode())
		return false
	}

	return true
}

const resolvedConfPath = "/etc/systemd/resolved.conf.d/adguardhome.conf"
const resolvedConfData = `[Resolve]
DNS=127.0.0.1
DNSStubListener=no
`
const resolvConfPath = "/etc/resolv.conf"

// Deactivate DNSStubListener
func disableDNSStubListener() error {
	dir := filepath.Dir(resolvedConfPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("os.MkdirAll: %s: %s", dir, err)
	}

	err = ioutil.WriteFile(resolvedConfPath, []byte(resolvedConfData), 0644)
	if err != nil {
		return fmt.Errorf("ioutil.WriteFile: %s: %s", resolvedConfPath, err)
	}

	_ = os.Rename(resolvConfPath, resolvConfPath+".backup")
	err = os.Symlink("/run/systemd/resolve/resolv.conf", resolvConfPath)
	if err != nil {
		_ = os.Remove(resolvedConfPath) // remove the file we've just created
		return fmt.Errorf("os.Symlink: %s: %s", resolvConfPath, err)
	}

	cmd := exec.Command("systemctl", "reload-or-restart", "systemd-resolved")
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	_, err = cmd.Output()
	if err != nil {
		return err
	}
	if cmd.ProcessState.ExitCode() != 0 {
		return fmt.Errorf("process %s exited with an error: %d",
			cmd.Path, cmd.ProcessState.ExitCode())
	}

	return nil
}

type applyConfigReqEnt struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}
type applyConfigReq struct {
	Web      applyConfigReqEnt `json:"web"`
	DNS      applyConfigReqEnt `json:"dns"`
	Username string            `json:"username"`
	Password string            `json:"password"`
}

// Copy installation parameters between two configuration objects
func copyInstallSettings(dst *configuration, src *configuration) {
	dst.BindHost = src.BindHost
	dst.BindPort = src.BindPort
	dst.DNS.BindHost = src.DNS.BindHost
	dst.DNS.Port = src.DNS.Port
}

// Apply new configuration, start DNS server, restart Web server
func handleInstallConfigure(w http.ResponseWriter, r *http.Request) {
	newSettings := applyConfigReq{}
	err := json.NewDecoder(r.Body).Decode(&newSettings)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse 'configure' JSON: %s", err)
		return
	}

	if newSettings.Web.Port == 0 || newSettings.DNS.Port == 0 {
		httpError(w, http.StatusBadRequest, "port value can't be 0")
		return
	}

	restartHTTP := true
	if config.BindHost == newSettings.Web.IP && config.BindPort == newSettings.Web.Port {
		// no need to rebind
		restartHTTP = false
	}

	// validate that hosts and ports are bindable
	if restartHTTP {
		err = checkPortAvailable(newSettings.Web.IP, newSettings.Web.Port)
		if err != nil {
			httpError(w, http.StatusBadRequest, "Impossible to listen on IP:port %s due to %s",
				net.JoinHostPort(newSettings.Web.IP, strconv.Itoa(newSettings.Web.Port)), err)
			return
		}
	}

	err = checkPacketPortAvailable(newSettings.DNS.IP, newSettings.DNS.Port)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	err = checkPortAvailable(newSettings.DNS.IP, newSettings.DNS.Port)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	var curConfig configuration
	copyInstallSettings(&curConfig, &config)

	config.firstRun = false
	config.BindHost = newSettings.Web.IP
	config.BindPort = newSettings.Web.Port
	config.DNS.BindHost = newSettings.DNS.IP
	config.DNS.Port = newSettings.DNS.Port

	err = initDNSServer()
	var err2 error
	if err == nil {
		err2 = startDNSServer()
		if err2 != nil {
			closeDNSServer()
		}
	}
	if err != nil || err2 != nil {
		config.firstRun = true
		copyInstallSettings(&config, &curConfig)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "Couldn't initialize DNS server: %s", err)
		} else {
			httpError(w, http.StatusInternalServerError, "Couldn't start DNS server: %s", err2)
		}
		return
	}

	u := User{}
	u.Name = newSettings.Username
	config.auth.UserAdd(&u, newSettings.Password)

	err = config.write()
	if err != nil {
		config.firstRun = true
		copyInstallSettings(&config, &curConfig)
		httpError(w, http.StatusInternalServerError, "Couldn't write config: %s", err)
		return
	}

	// this needs to be done in a goroutine because Shutdown() is a blocking call, and it will block
	// until all requests are finished, and _we_ are inside a request right now, so it will block indefinitely
	if restartHTTP {
		go func() {
			_ = Context.httpServer.Shutdown(context.TODO())
		}()
	}

	returnOK(w)
}

func registerInstallHandlers() {
	http.HandleFunc("/control/install/get_addresses", preInstall(ensureGET(handleInstallGetAddresses)))
	http.HandleFunc("/control/install/check_config", preInstall(ensurePOST(handleInstallCheckConfig)))
	http.HandleFunc("/control/install/configure", preInstall(ensurePOST(handleInstallConfigure)))
}
