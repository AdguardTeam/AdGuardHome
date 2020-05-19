package dhcpd

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

const defaultDiscoverTime = time.Second * 3
const leaseExpireStatic = 1

var webHandlersRegistered = false

// Lease contains the necessary information about a DHCP lease
// field ordering is important -- yaml fields will mirror ordering from here
type Lease struct {
	HWAddr   net.HardwareAddr `json:"mac" yaml:"hwaddr"`
	IP       net.IP           `json:"ip"`
	Hostname string           `json:"hostname"`

	// Lease expiration time
	// 1: static lease
	Expiry time.Time `json:"expires"`
}

// ServerConfig - DHCP server configuration
// field ordering is important -- yaml fields will mirror ordering from here
type ServerConfig struct {
	Conf4 V4ServerConf `json:"-" yaml:"dhcpv4"`
	Conf6 V6ServerConf `json:"-" yaml:"dhcpv6"`

	WorkDir    string `json:"-" yaml:"-"`
	DBFilePath string `json:"-" yaml:"-"` // path to DB file

	// Called when the configuration is changed by HTTP request
	ConfigModified func() `json:"-" yaml:"-"`

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request)) `json:"-" yaml:"-"`
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
	// conn *filterConn // listening UDP socket

	// ipnet *net.IPNet // if interface name changes, this needs to be reset

	// cond     *sync.Cond // Synchronize worker thread with main thread
	// mutex    sync.Mutex // Mutex for 'cond'
	// running  bool       // Set if the worker thread is running
	// stopping bool       // Set if the worker thread should be stopped

	// leases
	// leaseOptions dhcp4.Options // parsed from config GatewayIP and SubnetMask

	srv4 *V4Server
	srv6 *V6Server

	conf ServerConfig

	// Called when the leases DB is modified
	onLeaseChanged onLeaseChangedT
}

// Print information about the available network interfaces
func printInterfaces() {
	ifaces, _ := net.Interfaces()
	var buf strings.Builder
	for i := range ifaces {
		buf.WriteString(fmt.Sprintf("\"%s\", ", ifaces[i].Name))
	}
	log.Info("Available network interfaces: %s", buf.String())
}

// CheckConfig checks the configuration
func (s *Server) CheckConfig(config ServerConfig) error {
	return nil
}

// Create - create object
func Create(config ServerConfig) *Server {
	s := Server{}
	s.conf.Conf4.notify = s.onNotify
	s.conf.Conf6.notify = s.onNotify
	s.conf.DBFilePath = filepath.Join(config.WorkDir, dbFilename)

	if !webHandlersRegistered && s.conf.HTTPRegister != nil {
		webHandlersRegistered = true
		s.registerHandlers()
	}

	var err error
	s.srv4, err = v4Create(config.Conf4)
	if s.srv4 == nil {
		log.Error("%s", err)
		return nil
	}

	s.srv6, err = v6Create(config.Conf6)
	if s.srv6 == nil {
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

// Init checks the configuration and initializes the server
func (s *Server) Init(config ServerConfig) error {
	return nil
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
	s.srv4.WriteDiskConfig(&c.Conf4)
	s.srv6.WriteDiskConfig(&c.Conf6)
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
		return s.srv4.FindMACbyIP4(ip)
	}
	return s.srv6.FindMACbyIP6(ip)
}

// Reset internal state
func (s *Server) reset() {
	s.srv4.Reset()
	s.srv6.Reset()
}
