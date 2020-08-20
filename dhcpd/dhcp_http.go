package dhcpd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/AdguardTeam/golibs/jsonutil"
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

type v4ServerConfJSON struct {
	GatewayIP     string `json:"gateway_ip"`
	SubnetMask    string `json:"subnet_mask"`
	RangeStart    string `json:"range_start"`
	RangeEnd      string `json:"range_end"`
	LeaseDuration uint32 `json:"lease_duration"`
}

func v4ServerConfToJSON(c V4ServerConf) v4ServerConfJSON {
	return v4ServerConfJSON{
		GatewayIP:     c.GatewayIP,
		SubnetMask:    c.SubnetMask,
		RangeStart:    c.RangeStart,
		RangeEnd:      c.RangeEnd,
		LeaseDuration: c.LeaseDuration,
	}
}

func v4JSONToServerConf(j v4ServerConfJSON) V4ServerConf {
	return V4ServerConf{
		GatewayIP:     j.GatewayIP,
		SubnetMask:    j.SubnetMask,
		RangeStart:    j.RangeStart,
		RangeEnd:      j.RangeEnd,
		LeaseDuration: j.LeaseDuration,
	}
}

type v6ServerConfJSON struct {
	RangeStart    string `json:"range_start"`
	LeaseDuration uint32 `json:"lease_duration"`
}

func v6ServerConfToJSON(c V6ServerConf) v6ServerConfJSON {
	return v6ServerConfJSON{
		RangeStart:    c.RangeStart,
		LeaseDuration: c.LeaseDuration,
	}
}

func v6JSONToServerConf(j v6ServerConfJSON) V6ServerConf {
	return V6ServerConf{
		RangeStart:    j.RangeStart,
		LeaseDuration: j.LeaseDuration,
	}
}

func (s *Server) handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	leases := convertLeases(s.Leases(LeasesDynamic), true)
	staticLeases := convertLeases(s.Leases(LeasesStatic), false)

	v4conf := V4ServerConf{}
	s.srv4.WriteDiskConfig4(&v4conf)

	v6conf := V6ServerConf{}
	s.srv6.WriteDiskConfig6(&v6conf)

	status := map[string]interface{}{
		"enabled":        s.conf.Enabled,
		"interface_name": s.conf.InterfaceName,
		"v4":             v4ServerConfToJSON(v4conf),
		"v6":             v6ServerConfToJSON(v6conf),
		"leases":         leases,
		"static_leases":  staticLeases,
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
	Enabled       bool             `json:"enabled"`
	InterfaceName string           `json:"interface_name"`
	V4            v4ServerConfJSON `json:"v4"`
	V6            v6ServerConfJSON `json:"v6"`
}

