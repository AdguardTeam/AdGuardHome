package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/AdguardTeam/golibs/log"
)

// Client information
type Client struct {
	IP   string
	Name string
	//Source source // Hosts file / User settings / DHCP
}

type clientJSON struct {
	IP   string `json:"ip"`
	Name string `json:"name"`
}

var clients []Client
var clientsFilled bool

// Parse system 'hosts' file and fill clients array
func fillClientInfo() {
	hostsFn := "/etc/hosts"
	if runtime.GOOS == "windows" {
		hostsFn = os.ExpandEnv("$SystemRoot\\system32\\drivers\\etc\\hosts")
	}

	d, e := ioutil.ReadFile(hostsFn)
	if e != nil {
		log.Info("Can't read file %s: %v", hostsFn, e)
		return
	}

	lines := strings.Split(string(d), "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if len(ln) == 0 || ln[0] == '#' {
			continue
		}

		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}

		var c Client
		c.IP = fields[0]
		c.Name = fields[1]
		clients = append(clients, c)
		log.Tracef("%s -> %s", c.IP, c.Name)
	}

	log.Info("Added %d client aliases from %s", len(clients), hostsFn)
	clientsFilled = true
}

// respond with information about configured clients
func handleGetClients(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

	if !clientsFilled {
		fillClientInfo()
	}

	data := []clientJSON{}
	for _, c := range clients {
		cj := clientJSON{
			IP:   c.IP,
			Name: c.Name,
		}
		data = append(data, cj)
	}
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w).Encode(data)
	if e != nil {
		httpError(w, http.StatusInternalServerError, "Failed to encode to json: %v", e)
		return
	}
}

// RegisterClientsHandlers registers HTTP handlers
func RegisterClientsHandlers() {
	http.HandleFunc("/control/clients", postInstall(optionalAuth(ensureGET(handleGetClients))))
}
