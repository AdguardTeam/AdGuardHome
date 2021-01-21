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
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/util"

	"github.com/AdguardTeam/AdGuardHome/internal/sysutil"

	"github.com/AdguardTeam/golibs/log"
)

// getAddrsResponse is the response for /install/get_addresses endpoint.
type getAddrsResponse struct {
	WebPort    int                           `json:"web_port"`
	DNSPort    int                           `json:"dns_port"`
	Interfaces map[string]*util.NetInterface `json:"interfaces"`
}

// handleInstallGetAddresses is the handler for /install/get_addresses endpoint.
func (web *Web) handleInstallGetAddresses(w http.ResponseWriter, r *http.Request) {
	data := getAddrsResponse{}
	data.WebPort = 80
	data.DNSPort = 53

	ifaces, err := util.GetValidNetInterfacesForWeb()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)
		return
	}

	data.Interfaces = make(map[string]*util.NetInterface)
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
	IP      net.IP `json:"ip"`
	Autofix bool   `json:"autofix"`
}

type checkConfigReq struct {
	Web         checkConfigReqEnt `json:"web"`
	DNS         checkConfigReqEnt `json:"dns"`
	SetStaticIP bool              `json:"set_static_ip"`
}

type checkConfigRespEnt struct {
	Status     string `json:"status"`
	CanAutofix bool   `json:"can_autofix"`
}

type staticIPJSON struct {
	Static string `json:"static"`
	IP     string `json:"ip"`
	Error  string `json:"error"`
}

type checkConfigResp struct {
	Web      checkConfigRespEnt `json:"web"`
	DNS      checkConfigRespEnt `json:"dns"`
	StaticIP staticIPJSON       `json:"static_ip"`
}

// Check if ports are available, respond with results
func (web *Web) handleInstallCheckConfig(w http.ResponseWriter, r *http.Request) {
	reqData := checkConfigReq{}
	respData := checkConfigResp{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse 'check_config' JSON data: %s", err)
		return
	}

	if reqData.Web.Port != 0 && reqData.Web.Port != config.BindPort && reqData.Web.Port != config.BetaBindPort {
		err = util.CheckPortAvailable(reqData.Web.IP, reqData.Web.Port)
		if err != nil {
			respData.Web.Status = err.Error()
		}
	}

	if reqData.DNS.Port != 0 {
		err = util.CheckPacketPortAvailable(reqData.DNS.IP, reqData.DNS.Port)

		if util.ErrorIsAddrInUse(err) {
			canAutofix := checkDNSStubListener()
			if canAutofix && reqData.DNS.Autofix {

				err = disableDNSStubListener()
				if err != nil {
					log.Error("Couldn't disable DNSStubListener: %s", err)
				}

				err = util.CheckPacketPortAvailable(reqData.DNS.IP, reqData.DNS.Port)
				canAutofix = false
			}

			respData.DNS.CanAutofix = canAutofix
		}

		if err == nil {
			err = util.CheckPortAvailable(reqData.DNS.IP, reqData.DNS.Port)
		}

		if err != nil {
			respData.DNS.Status = err.Error()
		} else if !reqData.DNS.IP.IsUnspecified() {
			respData.StaticIP = handleStaticIP(reqData.DNS.IP, reqData.SetStaticIP)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(respData)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal JSON: %s", err)
		return
	}
}

// handleStaticIP - handles static IP request
// It either checks if we have a static IP
// Or if set=true, it tries to set it
func handleStaticIP(ip net.IP, set bool) staticIPJSON {
	resp := staticIPJSON{}

	interfaceName := util.GetInterfaceByIP(ip)
	resp.Static = "no"

	if len(interfaceName) == 0 {
		resp.Static = "error"
		resp.Error = fmt.Sprintf("Couldn't find network interface by IP %s", ip)
		return resp
	}

	if set {
		// Try to set static IP for the specified interface
		err := sysutil.IfaceSetStaticIP(interfaceName)
		if err != nil {
			resp.Static = "error"
			resp.Error = err.Error()
			return resp
		}
	}

	// Fallthrough here even if we set static IP
	// Check if we have a static IP and return the details
	isStaticIP, err := sysutil.IfaceHasStaticIP(interfaceName)
	if err != nil {
		resp.Static = "error"
		resp.Error = err.Error()
	} else {
		if isStaticIP {
			resp.Static = "yes"
		}
		resp.IP = util.GetSubnet(interfaceName).String()
	}
	return resp
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
		log.Info("command %s has failed: %v code:%d",
			cmd.Path, err, cmd.ProcessState.ExitCode())
		return false
	}

	cmd = exec.Command("grep", "-E", "#?DNSStubListener=yes", "/etc/systemd/resolved.conf")
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	_, err = cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		log.Info("command %s has failed: %v code:%d",
			cmd.Path, err, cmd.ProcessState.ExitCode())
		return false
	}

	return true
}

