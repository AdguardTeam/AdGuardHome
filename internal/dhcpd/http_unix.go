//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
)

type v4ServerConfJSON struct {
	GatewayIP     netip.Addr `json:"gateway_ip"`
	SubnetMask    netip.Addr `json:"subnet_mask"`
	RangeStart    netip.Addr `json:"range_start"`
	RangeEnd      netip.Addr `json:"range_end"`
	LeaseDuration uint32     `json:"lease_duration"`
}

func (j *v4ServerConfJSON) toServerConf() *V4ServerConf {
	if j == nil {
		return &V4ServerConf{}
	}

	return &V4ServerConf{
		GatewayIP:     j.GatewayIP,
		SubnetMask:    j.SubnetMask,
		RangeStart:    j.RangeStart,
		RangeEnd:      j.RangeEnd,
		LeaseDuration: j.LeaseDuration,
	}
}

type v6ServerConfJSON struct {
	RangeStart    netip.Addr `json:"range_start"`
	LeaseDuration uint32     `json:"lease_duration"`
}

func v6JSONToServerConf(j *v6ServerConfJSON) V6ServerConf {
	if j == nil {
		return V6ServerConf{}
	}

	return V6ServerConf{
		RangeStart:    j.RangeStart.AsSlice(),
		LeaseDuration: j.LeaseDuration,
	}
}

// dhcpStatusResponse is the response for /control/dhcp/status endpoint.
type dhcpStatusResponse struct {
	IfaceName    string          `json:"interface_name"`
	V4           V4ServerConf    `json:"v4"`
	V6           V6ServerConf    `json:"v6"`
	Leases       []*leaseDynamic `json:"leases"`
	StaticLeases []*leaseStatic  `json:"static_leases"`
	Enabled      bool            `json:"enabled"`
}

// leaseStatic is the JSON form of static DHCP lease.
type leaseStatic struct {
	HWAddr   string     `json:"mac"`
	IP       netip.Addr `json:"ip"`
	Hostname string     `json:"hostname"`
}

// leasesToStatic converts list of leases to their JSON form.
func leasesToStatic(leases []*dhcpsvc.Lease) (static []*leaseStatic) {
	static = make([]*leaseStatic, len(leases))

	for i, l := range leases {
		static[i] = &leaseStatic{
			HWAddr:   l.HWAddr.String(),
			IP:       l.IP,
			Hostname: l.Hostname,
		}
	}

	return static
}

// toLease converts leaseStatic to Lease or returns error.
func (l *leaseStatic) toLease() (lease *dhcpsvc.Lease, err error) {
	addr, err := net.ParseMAC(l.HWAddr)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse MAC address: %w", err)
	}

	return &dhcpsvc.Lease{
		HWAddr:   addr,
		IP:       l.IP,
		Hostname: l.Hostname,
		IsStatic: true,
	}, nil
}

// leaseDynamic is the JSON form of dynamic DHCP lease.
type leaseDynamic struct {
	HWAddr   string     `json:"mac"`
	IP       netip.Addr `json:"ip"`
	Hostname string     `json:"hostname"`
	Expiry   string     `json:"expires"`
}

// leasesToDynamic converts list of leases to their JSON form.
func leasesToDynamic(leases []*dhcpsvc.Lease) (dynamic []*leaseDynamic) {
	dynamic = make([]*leaseDynamic, len(leases))

	for i, l := range leases {
		dynamic[i] = &leaseDynamic{
			HWAddr:   l.HWAddr.String(),
			IP:       l.IP,
			Hostname: l.Hostname,
			// The front-end is waiting for RFC 3999 format of the time
			// value.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/2692.
			Expiry: l.Expiry.Format(time.RFC3339),
		}
	}

	return dynamic
}

