package dhcpd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/log"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info("DHCP: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

type v4ServerConfJSON struct {
	GatewayIP     net.IP `json:"gateway_ip"`
	SubnetMask    net.IP `json:"subnet_mask"`
	RangeStart    net.IP `json:"range_start"`
	RangeEnd      net.IP `json:"range_end"`
	LeaseDuration uint32 `json:"lease_duration"`
}

func v4JSONToServerConf(j *v4ServerConfJSON) V4ServerConf {
	if j == nil {
		return V4ServerConf{}
	}

	return V4ServerConf{
		GatewayIP:     j.GatewayIP,
		SubnetMask:    j.SubnetMask,
		RangeStart:    j.RangeStart,
		RangeEnd:      j.RangeEnd,
		LeaseDuration: j.LeaseDuration,
	}
}

type v6ServerConfJSON struct {
	RangeStart    net.IP `json:"range_start"`
	LeaseDuration uint32 `json:"lease_duration"`
}

func v6JSONToServerConf(j *v6ServerConfJSON) V6ServerConf {
	if j == nil {
		return V6ServerConf{}
	}

	return V6ServerConf{
		RangeStart:    j.RangeStart,
		LeaseDuration: j.LeaseDuration,
	}
}

// dhcpStatusResponse is the response for /control/dhcp/status endpoint.
type dhcpStatusResponse struct {
	Enabled      bool         `json:"enabled"`
	IfaceName    string       `json:"interface_name"`
	V4           V4ServerConf `json:"v4"`
	V6           V6ServerConf `json:"v6"`
	Leases       []Lease      `json:"leases"`
	StaticLeases []Lease      `json:"static_leases"`
}

func (s *Server) handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	status := &dhcpStatusResponse{
		Enabled:   s.conf.Enabled,
		IfaceName: s.conf.InterfaceName,
		V4:        V4ServerConf{},
		V6:        V6ServerConf{},
	}

	s.srv4.WriteDiskConfig4(&status.V4)
	s.srv6.WriteDiskConfig6(&status.V6)

	status.Leases = s.Leases(LeasesDynamic)
	status.StaticLeases = s.Leases(LeasesStatic)

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal DHCP status json: %s", err)
		return
	}
}

func (s *Server) enableDHCP(ifaceName string) (code int, err error) {
	var hasStaticIP bool
	hasStaticIP, err = aghnet.IfaceHasStaticIP(ifaceName)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			// ErrPermission may happen here on Linux systems where
			// AdGuard Home is installed using Snap.  That doesn't
			// necessarily mean that the machine doesn't have
			// a static IP, so we can assume that it has and go on.
			// If the machine doesn't, we'll get an error later.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/2667.
			//
			// TODO(a.garipov): I was thinking about moving this
			// into IfaceHasStaticIP, but then we wouldn't be able
			// to log it.  Think about it more.
			log.Info("error while checking static ip: %s; "+
				"assuming machine has static ip and going on", err)
			hasStaticIP = true
		} else if errors.Is(err, aghnet.ErrNoStaticIPInfo) {
			// Couldn't obtain a definitive answer.  Assume static
			// IP an go on.
			log.Info("can't check for static ip; " +
				"assuming machine has static ip and going on")
			hasStaticIP = true
		} else {
			err = fmt.Errorf("checking static ip: %w", err)

			return http.StatusInternalServerError, err
		}
	}

	if !hasStaticIP {
		err = aghnet.IfaceSetStaticIP(ifaceName)
		if err != nil {
			err = fmt.Errorf("setting static ip: %w", err)

			return http.StatusInternalServerError, err
		}
	}

	err = s.Start()
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("starting dhcp server: %w", err)
	}

	return 0, nil
}

type dhcpServerConfigJSON struct {
	V4            *v4ServerConfJSON `json:"v4"`
	V6            *v6ServerConfJSON `json:"v6"`
	InterfaceName string            `json:"interface_name"`
	Enabled       nullBool          `json:"enabled"`
}