const (
	resolvedConfPath = "/etc/systemd/resolved.conf.d/adguardhome.conf"
	resolvedConfData = `[Resolve]
DNS=127.0.0.1
DNSStubListener=no
`
)
const resolvConfPath = "/etc/resolv.conf"

// Deactivate DNSStubListener
func disableDNSStubListener() error {
	dir := filepath.Dir(resolvedConfPath)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return fmt.Errorf("os.MkdirAll: %s: %w", dir, err)
	}

	err = ioutil.WriteFile(resolvedConfPath, []byte(resolvedConfData), 0o644)
	if err != nil {
		return fmt.Errorf("ioutil.WriteFile: %s: %w", resolvedConfPath, err)
	}

	_ = os.Rename(resolvConfPath, resolvConfPath+".backup")
	err = os.Symlink("/run/systemd/resolve/resolv.conf", resolvConfPath)
	if err != nil {
		_ = os.Remove(resolvedConfPath) // remove the file we've just created
		return fmt.Errorf("os.Symlink: %s: %w", resolvConfPath, err)
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
	IP   net.IP `json:"ip"`
	Port int    `json:"port"`
}

type applyConfigReq struct {
	Web      applyConfigReqEnt `json:"web"`
	DNS      applyConfigReqEnt `json:"dns"`
	Username string            `json:"username"`
	Password string            `json:"password"`
}

// Copy installation parameters between two configuration objects
func copyInstallSettings(dst, src *configuration) {
	dst.BindHost = src.BindHost
	dst.BindPort = src.BindPort
	dst.BetaBindPort = src.BetaBindPort
	dst.DNS.BindHost = src.DNS.BindHost
	dst.DNS.Port = src.DNS.Port
}

// Apply new configuration, start DNS server, restart Web server
func (web *Web) handleInstallConfigure(w http.ResponseWriter, r *http.Request) {
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
	if config.BindHost.Equal(newSettings.Web.IP) && config.BindPort == newSettings.Web.Port {
		// no need to rebind
		restartHTTP = false
	}

	// validate that hosts and ports are bindable
	if restartHTTP {
		err = util.CheckPortAvailable(newSettings.Web.IP, newSettings.Web.Port)
		if err != nil {
			httpError(w, http.StatusBadRequest, "Impossible to listen on IP:port %s due to %s",
				net.JoinHostPort(newSettings.Web.IP.String(), strconv.Itoa(newSettings.Web.Port)), err)
			return
		}

	}

	err = util.CheckPacketPortAvailable(newSettings.DNS.IP, newSettings.DNS.Port)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	err = util.CheckPortAvailable(newSettings.DNS.IP, newSettings.DNS.Port)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	var curConfig configuration
	copyInstallSettings(&curConfig, &config)

	Context.firstRun = false
	config.BindHost = newSettings.Web.IP
	config.BindPort = newSettings.Web.Port
	config.DNS.BindHost = newSettings.DNS.IP
	config.DNS.Port = newSettings.DNS.Port

	err = StartMods()
	if err != nil {
		Context.firstRun = true
		copyInstallSettings(&config, &curConfig)
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}

	u := User{}
	u.Name = newSettings.Username
	Context.auth.UserAdd(&u, newSettings.Password)

	err = config.write()
	if err != nil {
		Context.firstRun = true
		copyInstallSettings(&config, &curConfig)
		httpError(w, http.StatusInternalServerError, "Couldn't write config: %s", err)
		return
	}

	web.conf.firstRun = false
	web.conf.BindHost = newSettings.Web.IP
	web.conf.BindPort = newSettings.Web.Port

	registerControlHandlers()

	returnOK(w)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// this needs to be done in a goroutine because Shutdown() is a blocking call, and it will block
	// until all requests are finished, and _we_ are inside a request right now, so it will block indefinitely
	if restartHTTP {
		go func() {
			_ = web.httpServer.Shutdown(context.TODO())
		}()
		if web.httpServerBeta != nil {
			go func() {
				_ = web.httpServerBeta.Shutdown(context.TODO())
			}()
		}
	}
}

func (web *Web) registerInstallHandlers() {
	Context.mux.HandleFunc("/control/install/get_addresses", preInstall(ensureGET(web.handleInstallGetAddresses)))
	Context.mux.HandleFunc("/control/install/check_config", preInstall(ensurePOST(web.handleInstallCheckConfig)))
	Context.mux.HandleFunc("/control/install/configure", preInstall(ensurePOST(web.handleInstallConfigure)))
}

// checkConfigReqEntBeta is a struct representing new client's config check
// request entry.  It supports multiple IP values unlike the checkConfigReqEnt.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default checkConfigReqEnt.
type checkConfigReqEntBeta struct {
	Port    int      `json:"port"`
	IP      []net.IP `json:"ip"`
	Autofix bool     `json:"autofix"`
}

