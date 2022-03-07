package home

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// getAddrsResponse is the response for /install/get_addresses endpoint.
type getAddrsResponse struct {
	Interfaces map[string]*aghnet.NetInterface `json:"interfaces"`

	// Version is the version of AdGuard Home.
	//
	// TODO(a.garipov): In the new API, rename this endpoint to something more
	// general, since there will be more information here than just network
	// interfaces.
	Version string `json:"version"`

	WebPort int `json:"web_port"`
	DNSPort int `json:"dns_port"`
}

// handleInstallGetAddresses is the handler for /install/get_addresses endpoint.
func (web *Web) handleInstallGetAddresses(w http.ResponseWriter, r *http.Request) {
	data := getAddrsResponse{
		Version: version.Version(),

		WebPort: defaultPortHTTP,
		DNSPort: defaultPortDNS,
	}

	ifaces, err := aghnet.GetValidNetInterfacesForWeb()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)

		return
	}

	data.Interfaces = make(map[string]*aghnet.NetInterface)
	for _, iface := range ifaces {
		data.Interfaces[iface.Name] = iface
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusInternalServerError,
			"Unable to marshal default addresses to json: %s",
			err,
		)

		return
	}
}

type checkConfReqEnt struct {
	IP      net.IP `json:"ip"`
	Port    int    `json:"port"`
	Autofix bool   `json:"autofix"`
}

type checkConfReq struct {
	Web         checkConfReqEnt `json:"web"`
	DNS         checkConfReqEnt `json:"dns"`
	SetStaticIP bool            `json:"set_static_ip"`
}

type checkConfRespEnt struct {
	Status     string `json:"status"`
	CanAutofix bool   `json:"can_autofix"`
}

type staticIPJSON struct {
	Static string `json:"static"`
	IP     string `json:"ip"`
	Error  string `json:"error"`
}

type checkConfResp struct {
	StaticIP staticIPJSON     `json:"static_ip"`
	Web      checkConfRespEnt `json:"web"`
	DNS      checkConfRespEnt `json:"dns"`
}

// validateWeb returns error is the web part if the initial configuration can't
// be set.
func (req *checkConfReq) validateWeb(uc aghalg.UniqChecker) (err error) {
	defer func() { err = errors.Annotate(err, "validating ports: %w") }()

	port := req.Web.Port
	addPorts(uc, config.BetaBindPort, port)
	if err = uc.Validate(aghalg.IntIsBefore); err != nil {
		// Avoid duplicating the error into the status of DNS.
		uc[port] = 1

		return err
	}

	switch port {
	case 0, config.BindPort:
		return nil
	default:
		// Go on and check the port binding only if it's not zero or won't be
		// unbound after install.
	}

	return aghnet.CheckPort("tcp", req.Web.IP, port)
}

// validateDNS returns error if the DNS part of the initial configuration can't
// be set.  canAutofix is true if the port can be unbound by AdGuard Home
// automatically.
func (req *checkConfReq) validateDNS(uc aghalg.UniqChecker) (canAutofix bool, err error) {
	defer func() { err = errors.Annotate(err, "validating ports: %w") }()

	port := req.DNS.Port
	addPorts(uc, port)
	if err = uc.Validate(aghalg.IntIsBefore); err != nil {
		return false, err
	}

	switch port {
	case 0:
		return false, nil
	case config.BindPort:
		// Go on and only check the UDP port since the TCP one is already bound
		// by AdGuard Home for web interface.
	default:
		// Check TCP as well.
		err = aghnet.CheckPort("tcp", req.DNS.IP, port)
		if err != nil {
			return false, err
		}
	}

	err = aghnet.CheckPort("udp", req.DNS.IP, port)
	if !aghnet.IsAddrInUse(err) {
		return false, err
	}

	// Try to fix automatically.
	canAutofix = checkDNSStubListener()
	if canAutofix && req.DNS.Autofix {
		if derr := disableDNSStubListener(); derr != nil {
			log.Error("disabling DNSStubListener: %s", err)
		}

		err = aghnet.CheckPort("udp", req.DNS.IP, port)
		canAutofix = false
	}

	return canAutofix, err
}

// handleInstallCheckConfig handles the /check_config endpoint.
func (web *Web) handleInstallCheckConfig(w http.ResponseWriter, r *http.Request) {
	req := &checkConfReq{}

	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "decoding the request: %s", err)

		return
	}

	resp := &checkConfResp{}
	uc := aghalg.UniqChecker{}

	if err = req.validateWeb(uc); err != nil {
		resp.Web.Status = err.Error()
	}

	if resp.DNS.CanAutofix, err = req.validateDNS(uc); err != nil {
		resp.DNS.Status = err.Error()
	} else if !req.DNS.IP.IsUnspecified() {
		resp.StaticIP = handleStaticIP(req.DNS.IP, req.SetStaticIP)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "encoding the response: %s", err)

		return
	}
}

