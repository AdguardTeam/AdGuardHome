package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
)

func handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"config": config.DHCP.Config,
		"leases": config.DHCP.Leases,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal DHCP status json: %s", err)
		return
	}
}

func handleDHCPSetConfig(w http.ResponseWriter, r *http.Request) {
	newconfig := dhcpConfig{}
	err := json.NewDecoder(r.Body).Decode(&newconfig)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse new DHCP config json: %s", err)
		return
	}

	config.DHCP.Config = newconfig
}

// TODO: implement
func handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	found := map[string]bool{
		"found": rand.Intn(2) == 1,
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(found)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to marshal DHCP found json: %s", err)
		return
	}
}