func (s *server) handleDHCPStatus(w http.ResponseWriter, r *http.Request) {
	status := &dhcpStatusResponse{
		Enabled:   s.conf.Enabled,
		IfaceName: s.conf.InterfaceName,
		V4:        V4ServerConf{},
		V6:        V6ServerConf{},
	}

	s.srv4.WriteDiskConfig4(&status.V4)
	s.srv6.WriteDiskConfig6(&status.V6)

	leases := s.Leases()
	slices.SortFunc(leases, func(a, b *dhcpsvc.Lease) (res int) {
		if a.IsStatic == b.IsStatic {
			return 0
		} else if a.IsStatic {
			return -1
		} else {
			return 1
		}
	})

	dynamicIdx := slices.IndexFunc(leases, func(l *dhcpsvc.Lease) (ok bool) {
		return !l.IsStatic
	})

	if dynamicIdx == -1 {
		dynamicIdx = len(leases)
	}

	status.Leases = leasesToDynamic(leases[dynamicIdx:])
	status.StaticLeases = leasesToStatic(leases[:dynamicIdx])

	aghhttp.WriteJSONResponseOK(w, r, status)
}

func (s *server) enableDHCP(ifaceName string) (code int, err error) {
	var hasStaticIP bool
	hasStaticIP, err = aghnet.IfaceHasStaticIP(ifaceName)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			// ErrPermission may happen here on Linux systems where AdGuard Home
			// is installed using Snap.  That doesn't necessarily mean that the
			// machine doesn't have a static IP, so we can assume that it has
			// and go on.  If the machine doesn't, we'll get an error later.
			//
			// See https://github.com/AdguardTeam/AdGuardHome/issues/2667.
			//
			// TODO(a.garipov): I was thinking about moving this into
			// IfaceHasStaticIP, but then we wouldn't be able to log it.  Think
			// about it more.
			log.Info("error while checking static ip: %s; "+
				"assuming machine has static ip and going on", err)
			hasStaticIP = true
		} else if errors.Is(err, aghnet.ErrNoStaticIPInfo) {
			// Couldn't obtain a definitive answer.  Assume static IP an go on.
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
	Enabled       aghalg.NullBool   `json:"enabled"`
}

func (s *server) handleDHCPSetConfigV4(
	conf *dhcpServerConfigJSON,
) (srv DHCPServer, enabled bool, err error) {
	if conf.V4 == nil {
		return nil, false, nil
	}

	v4Conf := conf.V4.toServerConf()
	v4Conf.Enabled = conf.Enabled == aghalg.NBTrue
	if !v4Conf.RangeStart.IsValid() {
		v4Conf.Enabled = false
	}

	v4Conf.InterfaceName = conf.InterfaceName

	// Set the default values for the fields not configurable via web API.
	c4 := &V4ServerConf{
		notify:      s.onNotify,
		ICMPTimeout: s.conf.Conf4.ICMPTimeout,
		Options:     s.conf.Conf4.Options,
	}

	s.srv4.WriteDiskConfig4(c4)
	v4Conf.notify = c4.notify
	v4Conf.ICMPTimeout = c4.ICMPTimeout
	v4Conf.Options = c4.Options

	srv4, err := v4Create(v4Conf)

	return srv4, srv4.enabled(), err
}

func (s *server) handleDHCPSetConfigV6(
	conf *dhcpServerConfigJSON,
) (srv6 DHCPServer, enabled bool, err error) {
	if conf.V6 == nil {
		return nil, false, nil
	}

	v6Conf := v6JSONToServerConf(conf.V6)
	v6Conf.Enabled = conf.Enabled == aghalg.NBTrue
	if len(v6Conf.RangeStart) == 0 {
		v6Conf.Enabled = false
	}

	// Don't overwrite the RA/SLAAC settings from the config file.
	//
	// TODO(a.garipov): Perhaps include them into the request to allow
	// changing them from the HTTP API?
	v6Conf.RASLAACOnly = s.conf.Conf6.RASLAACOnly
	v6Conf.RAAllowSLAAC = s.conf.Conf6.RAAllowSLAAC

	enabled = v6Conf.Enabled
	v6Conf.InterfaceName = conf.InterfaceName
	v6Conf.notify = s.onNotify

	srv6, err = v6Create(v6Conf)

	return srv6, enabled, err
}