// handleStaticIP - handles static IP request
// It either checks if we have a static IP
// Or if set=true, it tries to set it
func handleStaticIP(ip net.IP, set bool) staticIPJSON {
	resp := staticIPJSON{}

	interfaceName := aghnet.GetInterfaceByIP(ip)
	resp.Static = "no"

	if len(interfaceName) == 0 {
		resp.Static = "error"
		resp.Error = fmt.Sprintf("Couldn't find network interface by IP %s", ip)
		return resp
	}

	if set {
		// Try to set static IP for the specified interface
		err := aghnet.IfaceSetStaticIP(interfaceName)
		if err != nil {
			resp.Static = "error"
			resp.Error = err.Error()
			return resp
		}
	}

	// Fallthrough here even if we set static IP
	// Check if we have a static IP and return the details
	isStaticIP, err := aghnet.IfaceHasStaticIP(interfaceName)
	if err != nil {
		resp.Static = "error"
		resp.Error = err.Error()
	} else {
		if isStaticIP {
			resp.Static = "yes"
		}
		resp.IP = aghnet.GetSubnet(interfaceName).String()
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

	err = os.WriteFile(resolvedConfPath, []byte(resolvedConfData), 0o644)
	if err != nil {
		return fmt.Errorf("os.WriteFile: %s: %w", resolvedConfPath, err)
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
	Username string `json:"username"`
	Password string `json:"password"`

	Web applyConfigReqEnt `json:"web"`
	DNS applyConfigReqEnt `json:"dns"`
}

// copyInstallSettings copies the installation parameters between two
// configuration structures.
func copyInstallSettings(dst, src *configuration) {
	dst.BindHost = src.BindHost
	dst.BindPort = src.BindPort
	dst.BetaBindPort = src.BetaBindPort
	dst.DNS.BindHosts = src.DNS.BindHosts
	dst.DNS.Port = src.DNS.Port
}

// shutdownTimeout is the timeout for shutting HTTP server down operation.
const shutdownTimeout = 5 * time.Second

func shutdownSrv(ctx context.Context, srv *http.Server) {
	defer log.OnPanic("")

	if srv == nil {
		return
	}

	err := srv.Shutdown(ctx)
	if err != nil {
		const msgFmt = "shutting down http server %q: %s"
		if errors.Is(err, context.Canceled) {
			log.Debug(msgFmt, srv.Addr, err)
		} else {
			log.Error(msgFmt, srv.Addr, err)
		}
	}
}

// PasswordMinRunes is the minimum length of user's password in runes.
const PasswordMinRunes = 8

// Apply new configuration, start DNS server, restart Web server
func (web *Web) handleInstallConfigure(w http.ResponseWriter, r *http.Request) {
	req, restartHTTP, err := decodeApplyConfigReq(r.Body)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	if utf8.RuneCountInString(req.Password) < PasswordMinRunes {
		aghhttp.Error(
			r,
			w,
			http.StatusUnprocessableEntity,
			"password must be at least %d symbols long",
			PasswordMinRunes,
		)

		return
	}

	err = aghnet.CheckPort("udp", req.DNS.IP, req.DNS.Port)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	err = aghnet.CheckPort("tcp", req.DNS.IP, req.DNS.Port)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	curConfig := &configuration{}
	copyInstallSettings(curConfig, config)

	Context.firstRun = false
	config.BindHost = req.Web.IP
	config.BindPort = req.Web.Port
	config.DNS.BindHosts = []net.IP{req.DNS.IP}
	config.DNS.Port = req.DNS.Port

	// TODO(e.burkov): StartMods() should be put in a separate goroutine at the
	// moment we'll allow setting up TLS in the initial configuration or the
	// configuration itself will use HTTPS protocol, because the underlying
	// functions potentially restart the HTTPS server.
	err = StartMods()
	if err != nil {
		Context.firstRun = true
		copyInstallSettings(config, curConfig)
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	u := &User{
		Name: req.Username,
	}
	Context.auth.UserAdd(u, req.Password)

	err = config.write()
	if err != nil {
		Context.firstRun = true
		copyInstallSettings(config, curConfig)
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't write config: %s", err)

		return
	}

	web.conf.firstRun = false
	web.conf.BindHost = req.Web.IP
	web.conf.BindPort = req.Web.Port

	registerControlHandlers()

	aghhttp.OK(w)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	if !restartHTTP {
		return
	}

	// Method http.(*Server).Shutdown needs to be called in a separate goroutine
	// and with its own context, because it waits until all requests are handled
	// and will be blocked by it's own caller.
	go func(timeout time.Duration) {
		defer log.OnPanic("web")

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		shutdownSrv(ctx, web.httpServer)
		shutdownSrv(ctx, web.httpServerBeta)
	}(shutdownTimeout)
}

// decodeApplyConfigReq decodes the configuration, validates some parameters,
// and returns it along with the boolean indicating whether or not the HTTP
// server must be restarted.
func decodeApplyConfigReq(r io.Reader) (req *applyConfigReq, restartHTTP bool, err error) {
	req = &applyConfigReq{}
	err = json.NewDecoder(r).Decode(&req)
	if err != nil {
		return nil, false, fmt.Errorf("parsing request: %w", err)
	}

	if req.Web.Port == 0 || req.DNS.Port == 0 {
		return nil, false, errors.Error("ports cannot be 0")
	}

	restartHTTP = !config.BindHost.Equal(req.Web.IP) || config.BindPort != req.Web.Port
	if restartHTTP {
		err = aghnet.CheckPort("tcp", req.Web.IP, req.Web.Port)
		if err != nil {
			return nil, false, fmt.Errorf(
				"checking address %s:%d: %w",
				req.Web.IP.String(),
				req.Web.Port,
				err,
			)
		}
	}

	return req, restartHTTP, err
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
	IP      []net.IP `json:"ip"`
	Port    int      `json:"port"`
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
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to parse 'check_config' JSON data: %s", err)

		return
	}

	if len(reqData.DNS.IP) == 0 || len(reqData.Web.IP) == 0 {
		aghhttp.Error(r, w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))

		return
	}

	nonBetaReqData := checkConfReq{
		Web: checkConfReqEnt{
			IP:      reqData.Web.IP[0],
			Port:    reqData.Web.Port,
			Autofix: reqData.Web.Autofix,
		},
		DNS: checkConfReqEnt{
			IP:      reqData.DNS.IP[0],
			Port:    reqData.DNS.Port,
			Autofix: reqData.DNS.Autofix,
		},
		SetStaticIP: reqData.SetStaticIP,
	}

	nonBetaReqBody := &strings.Builder{}

	err = json.NewEncoder(nonBetaReqBody).Encode(nonBetaReqData)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"Failed to encode 'check_config' JSON data: %s",
			err,
		)

		return
	}
	body := nonBetaReqBody.String()
	r.Body = io.NopCloser(strings.NewReader(body))
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
	Username string `json:"username"`
	Password string `json:"password"`

	Web applyConfigReqEntBeta `json:"web"`
	DNS applyConfigReqEntBeta `json:"dns"`
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
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to parse 'check_config' JSON data: %s", err)

		return
	}

	if len(reqData.DNS.IP) == 0 || len(reqData.Web.IP) == 0 {
		aghhttp.Error(r, w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))

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
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"Failed to encode 'check_config' JSON data: %s",
			err,
		)

		return
	}
	body := nonBetaReqBody.String()
	r.Body = io.NopCloser(strings.NewReader(body))
	r.ContentLength = int64(len(body))

	web.handleInstallConfigure(w, r)
}

// getAddrsResponseBeta is a struct representing new client's getting addresses
// request body.  It uses array of structs instead of map.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default firstRunData.
type getAddrsResponseBeta struct {
	Interfaces []*aghnet.NetInterface `json:"interfaces"`
	WebPort    int                    `json:"web_port"`
	DNSPort    int                    `json:"dns_port"`
}

// handleInstallConfigureBeta is a substitution of /install/get_addresses
// handler for new client.
//
// TODO(e.burkov): This should removed with the API v1 when the appropriate
// functionality will appear in default handleInstallGetAddresses.
func (web *Web) handleInstallGetAddressesBeta(w http.ResponseWriter, r *http.Request) {
	data := getAddrsResponseBeta{
		WebPort: defaultPortHTTP,
		DNSPort: defaultPortDNS,
	}

	ifaces, err := aghnet.GetValidNetInterfacesForWeb()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)

		return
	}

	data.Interfaces = ifaces

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusInternalServerError,
			"Unable to marshal default addresses to json: %s",
			err,
		)

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
