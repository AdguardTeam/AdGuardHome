package home

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/golibs/log"
	"github.com/NYTimes/gziphandler"
)

// ----------------
// helper functions
// ----------------

func returnOK(w http.ResponseWriter) {
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func httpOK(r *http.Request, w http.ResponseWriter) {
}

func httpError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info(text)
	http.Error(w, text, code)
}

// ---------------
// dns run control
// ---------------
func writeAllConfigsAndReloadDNS() error {
	err := writeAllConfigs()
	if err != nil {
		log.Error("Couldn't write all configs: %s", err)
		return err
	}
	return reconfigureDNSServer()
}

func addDNSAddress(dnsAddresses *[]string, addr string) {
	if config.DNS.Port != 53 {
		addr = fmt.Sprintf("%s:%d", addr, config.DNS.Port)
	}
	*dnsAddresses = append(*dnsAddresses, addr)
}

// Get the list of DNS addresses the server is listening on
func getDNSAddresses() []string {
	dnsAddresses := []string{}

	if config.DNS.BindHost == "0.0.0.0" {

		ifaces, e := getValidNetInterfacesForWeb()
		if e != nil {
			log.Error("Couldn't get network interfaces: %v", e)
			return []string{}
		}

		for _, iface := range ifaces {
			for _, addr := range iface.Addresses {
				addDNSAddress(&dnsAddresses, addr)
			}
		}

	} else {
		addDNSAddress(&dnsAddresses, config.DNS.BindHost)
	}

	if config.TLS.Enabled && len(config.TLS.ServerName) != 0 {

		if config.TLS.PortHTTPS != 0 {
			addr := config.TLS.ServerName
			if config.TLS.PortHTTPS != 443 {
				addr = fmt.Sprintf("%s:%d", addr, config.TLS.PortHTTPS)
			}
			addr = fmt.Sprintf("https://%s/dns-query", addr)
			dnsAddresses = append(dnsAddresses, addr)
		}

		if config.TLS.PortDNSOverTLS != 0 {
			addr := fmt.Sprintf("tls://%s:%d", config.TLS.ServerName, config.TLS.PortDNSOverTLS)
			dnsAddresses = append(dnsAddresses, addr)
		}
	}

	return dnsAddresses
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	c := dnsforward.FilteringConfig{}
	if Context.dnsServer != nil {
		Context.dnsServer.WriteDiskConfig(&c)
	}
	data := map[string]interface{}{
		"dns_addresses": getDNSAddresses(),
		"http_port":     config.BindPort,
		"dns_port":      config.DNS.Port,
		"running":       isRunning(),
		"version":       versionString,
		"language":      config.Language,

		"protection_enabled": c.ProtectionEnabled,
		"bootstrap_dns":      c.BootstrapDNS,
		"upstream_dns":       c.UpstreamDNS,
		"all_servers":        c.AllServers,
	}

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

type profileJSON struct {
	Name string `json:"name"`
}

func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	pj := profileJSON{}
	u := config.auth.GetCurrentUser(r)
	pj.Name = u.Name

	data, err := json.Marshal(pj)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json.Marshal: %s", err)
		return
	}
	_, _ = w.Write(data)
}

// --------------
// DNS-over-HTTPS
// --------------
func handleDOH(w http.ResponseWriter, r *http.Request) {
	if !config.TLS.AllowUnencryptedDOH && r.TLS == nil {
		httpError(w, http.StatusNotFound, "Not Found")
		return
	}

	if !isRunning() {
		httpError(w, http.StatusInternalServerError, "DNS server is not running")
		return
	}

	Context.dnsServer.ServeHTTP(w, r)
}

// ------------------------
// registration of handlers
// ------------------------
func registerControlHandlers() {
	httpRegister(http.MethodGet, "/control/status", handleStatus)
	httpRegister(http.MethodPost, "/control/i18n/change_language", handleI18nChangeLanguage)
	httpRegister(http.MethodGet, "/control/i18n/current_language", handleI18nCurrentLanguage)
	http.HandleFunc("/control/version.json", postInstall(optionalAuth(handleGetVersionJSON)))
	httpRegister(http.MethodPost, "/control/update", handleUpdate)

	httpRegister("GET", "/control/profile", handleGetProfile)

	RegisterFilteringHandlers()
	RegisterTLSHandlers()
	RegisterBlockedServicesHandlers()
	RegisterAuthHandlers()

	http.HandleFunc("/dns-query", postInstall(handleDOH))
}

func httpRegister(method string, url string, handler func(http.ResponseWriter, *http.Request)) {
	http.Handle(url, postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(ensureHandler(method, handler)))))
}
