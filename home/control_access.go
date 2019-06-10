package home

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/AdguardTeam/golibs/log"
)

type accessListJSON struct {
	AllowedClients    []string `json:"allowed_clients"`
	DisallowedClients []string `json:"disallowed_clients"`
	BlockedHosts      []string `json:"blocked_hosts"`
}

func handleAccessList(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

	controlLock.Lock()
	j := accessListJSON{
		AllowedClients:    config.DNS.AllowedClients,
		DisallowedClients: config.DNS.DisallowedClients,
		BlockedHosts:      config.DNS.BlockedHosts,
	}
	controlLock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(j)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json.Encode: %s", err)
		return
	}
}

func checkIPCIDRArray(src []string) error {
	for _, s := range src {
		ip := net.ParseIP(s)
		if ip != nil {
			continue
		}

		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleAccessSet(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

	j := accessListJSON{}
	err := json.NewDecoder(r.Body).Decode(&j)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	err = checkIPCIDRArray(j.AllowedClients)
	if err == nil {
		err = checkIPCIDRArray(j.DisallowedClients)
	}
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	config.Lock()
	config.DNS.AllowedClients = j.AllowedClients
	config.DNS.DisallowedClients = j.DisallowedClients
	config.DNS.BlockedHosts = j.BlockedHosts
	config.Unlock()

	log.Tracef("Update access lists: %d, %d, %d",
		len(j.AllowedClients), len(j.DisallowedClients), len(j.BlockedHosts))

	err = writeAllConfigsAndReloadDNS()
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	returnOK(w)
}