func (s *Server) handleDHCPSetConfig(w http.ResponseWriter, r *http.Request) {
	newconfig := dhcpServerConfigJSON{}
	newconfig.Enabled = s.conf.Enabled
	newconfig.InterfaceName = s.conf.InterfaceName

	js, err := jsonutil.DecodeObject(&newconfig, r.Body)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Failed to parse new DHCP config json: %s", err)
		return
	}

	var s4 DHCPServer
	var s6 DHCPServer
	v4Enabled := false
	v6Enabled := false

	if js.Exists("v4") {
		v4conf := v4JSONToServerConf(newconfig.V4)
		v4conf.Enabled = newconfig.Enabled
		if len(v4conf.RangeStart) == 0 {
			v4conf.Enabled = false
		}
		v4Enabled = v4conf.Enabled
		v4conf.InterfaceName = newconfig.InterfaceName

		c4 := V4ServerConf{}
		s.srv4.WriteDiskConfig4(&c4)
		v4conf.notify = c4.notify
		v4conf.ICMPTimeout = c4.ICMPTimeout

		s4, err = v4Create(v4conf)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "Invalid DHCPv4 configuration: %s", err)
			return
		}
	}

	if js.Exists("v6") {
		v6conf := v6JSONToServerConf(newconfig.V6)
		v6conf.Enabled = newconfig.Enabled
		if len(v6conf.RangeStart) == 0 {
			v6conf.Enabled = false
		}
		v6Enabled = v6conf.Enabled
		v6conf.InterfaceName = newconfig.InterfaceName
		v6conf.notify = s.onNotify
		s6, err = v6Create(v6conf)
		if s6 == nil {
			httpError(r, w, http.StatusBadRequest, "Invalid DHCPv6 configuration: %s", err)
			return
		}
	}

	if newconfig.Enabled && !v4Enabled && !v6Enabled {
		httpError(r, w, http.StatusBadRequest, "DHCPv4 or DHCPv6 configuration must be complete")
		return
	}

	s.Stop()

	if js.Exists("enabled") {
		s.conf.Enabled = newconfig.Enabled
	}

	if js.Exists("interface_name") {
		s.conf.InterfaceName = newconfig.InterfaceName
	}

	if s4 != nil {
		s.srv4 = s4
	}
	if s6 != nil {
		s.srv6 = s6
	}
	s.conf.ConfigModified()
	s.dbLoad()

	if s.conf.Enabled {
		staticIP, err := HasStaticIP(newconfig.InterfaceName)
		if !staticIP && err == nil {
			err = SetStaticIP(newconfig.InterfaceName)
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

type netInterfaceJSON struct {
	Name         string   `json:"name"`
	GatewayIP    string   `json:"gateway_ip"`
	HardwareAddr string   `json:"hardware_address"`
	Addrs4       []string `json:"ipv4_addresses"`
	Addrs6       []string `json:"ipv6_addresses"`
	Flags        string   `json:"flags"`
}

func (s *Server) handleDHCPInterfaces(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{}

	ifaces, err := util.GetValidNetInterfaces()
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

		jsonIface := netInterfaceJSON{
			Name:         iface.Name,
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
			if ipnet.IP.To4() != nil {
				jsonIface.Addrs4 = append(jsonIface.Addrs4, ipnet.IP.String())
			} else {
				jsonIface.Addrs6 = append(jsonIface.Addrs6, ipnet.IP.String())
			}
		}
		if len(jsonIface.Addrs4)+len(jsonIface.Addrs6) != 0 {
			jsonIface.GatewayIP = getGatewayIP(iface.Name)
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

	found4, err4 := CheckIfOtherDHCPServersPresentV4(interfaceName)

	staticIP := map[string]interface{}{}
	isStaticIP, err := HasStaticIP(interfaceName)
	staticIPStatus := "yes"
	if err != nil {
		staticIPStatus = "error"
		staticIP["error"] = err.Error()
	} else if !isStaticIP {
		staticIPStatus = "no"
		staticIP["ip"] = util.GetSubnet(interfaceName)
	}
	staticIP["static"] = staticIPStatus

	v4 := map[string]interface{}{}
	othSrv := map[string]interface{}{}
	foundVal := "no"
	if found4 {
		foundVal = "yes"
	} else if err != nil {
		foundVal = "error"
		othSrv["error"] = err4.Error()
	}
	othSrv["found"] = foundVal
	v4["other_server"] = othSrv
	v4["static_ip"] = staticIP

	found6, err6 := CheckIfOtherDHCPServersPresentV6(interfaceName)

	v6 := map[string]interface{}{}
	othSrv = map[string]interface{}{}
	foundVal = "no"
	if found6 {
		foundVal = "yes"
	} else if err6 != nil {
		foundVal = "error"
		othSrv["error"] = err6.Error()
	}
	othSrv["found"] = foundVal
	v6["other_server"] = othSrv

	result := map[string]interface{}{}
	result["v4"] = v4
	result["v6"] = v6

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Failed to marshal DHCP found json: %s", err)
		return
	}
}

func (s *Server) handleDHCPAddStaticLease(w http.ResponseWriter, r *http.Request) {

	lj := staticLeaseJSON{}
	err := json.NewDecoder(r.Body).Decode(&lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	ip := net.ParseIP(lj.IP)
	if ip != nil && ip.To4() == nil {
		mac, err := net.ParseMAC(lj.HWAddr)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "invalid MAC")
			return
		}

		lease := Lease{
			IP:     ip,
			HWAddr: mac,
		}

		err = s.srv6.AddStaticLease(lease)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "%s", err)
			return
		}
		return
	}

	ip, _ = parseIPv4(lj.IP)
	if ip == nil {
		httpError(r, w, http.StatusBadRequest, "invalid IP")
		return
	}

	mac, err := net.ParseMAC(lj.HWAddr)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "invalid MAC")
		return
	}

	lease := Lease{
		IP:       ip,
		HWAddr:   mac,
		Hostname: lj.Hostname,
	}
	err = s.srv4.AddStaticLease(lease)
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

	ip := net.ParseIP(lj.IP)
	if ip != nil && ip.To4() == nil {
		mac, err := net.ParseMAC(lj.HWAddr)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "invalid MAC")
			return
		}

		lease := Lease{
			IP:     ip,
			HWAddr: mac,
		}

		err = s.srv6.RemoveStaticLease(lease)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "%s", err)
			return
		}
		return
	}

	ip, _ = parseIPv4(lj.IP)
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
	err = s.srv4.RemoveStaticLease(lease)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)
		return
	}
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	s.Stop()

	err := os.Remove(s.conf.DBFilePath)
	if err != nil && !os.IsNotExist(err) {
		log.Error("DHCP: os.Remove: %s: %s", s.conf.DBFilePath, err)
	}

	oldconf := s.conf
	s.conf = ServerConfig{}
	s.conf.WorkDir = oldconf.WorkDir
	s.conf.HTTPRegister = oldconf.HTTPRegister
	s.conf.ConfigModified = oldconf.ConfigModified
	s.conf.DBFilePath = oldconf.DBFilePath

	v4conf := V4ServerConf{}
	v4conf.ICMPTimeout = 1000
	v4conf.notify = s.onNotify
	s.srv4, _ = v4Create(v4conf)

	v6conf := V6ServerConf{}
	v6conf.notify = s.onNotify
	s.srv6, _ = v6Create(v6conf)

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
