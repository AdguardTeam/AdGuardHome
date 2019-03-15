package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
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
		err := dhcpServer.Start(&newconfig)
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
	result := map[string]interface{}{}
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["found"] = found
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Failed to marshal DHCP found json: %s", err)
		return
	}
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
