package dhcpd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info("DHCP: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

// []Lease -> JSON
func convertLeases(inputLeases []Lease, includeExpires bool) []map[string]string {
	leases := []map[string]string{}
	for _, l := range inputLeases {
		lease := map[string]string{
			"mac":      l.HWAddr.String(),
			"ip":       l.IP.String(),
			"hostname": l.Hostname,
		}

		if includeExpires {
			lease["expires"] = l.Expiry.Format(time.RFC3339)
		}

		leases = append(leases, lease)
	}
	return leases
}

func (s *Server) handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	leases := convertLeases(s.Leases(LeasesDynamic), true)
	staticLeases := convertLeases(s.Leases(LeasesStatic), false)
	status := map[string]interface{}{
		"config":        s.conf,
		"leases":        leases,
		"static_leases": staticLeases,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal DHCP status json: %s", err)
		return
	}
}

type staticLeaseJSON struct {
	HWAddr   string `json:"mac"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

type dhcpServerConfigJSON struct {
	ServerConfig `json:",inline"`
	StaticLeases []staticLeaseJSON `json:"static_leases"`
}

func (s *Server) handleDHCPSetConfig(w http.ResponseWriter, r *http.Request) {
	newconfig := dhcpServerConfigJSON{}
	err := json.NewDecoder(r.Body).Decode(&newconfig)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Failed to parse new DHCP config json: %s", err)
		return
	}

	err = s.CheckConfig(newconfig.ServerConfig)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Invalid DHCP configuration: %s", err)
		return
	}

	err = s.Stop()
	if err != nil {
		log.Error("failed to stop the DHCP server: %s", err)
	}

	err = s.Init(newconfig.ServerConfig)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Invalid DHCP configuration: %s", err)
		return
	}
	s.conf.ConfigModified()

	if newconfig.Enabled {

		staticIP, err := hasStaticIP(newconfig.InterfaceName)
		if !staticIP && err == nil {
			err = setStaticIP(newconfig.InterfaceName)
			if err != nil {
				httpError(r, w, http.StatusInternalServerError, "Failed to configure static IP: %s", err)
				return
			}
		}

		err = s.Start()
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "Failed to start DHCP server: %s", err)
			return
		}
	}
}

type netInterface struct {
	Name         string   `json:"name"`
	MTU          int      `json:"mtu"`
	HardwareAddr string   `json:"hardware_address"`
	Addresses    []string `json:"ip_addresses"`
	Flags        string   `json:"flags"`
}

// getValidNetInterfaces returns interfaces that are eligible for DNS and/or DHCP
// invalid interface is a ppp interface or the one that doesn't allow broadcasts
func getValidNetInterfaces() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get list of interfaces: %s", err)
	}

	netIfaces := []net.Interface{}

	for i := range ifaces {
		if ifaces[i].Flags&net.FlagPointToPoint != 0 {
			// this interface is ppp, we're not interested in this one
			continue
		}

		iface := ifaces[i]
		netIfaces = append(netIfaces, iface)
	}

	return netIfaces, nil
}

func (s *Server) handleDHCPInterfaces(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{}

	ifaces, err := getValidNetInterfaces()
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)
		return
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			// it's a loopback, skip it
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			// this interface doesn't support broadcast, skip it
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			httpError(r, w, http.StatusInternalServerError, "Failed to get addresses for interface %s: %s", iface.Name, err)
			return
		}

		jsonIface := netInterface{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr.String(),
		}

		if iface.Flags != 0 {
			jsonIface.Flags = iface.Flags.String()
		}
		// we don't want link-local addresses in json, so skip them
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				// not an IPNet, should not happen
				httpError(r, w, http.StatusInternalServerError, "SHOULD NOT HAPPEN: got iface.Addrs() element %s that is not net.IPNet, it is %T", addr, addr)
				return
			}
			// ignore link-local
			if ipnet.IP.IsLinkLocalUnicast() {
				continue
			}
			jsonIface.Addresses = append(jsonIface.Addresses, ipnet.IP.String())
		}
		if len(jsonIface.Addresses) != 0 {
			response[iface.Name] = jsonIface
		}

	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Failed to marshal json with available interfaces: %s", err)
		return
	}
}

// Perform the following tasks:
// . Search for another DHCP server running
// . Check if a static IP is configured for the network interface
// Respond with results
func (s *Server) handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to read request body: %s", err)
		log.Error(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	interfaceName := strings.TrimSpace(string(body))
	if interfaceName == "" {
		errorText := fmt.Sprintf("empty interface name specified")
		log.Error(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	found, err := CheckIfOtherDHCPServersPresent(interfaceName)

	othSrv := map[string]interface{}{}
	foundVal := "no"
	if found {
		foundVal = "yes"
	} else if err != nil {
		foundVal = "error"
		othSrv["error"] = err.Error()
	}
	othSrv["found"] = foundVal

	staticIP := map[string]interface{}{}
	isStaticIP, err := hasStaticIP(interfaceName)
	staticIPStatus := "yes"
	if err != nil {
		staticIPStatus = "error"
		staticIP["error"] = err.Error()
	} else if !isStaticIP {
		staticIPStatus = "no"
		staticIP["ip"] = getFullIP(interfaceName)
	}
	staticIP["static"] = staticIPStatus

	result := map[string]interface{}{}
	result["other_server"] = othSrv
	result["static_ip"] = staticIP

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Failed to marshal DHCP found json: %s", err)
		return
	}
}

// Check if network interface has a static IP configured
func hasStaticIP(ifaceName string) (bool, error) {
	if runtime.GOOS == "windows" {
		return false, errors.New("Can't detect static IP: not supported on Windows")
	}

	body, err := ioutil.ReadFile("/etc/dhcpcd.conf")
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(body), "\n")
	nameLine := fmt.Sprintf("interface %s", ifaceName)
	withinInterfaceCtx := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if withinInterfaceCtx && len(line) == 0 {
			// an empty line resets our state
			withinInterfaceCtx = false
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}
		line = strings.TrimSpace(line)

		if !withinInterfaceCtx {
			if line == nameLine {
				// we found our interface
				withinInterfaceCtx = true
			}

		} else {
			if strings.HasPrefix(line, "interface ") {
				// we found another interface - reset our state
				withinInterfaceCtx = false
				continue
			}
			if strings.HasPrefix(line, "static ip_address=") {
				return true, nil
			}
		}
	}

	return false, nil
}

// Get IP address with netmask
func getFullIP(ifaceName string) string {
	cmd := exec.Command("ip", "-oneline", "-family", "inet", "address", "show", ifaceName)
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	d, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return ""
	}

	fields := strings.Fields(string(d))
	if len(fields) < 4 {
		return ""
	}
	_, _, err = net.ParseCIDR(fields[3])
	if err != nil {
		return ""
	}

	return fields[3]
}

// Get gateway IP address
func getGatewayIP(ifaceName string) string {
	cmd := exec.Command("ip", "route", "show", "dev", ifaceName)
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	d, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return ""
	}

	fields := strings.Fields(string(d))
	if len(fields) < 3 || fields[0] != "default" {
		return ""
	}

	ip := net.ParseIP(fields[2])
	if ip == nil {
		return ""
	}

	return fields[2]
}

// Set a static IP for network interface
func setStaticIP(ifaceName string) error {
	ip := getFullIP(ifaceName)
	if len(ip) == 0 {
		return errors.New("Can't get IP address")
	}

	body, err := ioutil.ReadFile("/etc/dhcpcd.conf")
	if err != nil {
		return err
	}

	ip4, _, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}

	add := fmt.Sprintf("\ninterface %s\nstatic ip_address=%s\n",
		ifaceName, ip)
	body = append(body, []byte(add)...)

	gatewayIP := getGatewayIP(ifaceName)
	if len(gatewayIP) != 0 {
		add = fmt.Sprintf("static routers=%s\n",
			gatewayIP)
		body = append(body, []byte(add)...)
	}

	add = fmt.Sprintf("static domain_name_servers=%s\n\n",
		ip4)
	body = append(body, []byte(add)...)

	err = file.SafeWrite("/etc/dhcpcd.conf", body)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) handleDHCPAddStaticLease(w http.ResponseWriter, r *http.Request) {

	lj := staticLeaseJSON{}
	err := json.NewDecoder(r.Body).Decode(&lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	ip, _ := parseIPv4(lj.IP)
	if ip == nil {
		httpError(r, w, http.StatusBadRequest, "invalid IP")
		return
	}

	mac, _ := net.ParseMAC(lj.HWAddr)

	lease := Lease{
		IP:       ip,
		HWAddr:   mac,
		Hostname: lj.Hostname,
	}
	err = s.AddStaticLease(lease)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)
		return
	}
}

func (s *Server) handleDHCPRemoveStaticLease(w http.ResponseWriter, r *http.Request) {

	lj := staticLeaseJSON{}
	err := json.NewDecoder(r.Body).Decode(&lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	ip, _ := parseIPv4(lj.IP)
	if ip == nil {
		httpError(r, w, http.StatusBadRequest, "invalid IP")
		return
	}

	mac, _ := net.ParseMAC(lj.HWAddr)

	lease := Lease{
		IP:       ip,
		HWAddr:   mac,
		Hostname: lj.Hostname,
	}
	err = s.RemoveStaticLease(lease)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)
		return
	}
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	err := s.Stop()
	if err != nil {
		log.Error("DHCP: Stop: %s", err)
	}

	err = os.Remove(s.conf.DBFilePath)
	if err != nil && !os.IsNotExist(err) {
		log.Error("DHCP: os.Remove: %s: %s", s.conf.DBFilePath, err)
	}

	oldconf := s.conf
	s.conf = ServerConfig{}
	s.conf.LeaseDuration = 86400
	s.conf.ICMPTimeout = 1000
	s.conf.WorkDir = oldconf.WorkDir
	s.conf.HTTPRegister = oldconf.HTTPRegister
	s.conf.ConfigModified = oldconf.ConfigModified
	s.conf.DBFilePath = oldconf.DBFilePath
	s.conf.ConfigModified()
}

func (s *Server) registerHandlers() {
	s.conf.HTTPRegister("GET", "/control/dhcp/status", s.handleDHCPStatus)
	s.conf.HTTPRegister("GET", "/control/dhcp/interfaces", s.handleDHCPInterfaces)
	s.conf.HTTPRegister("POST", "/control/dhcp/set_config", s.handleDHCPSetConfig)
	s.conf.HTTPRegister("POST", "/control/dhcp/find_active_dhcp", s.handleDHCPFindActiveServer)
	s.conf.HTTPRegister("POST", "/control/dhcp/add_static_lease", s.handleDHCPAddStaticLease)
	s.conf.HTTPRegister("POST", "/control/dhcp/remove_static_lease", s.handleDHCPRemoveStaticLease)
	s.conf.HTTPRegister("POST", "/control/dhcp/reset", s.handleReset)
}
