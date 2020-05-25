package dhcpd

import (
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

const defaultDiscoverTime = time.Second * 3
const leaseExpireStatic = 1

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

// ServerConfig - DHCP server configuration
// field ordering is important -- yaml fields will mirror ordering from here
type ServerConfig struct {
	Conf4 V4ServerConf `yaml:"dhcpv4"`
	Conf6 V6ServerConf `yaml:"dhcpv6"`

	WorkDir    string `yaml:"-"`
	DBFilePath string `yaml:"-"` // path to DB file

	// Called when the configuration is changed by HTTP request
	ConfigModified func() `yaml:"-"`

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request)) `yaml:"-"`
}

type onLeaseChangedT func(flags int)

// flags for onLeaseChanged()
const (
	LeaseChangedAdded = iota
	LeaseChangedAddedStatic
	LeaseChangedRemovedStatic
	LeaseChangedBlacklisted

	LeaseChangedDBStore
)

// Server - the current state of the DHCP server
type Server struct {
	srv4 DHCPServer
	srv6 DHCPServer

	conf ServerConfig

	// Called when the leases DB is modified
	onLeaseChanged onLeaseChangedT
}

// CheckConfig checks the configuration
func (s *Server) CheckConfig(config ServerConfig) error {
	return nil
}

// Create - create object
func Create(config ServerConfig) *Server {
	s := Server{}
	s.conf.HTTPRegister = config.HTTPRegister
	s.conf.ConfigModified = config.ConfigModified
	s.conf.DBFilePath = filepath.Join(config.WorkDir, dbFilename)

	if !webHandlersRegistered && s.conf.HTTPRegister != nil {
		webHandlersRegistered = true
		s.registerHandlers()
	}

	var err error
	config.Conf4.notify = s.onNotify
	s.srv4, err = v4Create(config.Conf4)
	if err != nil {
		log.Error("%s", err)
		return nil
	}

	config.Conf6.notify = s.onNotify
	s.srv6, err = v6Create(config.Conf6)
	if err != nil {
		log.Error("%s", err)
		return nil
	}

	// we can't delay database loading until DHCP server is started,
	//  because we need static leases functionality available beforehand
	s.dbLoad()
	return &s
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
func (s *Server) SetOnLeaseChanged(onLeaseChanged onLeaseChangedT) {
	s.onLeaseChanged = onLeaseChanged
}

func (s *Server) notify(flags int) {
	if s.onLeaseChanged == nil {
		return
	}
	s.onLeaseChanged(flags)
}

// WriteDiskConfig - write configuration
func (s *Server) WriteDiskConfig(c *ServerConfig) {
	s.srv4.WriteDiskConfig4(&c.Conf4)
	s.srv6.WriteDiskConfig6(&c.Conf6)
}

// Start will listen on port 67 and serve DHCP requests.
func (s *Server) Start() error {
	err := s.srv4.Start()
	if err != nil {
		return err
	}

	err = s.srv6.Start()
	if err != nil {
		return err
	}

	return nil
}

// Stop closes the listening UDP socket
func (s *Server) Stop() error {
	s.srv4.Stop()
	s.srv6.Stop()
	return nil
}

// flags for Leases() function
const (
	LeasesDynamic = 1
	LeasesStatic  = 2
	LeasesAll     = LeasesDynamic | LeasesStatic
)

// Leases returns the list of current DHCP leases (thread-safe)
func (s *Server) Leases(flags int) []Lease {
	result := s.srv4.GetLeases(flags)

	if s.srv6 != nil {
		v6leases := s.srv6.GetLeases(flags)
		result = append(result, v6leases...)
	}

	return result
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