func (s *Server) handleDHCPSetConfig(w http.ResponseWriter, r *http.Request) {
	conf := dhcpServerConfigJSON{}
	conf.Enabled = boolToNullBool(s.conf.Enabled)
	conf.InterfaceName = s.conf.InterfaceName

	err := json.NewDecoder(r.Body).Decode(&conf)
	if err != nil {
		httpError(r, w, http.StatusBadRequest,
			"failed to parse new dhcp config json: %s", err)

		return
	}

	var s4 DHCPServer
	var s6 DHCPServer
	v4Enabled := false
	v6Enabled := false

	if conf.V4 != nil {
		v4Conf := v4JSONToServerConf(conf.V4)
		v4Conf.Enabled = conf.Enabled == nbTrue
		if len(v4Conf.RangeStart) == 0 {
			v4Conf.Enabled = false
		}

		v4Enabled = v4Conf.Enabled
		v4Conf.InterfaceName = conf.InterfaceName

		c4 := V4ServerConf{}
		s.srv4.WriteDiskConfig4(&c4)
		v4Conf.notify = c4.notify
		v4Conf.ICMPTimeout = c4.ICMPTimeout

		s4, err = v4Create(v4Conf)
		if err != nil {
			httpError(r, w, http.StatusBadRequest,
				"invalid dhcpv4 configuration: %s", err)

			return
		}
	}

	if conf.V6 != nil {
		v6Conf := v6JSONToServerConf(conf.V6)
		v6Conf.Enabled = conf.Enabled == nbTrue
		if len(v6Conf.RangeStart) == 0 {
			v6Conf.Enabled = false
		}

		// Don't overwrite the RA/SLAAC settings from the config file.
		//
		// TODO(a.garipov): Perhaps include them into the request to
		// allow changing them from the HTTP API?
		v6Conf.RASLAACOnly = s.conf.Conf6.RASLAACOnly
		v6Conf.RAAllowSLAAC = s.conf.Conf6.RAAllowSLAAC

		v6Enabled = v6Conf.Enabled
		v6Conf.InterfaceName = conf.InterfaceName
		v6Conf.notify = s.onNotify

		s6, err = v6Create(v6Conf)
		if err != nil {
			httpError(r, w, http.StatusBadRequest,
				"invalid dhcpv6 configuration: %s", err)

			return
		}
	}

	if conf.Enabled == nbTrue && !v4Enabled && !v6Enabled {
		httpError(r, w, http.StatusBadRequest,
			"dhcpv4 or dhcpv6 configuration must be complete")

		return
	}

	s.Stop()

	if conf.Enabled != nbNull {
		s.conf.Enabled = conf.Enabled == nbTrue
	}

	if conf.InterfaceName != "" {
		s.conf.InterfaceName = conf.InterfaceName
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
		var code int
		code, err = s.enableDHCP(conf.InterfaceName)
		if err != nil {
			httpError(r, w, code, "enabling dhcp: %s", err)

			return
		}
	}
}

type netInterfaceJSON struct {
	Name         string   `json:"name"`
	GatewayIP    net.IP   `json:"gateway_ip"`
	HardwareAddr string   `json:"hardware_address"`
	Addrs4       []net.IP `json:"ipv4_addresses"`
	Addrs6       []net.IP `json:"ipv6_addresses"`
	Flags        string   `json:"flags"`
}

func (s *Server) handleDHCPInterfaces(w http.ResponseWriter, r *http.Request) {
	response := map[string]netInterfaceJSON{}

	ifaces, err := net.Interfaces()
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

		var addrs []net.Addr
		addrs, err = iface.Addrs()
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
				jsonIface.Addrs4 = append(jsonIface.Addrs4, ipnet.IP)
			} else {
				jsonIface.Addrs6 = append(jsonIface.Addrs6, ipnet.IP)
			}
		}
		if len(jsonIface.Addrs4)+len(jsonIface.Addrs6) != 0 {
			jsonIface.GatewayIP = aghnet.GatewayIP(iface.Name)
			response[iface.Name] = jsonIface
		}
	}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Failed to marshal json with available interfaces: %s", err)
		return
	}
}

// dhcpSearchOtherResult contains information about other DHCP server for
// specific network interface.
type dhcpSearchOtherResult struct {
	Found string `json:"found,omitempty"`
	Error string `json:"error,omitempty"`
}

// dhcpStaticIPStatus contains information about static IP address for DHCP
// server.
type dhcpStaticIPStatus struct {
	Static string `json:"static"`
	IP     string `json:"ip,omitempty"`
	Error  string `json:"error,omitempty"`
}

// dhcpSearchV4Result contains information about DHCPv4 server for specific
// network interface.
type dhcpSearchV4Result struct {
	OtherServer dhcpSearchOtherResult `json:"other_server"`
	StaticIP    dhcpStaticIPStatus    `json:"static_ip"`
}

// dhcpSearchV6Result contains information about DHCPv6 server for specific
// network interface.
type dhcpSearchV6Result struct {
	OtherServer dhcpSearchOtherResult `json:"other_server"`
}

// dhcpSearchResult is a response for /control/dhcp/find_active_dhcp endpoint.
type dhcpSearchResult struct {
	V4 dhcpSearchV4Result `json:"v4"`
	V6 dhcpSearchV6Result `json:"v6"`
}