// createServers returns DHCPv4 and DHCPv6 servers created from the provided
// configuration conf.
func (s *server) createServers(conf *dhcpServerConfigJSON) (srv4, srv6 DHCPServer, err error) {
	srv4, v4Enabled, err := s.handleDHCPSetConfigV4(conf)
	if err != nil {
		return nil, nil, fmt.Errorf("bad dhcpv4 configuration: %w", err)
	}

	srv6, v6Enabled, err := s.handleDHCPSetConfigV6(conf)
	if err != nil {
		return nil, nil, fmt.Errorf("bad dhcpv6 configuration: %w", err)
	}

	if conf.Enabled == aghalg.NBTrue && !v4Enabled && !v6Enabled {
		return nil, nil, fmt.Errorf("dhcpv4 or dhcpv6 configuration must be complete")
	}

	return srv4, srv6, nil
}

// handleDHCPSetConfig is the handler for the POST /control/dhcp/set_config
// HTTP API.
func (s *server) handleDHCPSetConfig(w http.ResponseWriter, r *http.Request) {
	conf := &dhcpServerConfigJSON{}
	conf.Enabled = aghalg.BoolToNullBool(s.conf.Enabled)
	conf.InterfaceName = s.conf.InterfaceName

	err := json.NewDecoder(r.Body).Decode(conf)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "failed to parse new dhcp config json: %s", err)

		return
	}

	srv4, srv6, err := s.createServers(conf)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	err = s.Stop()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "stopping dhcp: %s", err)

		return
	}

	s.setConfFromJSON(conf, srv4, srv6)
	s.conf.ConfigModified()

	err = s.dbLoad()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "loading leases db: %s", err)

		return
	}

	if s.conf.Enabled {
		var code int
		code, err = s.enableDHCP(conf.InterfaceName)
		if err != nil {
			aghhttp.Error(r, w, code, "enabling dhcp: %s", err)
		}
	}
}

// setConfFromJSON sets configuration parameters in s from the new configuration
// decoded from JSON.
func (s *server) setConfFromJSON(conf *dhcpServerConfigJSON, srv4, srv6 DHCPServer) {
	if conf.Enabled != aghalg.NBNull {
		s.conf.Enabled = conf.Enabled == aghalg.NBTrue
	}

	if conf.InterfaceName != "" {
		s.conf.InterfaceName = conf.InterfaceName
	}

	if srv4 != nil {
		s.srv4 = srv4
	}

	if srv6 != nil {
		s.srv6 = srv6
	}
}

type netInterfaceJSON struct {
	Name         string       `json:"name"`
	HardwareAddr string       `json:"hardware_address"`
	Flags        string       `json:"flags"`
	GatewayIP    netip.Addr   `json:"gateway_ip"`
	Addrs4       []netip.Addr `json:"ipv4_addresses"`
	Addrs6       []netip.Addr `json:"ipv6_addresses"`
}

