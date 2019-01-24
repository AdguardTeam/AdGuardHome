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
	"github.com/hmage/golibs/log"
	"github.com/joomcode/errorx"
)

var dhcpServer = dhcpd.Server{}

func handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
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
	newconfig := dhcpd.ServerConfig{}
	err := json.NewDecoder(r.Body).Decode(&newconfig)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse new DHCP config json: %s", err)
		return
	}

	if newconfig.Enabled {
		err := dhcpServer.Start(&newconfig)
		if err != nil {
			httpError(w, http.StatusBadRequest, "Failed to start DHCP server: %s", err)
			return
		}
	}
	if !newconfig.Enabled {
		err := dhcpServer.Stop()
		if err != nil {
			log.Printf("failed to stop the DHCP server: %s", err)
		}
	}
	config.DHCP = newconfig
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleDHCPInterfaces(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{}

	ifaces, err := net.Interfaces()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get list of interfaces: %s", err)
		return
	}

	type responseInterface struct {
		Name         string   `json:"name"`
		MTU          int      `json:"mtu"`
		HardwareAddr string   `json:"hardware_address"`
		Addresses    []string `json:"ip_addresses"`
	}

	for i := range ifaces {
		if ifaces[i].Flags&net.FlagLoopback != 0 {
			// it's a loopback, skip it
			continue
		}
		if ifaces[i].Flags&net.FlagBroadcast == 0 {
			// this interface doesn't support broadcast, skip it
			continue
		}
		if ifaces[i].Flags&net.FlagPointToPoint != 0 {
			// this interface is ppp, don't do dhcp over it
			continue
		}
		iface := responseInterface{
			Name:         ifaces[i].Name,
			MTU:          ifaces[i].MTU,
			HardwareAddr: ifaces[i].HardwareAddr.String(),
		}
		addrs, err := ifaces[i].Addrs()
		if err != nil {
			httpError(w, http.StatusInternalServerError, "Failed to get addresses for interface %v: %s", ifaces[i].Name, err)
			return
		}
		for _, addr := range addrs {
			iface.Addresses = append(iface.Addresses, addr.String())
		}
		if len(iface.Addresses) == 0 {
			// this interface has no addresses, skip it
			continue
		}
		response[ifaces[i].Name] = iface
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Failed to marshal json with available interfaces: %s", err)
		return
	}
}

func handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to read request body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	interfaceName := strings.TrimSpace(string(body))
	if interfaceName == "" {
		errorText := fmt.Sprintf("empty interface name specified")
		log.Println(errorText)
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
	if config.DHCP.Enabled == false {
		// not enabled, don't do anything
		return nil
	}
	err := dhcpServer.Start(&config.DHCP)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start DHCP server")
	}
	return nil
}