// checkConfigReqBeta is a struct representing new client's config check request
// body.  It uses checkConfigReqEntBeta instead of checkConfigReqEnt.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default checkConfigReq.
type checkConfigReqBeta struct {
	Web         checkConfigReqEntBeta `json:"web"`
	DNS         checkConfigReqEntBeta `json:"dns"`
	SetStaticIP bool                  `json:"set_static_ip"`
}

// handleInstallCheckConfigBeta is a substitution of /install/check_config
// handler for new client.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default handleInstallCheckConfig.
func (web *Web) handleInstallCheckConfigBeta(w http.ResponseWriter, r *http.Request) {
	reqData := checkConfigReqBeta{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse 'check_config' JSON data: %s", err)
		return
	}

	if len(reqData.DNS.IP) == 0 || len(reqData.Web.IP) == 0 {
		httpError(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	nonBetaReqData := checkConfigReq{
		Web: checkConfigReqEnt{
			Port:    reqData.Web.Port,
			IP:      reqData.Web.IP[0],
			Autofix: reqData.Web.Autofix,
		},
		DNS: checkConfigReqEnt{
			Port:    reqData.DNS.Port,
			IP:      reqData.DNS.IP[0],
			Autofix: reqData.DNS.Autofix,
		},
		SetStaticIP: reqData.SetStaticIP,
	}

	nonBetaReqBody := &strings.Builder{}

	err = json.NewEncoder(nonBetaReqBody).Encode(nonBetaReqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to encode 'check_config' JSON data: %s", err)
		return
	}
	body := nonBetaReqBody.String()
	r.Body = ioutil.NopCloser(strings.NewReader(body))
	r.ContentLength = int64(len(body))

	web.handleInstallCheckConfig(w, r)
}

// applyConfigReqEntBeta is a struct representing new client's config setting
// request entry.  It supports multiple IP values unlike the applyConfigReqEnt.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default applyConfigReqEnt.
type applyConfigReqEntBeta struct {
	IP   []net.IP `json:"ip"`
	Port int      `json:"port"`
}

// applyConfigReqBeta is a struct representing new client's config setting
// request body.  It uses applyConfigReqEntBeta instead of applyConfigReqEnt.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default applyConfigReq.
type applyConfigReqBeta struct {
	Web      applyConfigReqEntBeta `json:"web"`
	DNS      applyConfigReqEntBeta `json:"dns"`
	Username string                `json:"username"`
	Password string                `json:"password"`
}

// handleInstallConfigureBeta is a substitution of /install/configure handler
// for new client.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default handleInstallConfigure.
func (web *Web) handleInstallConfigureBeta(w http.ResponseWriter, r *http.Request) {
	reqData := applyConfigReqBeta{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse 'check_config' JSON data: %s", err)
		return
	}

	if len(reqData.DNS.IP) == 0 || len(reqData.Web.IP) == 0 {
		httpError(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	nonBetaReqData := applyConfigReq{
		Web: applyConfigReqEnt{
			IP:   reqData.Web.IP[0],
			Port: reqData.Web.Port,
		},
		DNS: applyConfigReqEnt{
			IP:   reqData.DNS.IP[0],
			Port: reqData.DNS.Port,
		},
		Username: reqData.Username,
		Password: reqData.Password,
	}

	nonBetaReqBody := &strings.Builder{}

	err = json.NewEncoder(nonBetaReqBody).Encode(nonBetaReqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to encode 'check_config' JSON data: %s", err)
		return
	}
	body := nonBetaReqBody.String()
	r.Body = ioutil.NopCloser(strings.NewReader(body))
	r.ContentLength = int64(len(body))

	web.handleInstallConfigure(w, r)
}

// getAddrsResponseBeta is a struct representing new client's getting addresses
// request body.  It uses array of structs instead of map.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default firstRunData.
type getAddrsResponseBeta struct {
	WebPort    int                  `json:"web_port"`
	DNSPort    int                  `json:"dns_port"`
	Interfaces []*util.NetInterface `json:"interfaces"`
}

// handleInstallConfigureBeta is a substitution of /install/get_addresses
// handler for new client.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default handleInstallGetAddresses.
func (web *Web) handleInstallGetAddressesBeta(w http.ResponseWriter, r *http.Request) {
	data := getAddrsResponseBeta{}
	data.WebPort = 80
	data.DNSPort = 53

	ifaces, err := util.GetValidNetInterfacesForWeb()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)
		return
	}

	data.Interfaces = ifaces

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal default addresses to json: %s", err)
		return
	}
}

// registerBetaInstallHandlers registers the install handlers for new client
// with the structures it supports.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default handlers.
func (web *Web) registerBetaInstallHandlers() {
	Context.mux.HandleFunc("/control/install/get_addresses_beta", preInstall(ensureGET(web.handleInstallGetAddressesBeta)))
	Context.mux.HandleFunc("/control/install/check_config_beta", preInstall(ensurePOST(web.handleInstallCheckConfigBeta)))
	Context.mux.HandleFunc("/control/install/configure_beta", preInstall(ensurePOST(web.handleInstallConfigureBeta)))
}