// handleDHCPInterfaces is the handler for the GET /control/dhcp/interfaces
// HTTP API.
func (s *server) handleDHCPInterfaces(w http.ResponseWriter, r *http.Request) {
	resp := map[string]*netInterfaceJSON{}

	ifaces, err := net.Interfaces()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)

		return
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			// It's a loopback, skip it.
			continue
		}

		if iface.Flags&net.FlagBroadcast == 0 {
			// This interface doesn't support broadcast, skip it.
			continue
		}

		jsonIface, iErr := newNetInterfaceJSON(iface)
		if iErr != nil {
			aghhttp.Error(r, w, http.StatusInternalServerError, "%s", iErr)

			return
		}

		if jsonIface != nil {
			resp[iface.Name] = jsonIface
		}
	}

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// newNetInterfaceJSON creates a JSON object from a [net.Interface] iface.
func newNetInterfaceJSON(iface net.Interface) (out *netInterfaceJSON, err error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get addresses for interface %s: %w",
			iface.Name,
			err,
		)
	}

	out = &netInterfaceJSON{
		Name:         iface.Name,
		HardwareAddr: iface.HardwareAddr.String(),
	}

	if iface.Flags != 0 {
		out.Flags = iface.Flags.String()
	}

	// We don't want link-local addresses in JSON, so skip them.
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			// Not an IPNet, should not happen.
			return nil, fmt.Errorf("got iface.Addrs() element %[1]s that is not"+
				" net.IPNet, it is %[1]T", addr)
		}

		// Ignore link-local.
		//
		// TODO(e.burkov):  Try to listen DHCP on LLA as well.
		if ipNet.IP.IsLinkLocalUnicast() {
			continue
		}

		vAddr, iErr := netutil.IPToAddrNoMapped(ipNet.IP)
		if iErr != nil {
			// Not an IPNet, should not happen.
			return nil, fmt.Errorf("failed to convert IP address %[1]s: %w", addr, iErr)
		}

		if vAddr.Is4() {
			out.Addrs4 = append(out.Addrs4, vAddr)
		} else {
			out.Addrs6 = append(out.Addrs6, vAddr)
		}
	}

	if len(out.Addrs4)+len(out.Addrs6) == 0 {
		return nil, nil
	}

	out.GatewayIP = aghnet.GatewayIP(iface.Name)

	return out, nil
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

// findActiveServerReq is the JSON structure for the request to find active DHCP
// servers.
type findActiveServerReq struct {
	Interface string `json:"interface"`
}

// handleDHCPFindActiveServer performs the following tasks:
//  1. searches for another DHCP server in the network;
//  2. check if a static IP is configured for the network interface;
//  3. responds with the results.
func (s *server) handleDHCPFindActiveServer(w http.ResponseWriter, r *http.Request) {
	if aghhttp.WriteTextPlainDeprecated(w, r) {
		return
	}

	req := &findActiveServerReq{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	ifaceName := req.Interface
	if ifaceName == "" {
		aghhttp.Error(r, w, http.StatusBadRequest, "empty interface name")

		return
	}

	result := &dhcpSearchResult{
		V4: dhcpSearchV4Result{
			OtherServer: dhcpSearchOtherResult{
				Found: "no",
			},
			StaticIP: dhcpStaticIPStatus{
				Static: "yes",
			},
		},
		V6: dhcpSearchV6Result{
			OtherServer: dhcpSearchOtherResult{
				Found: "no",
			},
		},
	}

	if isStaticIP, serr := aghnet.IfaceHasStaticIP(ifaceName); serr != nil {
		result.V4.StaticIP.Static = "error"
		result.V4.StaticIP.Error = serr.Error()
	} else if !isStaticIP {
		result.V4.StaticIP.Static = "no"
		// TODO(e.burkov):  The returned IP should only be of version 4.
		result.V4.StaticIP.IP = aghnet.GetSubnet(ifaceName).String()
	}

	setOtherDHCPResult(ifaceName, result)

	aghhttp.WriteJSONResponseOK(w, r, result)
}

// setOtherDHCPResult sets the results of the check for another DHCP server in
// result.
func setOtherDHCPResult(ifaceName string, result *dhcpSearchResult) {
	found4, found6, err4, err6 := aghnet.CheckOtherDHCP(ifaceName)
	if err4 != nil {
		result.V4.OtherServer.Found = "error"
		result.V4.OtherServer.Error = err4.Error()
	} else if found4 {
		result.V4.OtherServer.Found = "yes"
	}

	if err6 != nil {
		result.V6.OtherServer.Found = "error"
		result.V6.OtherServer.Error = err6.Error()
	} else if found6 {
		result.V6.OtherServer.Found = "yes"
	}
}

// parseLease parses a lease from r.  If there is no error returns DHCPServer
// and *Lease.  r must be non-nil.
func (s *server) parseLease(r io.Reader) (srv DHCPServer, lease *dhcpsvc.Lease, err error) {
	l := &leaseStatic{}
	err = json.NewDecoder(r).Decode(l)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding json: %w", err)
	}

	if !l.IP.IsValid() {
		return nil, nil, errors.Error("invalid ip")
	}

	l.IP = l.IP.Unmap()

	lease, err = l.toLease()
	if err != nil {
		return nil, nil, fmt.Errorf("parsing: %w", err)
	}

	if lease.IP.Is4() {
		srv = s.srv4
	} else {
		srv = s.srv6
	}

	return srv, lease, nil
}

