package main

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
)

var dhcpServer = dhcpd.Server{}

func handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"config": config.DHCP,
		"leases": dhcpServer.Leases(),
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
		dhcpServer.Stop()
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

	type address struct {
		IP      string
		Netmask string
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

// implement
func handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	found, err := dhcpd.CheckIfOtherDHCPServersPresent(config.DHCP.InterfaceName)
	result := map[string]interface{}{
		"found": found,
	}
	if err != nil {
		result["found"] = false
		result["error"] = err
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Failed to marshal DHCP found json: %s", err)
		return
	}
}