// Perform the following tasks:
// . Search for another DHCP server running
// . Check if a static IP is configured for the network interface
// Respond with results
func (s *Server) handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	// This use of ReadAll is safe, because request's body is now limited.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to read request body: %s", err)
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	interfaceName := strings.TrimSpace(string(body))
	if interfaceName == "" {
		msg := "empty interface name specified"
		log.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	result := dhcpSearchResult{
		V4: dhcpSearchV4Result{
			OtherServer: dhcpSearchOtherResult{},
			StaticIP:    dhcpStaticIPStatus{},
		},
		V6: dhcpSearchV6Result{
			OtherServer: dhcpSearchOtherResult{},
		},
	}

	found4, err4 := CheckIfOtherDHCPServersPresentV4(interfaceName)

	isStaticIP, err := aghnet.IfaceHasStaticIP(interfaceName)
	if err != nil {
		result.V4.StaticIP.Static = "error"
		result.V4.StaticIP.Error = err.Error()
	} else if !isStaticIP {
		result.V4.StaticIP.Static = "no"
		result.V4.StaticIP.IP = aghnet.GetSubnet(interfaceName).String()
	}

	if found4 {
		result.V4.OtherServer.Found = "yes"
	} else if err4 != nil {
		result.V4.OtherServer.Found = "error"
		result.V4.OtherServer.Error = err4.Error()
	}

	found6, err6 := CheckIfOtherDHCPServersPresentV6(interfaceName)

	if found6 {
		result.V6.OtherServer.Found = "yes"
	} else if err6 != nil {
		result.V6.OtherServer.Found = "error"
		result.V6.OtherServer.Error = err6.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Failed to marshal DHCP found json: %s", err)
		return
	}
}

func (s *Server) handleDHCPAddStaticLease(w http.ResponseWriter, r *http.Request) {
	lj := Lease{}
	err := json.NewDecoder(r.Body).Decode(&lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	if lj.IP == nil {
		httpError(r, w, http.StatusBadRequest, "invalid IP")

		return
	}

	ip4 := lj.IP.To4()

	if ip4 == nil {
		lj.IP = lj.IP.To16()

		err = s.srv6.AddStaticLease(lj)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "%s", err)
		}

		return
	}

	lj.IP = ip4
	err = s.srv4.AddStaticLease(lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)

		return
	}
}

func (s *Server) handleDHCPRemoveStaticLease(w http.ResponseWriter, r *http.Request) {
	lj := Lease{}
	err := json.NewDecoder(r.Body).Decode(&lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)

		return
	}

	if lj.IP == nil {
		httpError(r, w, http.StatusBadRequest, "invalid IP")

		return
	}

	ip4 := lj.IP.To4()

	if ip4 == nil {
		lj.IP = lj.IP.To16()

		err = s.srv6.RemoveStaticLease(lj)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "%s", err)
		}

		return
	}

	lj.IP = ip4
	err = s.srv4.RemoveStaticLease(lj)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)

		return
	}
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	s.Stop()

	err := os.Remove(s.conf.DBFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("dhcp: removing %q: %s", s.conf.DBFilePath, err)
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
	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/status", s.handleDHCPStatus)
	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/interfaces", s.handleDHCPInterfaces)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/set_config", s.handleDHCPSetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/find_active_dhcp", s.handleDHCPFindActiveServer)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/add_static_lease", s.handleDHCPAddStaticLease)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/remove_static_lease", s.handleDHCPRemoveStaticLease)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/reset", s.handleReset)
}

// jsonError is a generic JSON error response.
//
// TODO(a.garipov): Merge together with the implementations in .../home and
// other packages after refactoring the web handler registering.
type jsonError struct {
	// Message is the error message, an opaque string.
	Message string `json:"message"`
}

// notImplemented returns a handler that replies to any request with an HTTP 501
// Not Implemented status and a JSON error with the provided message msg.
//
// TODO(a.garipov): Either take the logger from the server after we've
// refactored logging or make this not a method of *Server.
func (s *Server) notImplemented(msg string) (f func(http.ResponseWriter, *http.Request)) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)

		err := json.NewEncoder(w).Encode(&jsonError{
			Message: msg,
		})
		if err != nil {
			log.Debug("writing 501 json response: %s", err)
		}
	}
}

func (s *Server) registerNotImplementedHandlers() {
	h := s.notImplemented("dhcp is not supported on windows")

	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/status", h)
	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/interfaces", h)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/set_config", h)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/find_active_dhcp", h)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/add_static_lease", h)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/remove_static_lease", h)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/reset", h)
}