// handleDHCPAddStaticLease is the handler for the POST
// /control/dhcp/add_static_lease HTTP API.
func (s *server) handleDHCPAddStaticLease(w http.ResponseWriter, r *http.Request) {
	srv, lease, err := s.parseLease(r.Body)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	if err = srv.AddStaticLease(lease); err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)
	}
}

// handleDHCPRemoveStaticLease is the handler for the POST
// /control/dhcp/remove_static_lease HTTP API.
func (s *server) handleDHCPRemoveStaticLease(w http.ResponseWriter, r *http.Request) {
	srv, lease, err := s.parseLease(r.Body)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	if err = srv.RemoveStaticLease(lease); err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)
	}
}

// handleDHCPUpdateStaticLease is the handler for the POST
// /control/dhcp/update_static_lease HTTP API.
func (s *server) handleDHCPUpdateStaticLease(w http.ResponseWriter, r *http.Request) {
	srv, lease, err := s.parseLease(r.Body)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	if err = srv.UpdateStaticLease(lease); err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)
	}
}

func (s *server) handleReset(w http.ResponseWriter, r *http.Request) {
	err := s.Stop()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "stopping dhcp: %s", err)

		return
	}

	err = os.Remove(s.conf.dbFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("dhcp: removing db: %s", err)
	}

	s.conf = &ServerConfig{
		ConfigModified: s.conf.ConfigModified,

		HTTPRegister: s.conf.HTTPRegister,

		LocalDomainName: s.conf.LocalDomainName,

		DataDir:    s.conf.DataDir,
		dbFilePath: s.conf.dbFilePath,
	}

	v4conf := &V4ServerConf{
		LeaseDuration: DefaultDHCPLeaseTTL,
		ICMPTimeout:   DefaultDHCPTimeoutICMP,
		notify:        s.onNotify,
	}
	s.srv4, _ = v4Create(v4conf)

	v6conf := V6ServerConf{
		LeaseDuration: DefaultDHCPLeaseTTL,
		notify:        s.onNotify,
	}
	s.srv6, _ = v6Create(v6conf)

	s.conf.ConfigModified()
}

func (s *server) handleResetLeases(w http.ResponseWriter, r *http.Request) {
	err := s.resetLeases()
	if err != nil {
		msg := "resetting leases: %s"
		aghhttp.Error(r, w, http.StatusInternalServerError, msg, err)

		return
	}
}

func (s *server) registerHandlers() {
	if s.conf.HTTPRegister == nil {
		return
	}

	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/status", s.handleDHCPStatus)
	s.conf.HTTPRegister(http.MethodGet, "/control/dhcp/interfaces", s.handleDHCPInterfaces)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/set_config", s.handleDHCPSetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/find_active_dhcp", s.handleDHCPFindActiveServer)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/add_static_lease", s.handleDHCPAddStaticLease)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/remove_static_lease", s.handleDHCPRemoveStaticLease)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/update_static_lease", s.handleDHCPUpdateStaticLease)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/reset", s.handleReset)
	s.conf.HTTPRegister(http.MethodPost, "/control/dhcp/reset_leases", s.handleResetLeases)
}
