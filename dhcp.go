package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

var dhcpServer = dhcpd.Server{}

func handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	rawLeases := dhcpServer.Leases()
	leases := []map[string]string{}
	for i := range rawLeases {
		lease := map[string]string{
			"mac":      rawLeases[i].HWAddr.String(),
			"ip":       rawLeases[i].IP.String(),
			"hostname": rawLeases[i].Hostname,
			"expires":  rawLeases[i].Expiry.Format(time.RFC3339),
		}
		leases = append(leases, lease)

	}
	status := map[string]interface{}{
		"config": config.DHCP,
		"leases": leases,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal DHCP status json: %s", err)
		return
	}
}

func handleDHCPSetConfig(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	newconfig := dhcpd.ServerConfig{}
	err := json.NewDecoder(r.Body).Decode(&newconfig)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse new DHCP config json: %s", err)
		return
	}

	err = dhcpServer.Stop()
	if err != nil {
		log.Error("failed to stop the DHCP server: %s", err)
	}

	if newconfig.Enabled {

		staticIP, err := hasStaticIP(newconfig.InterfaceName)
		if !staticIP && err == nil {
			err = setStaticIP(newconfig.InterfaceName)
			if err != nil {
				httpError(w, http.StatusInternalServerError, "Failed to configure static IP: %s", err)
				return
			}
		}

		err = dhcpServer.Start(&newconfig)
		if err != nil {
			httpError(w, http.StatusBadRequest, "Failed to start DHCP server: %s", err)
			return
		}
	}

	config.DHCP = newconfig
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleDHCPInterfaces(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	response := map[string]interface{}{}

	ifaces, err := getValidNetInterfaces()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)
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
			httpError(w, http.StatusInternalServerError, "Failed to get addresses for interface %s: %s", iface.Name, err)
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
				httpError(w, http.StatusInternalServerError, "SHOULD NOT HAPPEN: got iface.Addrs() element %s that is not net.IPNet, it is %T", addr, addr)
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
		httpError(w, http.StatusInternalServerError, "Failed to marshal json with available interfaces: %s", err)
		return
	}
}

// Perform the following tasks:
// . Search for another DHCP server running
// . Check if a static IP is configured for the network interface
// Respond with results
func handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
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

	found, err := dhcpd.CheckIfOtherDHCPServersPresent(interfaceName)

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
		httpError(w, http.StatusInternalServerError, "Failed to marshal DHCP found json: %s", err)
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
	state := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if state == 1 && len(line) == 0 {
			// an empty line resets our state
			state = 0
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}
		line = strings.TrimSpace(line)

		if state == 0 {
			if line == nameLine {
				state = 1
			}

		} else if state == 1 {
			if strings.HasPrefix(line, "interface ") {
				state = 0
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

func startDHCPServer() error {
	if !config.DHCP.Enabled {
		// not enabled, don't do anything
		return nil
	}
	err := dhcpServer.Start(&config.DHCP)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start DHCP server")
	}
	return nil
}

func stopDHCPServer() error {
	if !config.DHCP.Enabled {
		return nil
	}

	if !dhcpServer.Enabled {
		return nil
	}

	err := dhcpServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop DHCP server")
	}

	return nil
}
