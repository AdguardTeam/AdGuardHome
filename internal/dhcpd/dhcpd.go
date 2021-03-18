// Package dhcpd provides a DHCP server.
package dhcpd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

const (
	defaultDiscoverTime = time.Second * 3
	// leaseExpireStatic is used to define the Expiry field for static
	// leases.
	//
	// TODO(e.burkov): Remove it when static leases determining mechanism
	// will be improved.
	leaseExpireStatic = 1
)

var webHandlersRegistered = false

// Lease contains the necessary information about a DHCP lease
type Lease struct {
	HWAddr   net.HardwareAddr `json:"mac"`
	IP       net.IP           `json:"ip"`
	Hostname string           `json:"hostname"`

	// Lease expiration time
	// 1: static lease
	Expiry time.Time `json:"expires"`
}

// IsStatic returns true if the lease is static.
//
// TODO(a.garipov): Just make it a boolean field.
func (l *Lease) IsStatic() (ok bool) {
	return l != nil && l.Expiry.Unix() == leaseExpireStatic
}

// MarshalJSON implements the json.Marshaler interface for *Lease.
func (l *Lease) MarshalJSON() ([]byte, error) {
	var expiryStr string
	if !l.IsStatic() {
		// The front-end is waiting for RFC 3999 format of the time
		// value.  It also shouldn't got an Expiry field for static
		// leases.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2692.
		expiryStr = l.Expiry.Format(time.RFC3339)
	}

	type lease Lease
	return json.Marshal(&struct {
		HWAddr string `json:"mac"`
		Expiry string `json:"expires,omitempty"`
		*lease
	}{
		HWAddr: l.HWAddr.String(),
		Expiry: expiryStr,
		lease:  (*lease)(l),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface for *Lease.
func (l *Lease) UnmarshalJSON(data []byte) (err error) {
	type lease Lease
	aux := struct {
		HWAddr string `json:"mac"`
		*lease
	}{
		lease: (*lease)(l),
	}
	if err = json.Unmarshal(data, &aux); err != nil {
		return err
	}

	l.HWAddr, err = net.ParseMAC(aux.HWAddr)
	if err != nil {
		return fmt.Errorf("couldn't parse MAC address: %w", err)
	}

	return nil
}

// ServerConfig - DHCP server configuration
// field ordering is important -- yaml fields will mirror ordering from here
type ServerConfig struct {
	Enabled       bool   `yaml:"enabled"`
	InterfaceName string `yaml:"interface_name"`

	Conf4 V4ServerConf `yaml:"dhcpv4"`
	Conf6 V6ServerConf `yaml:"dhcpv6"`

	WorkDir    string `yaml:"-"`
	DBFilePath string `yaml:"-"` // path to DB file

	// Called when the configuration is changed by HTTP request
	ConfigModified func() `yaml:"-"`

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request)) `yaml:"-"`
}

// OnLeaseChangedT is a callback for lease changes.
type OnLeaseChangedT func(flags int)

// flags for onLeaseChanged()
const (
	LeaseChangedAdded = iota
	LeaseChangedAddedStatic
	LeaseChangedRemovedStatic

	LeaseChangedDBStore
)

// Server - the current state of the DHCP server
type Server struct {
	srv4 DHCPServer
	srv6 DHCPServer

	conf ServerConfig

	// Called when the leases DB is modified
	onLeaseChanged []OnLeaseChangedT
}

// ServerInterface is an interface for servers.
type ServerInterface interface {
	Leases(flags int) []Lease
	SetOnLeaseChanged(onLeaseChanged OnLeaseChangedT)
}

// Create - create object
func Create(conf ServerConfig) *Server {
	s := &Server{}

	s.conf.Enabled = conf.Enabled
	s.conf.InterfaceName = conf.InterfaceName
	s.conf.HTTPRegister = conf.HTTPRegister
	s.conf.ConfigModified = conf.ConfigModified
	s.conf.DBFilePath = filepath.Join(conf.WorkDir, dbFilename)

	if !webHandlersRegistered && s.conf.HTTPRegister != nil {
		if runtime.GOOS == "windows" {
			// Our DHCP server doesn't work on Windows yet, so
			// signal that to the front with an HTTP 501.
			//
			// TODO(a.garipov): This needs refactoring.  We
			// shouldn't even try and initialize a DHCP server on
			// Windows, but there are currently too many
			// interconnected parts--such as HTTP handlers and
			// frontend--to make that work properly.
			s.registerNotImplementedHandlers()
		} else {
			s.registerHandlers()
		}

		webHandlersRegistered = true
	}

	var err4, err6 error
	v4conf := conf.Conf4
	v4conf.Enabled = s.conf.Enabled
	if len(v4conf.RangeStart) == 0 {
		v4conf.Enabled = false
	}
	v4conf.InterfaceName = s.conf.InterfaceName
	v4conf.notify = s.onNotify
	s.srv4, err4 = v4Create(v4conf)

	v6conf := conf.Conf6
	v6conf.Enabled = s.conf.Enabled
	if len(v6conf.RangeStart) == 0 {
		v6conf.Enabled = false
	}
	v6conf.InterfaceName = s.conf.InterfaceName
	v6conf.notify = s.onNotify
	s.srv6, err6 = v6Create(v6conf)

	if err4 != nil {
		log.Error("%s", err4)
		return nil
	}
	if err6 != nil {
		log.Error("%s", err6)
		return nil
	}

	s.conf.Conf4 = conf.Conf4
	s.conf.Conf6 = conf.Conf6

	if s.conf.Enabled && !v4conf.Enabled && !v6conf.Enabled {
		log.Error("Can't enable DHCP server because neither DHCPv4 nor DHCPv6 servers are configured")
		return nil
	}

	// we can't delay database loading until DHCP server is started,
	//  because we need static leases functionality available beforehand
	s.dbLoad()
	return s
}

// server calls this function after DB is updated
func (s *Server) onNotify(flags uint32) {
	if flags == LeaseChangedDBStore {
		s.dbStore()

		return
	}

	s.notify(int(flags))
}

// SetOnLeaseChanged - set callback
func (s *Server) SetOnLeaseChanged(onLeaseChanged OnLeaseChangedT) {
	s.onLeaseChanged = append(s.onLeaseChanged, onLeaseChanged)
}

func (s *Server) notify(flags int) {
	if len(s.onLeaseChanged) == 0 {
		return
	}

	for _, f := range s.onLeaseChanged {
		f(flags)
	}
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *ServerConfig) {
	c.Enabled = s.conf.Enabled
	c.InterfaceName = s.conf.InterfaceName
	s.srv4.WriteDiskConfig4(&c.Conf4)
	s.srv6.WriteDiskConfig6(&c.Conf6)
}

// Start will listen on port 67 and serve DHCP requests.
func (s *Server) Start() error {
	err := s.srv4.Start()
	if err != nil {
		log.Error("DHCPv4: start: %s", err)
		return err
	}

	err = s.srv6.Start()
	if err != nil {
		log.Error("DHCPv6: start: %s", err)
		return err
	}

	return nil
}

// Stop closes the listening UDP socket
func (s *Server) Stop() {
	s.srv4.Stop()
	s.srv6.Stop()
}

// flags for Leases() function
const (
	LeasesDynamic = 1
	LeasesStatic  = 2
	LeasesAll     = LeasesDynamic | LeasesStatic
)

// Leases returns the list of active IPv4 and IPv6 DHCP leases.  It's safe for
// concurrent use.
func (s *Server) Leases(flags int) (leases []Lease) {
	return append(s.srv4.GetLeases(flags), s.srv6.GetLeases(flags)...)
}

// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
func (s *Server) FindMACbyIP(ip net.IP) net.HardwareAddr {
	if ip.To4() != nil {
		return s.srv4.FindMACbyIP(ip)
	}
	return s.srv6.FindMACbyIP(ip)
}

// AddStaticLease - add static v4 lease
func (s *Server) AddStaticLease(lease Lease) error {
	return s.srv4.AddStaticLease(lease)
}
