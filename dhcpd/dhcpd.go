package dhcpd

import (
	"encoding/hex"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"
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

type OnLeaseChangedT func(flags int)

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
	onLeaseChanged []OnLeaseChangedT
}

type ServerInterface interface {
	Leases(flags int) []Lease
	SetOnLeaseChanged(onLeaseChanged OnLeaseChangedT)
}

// CheckConfig checks the configuration
func (s *Server) CheckConfig(config ServerConfig) error {
	return nil
}

// Create - create object
func Create(config ServerConfig) *Server {
	s := Server{}
	s.conf.Enabled = config.Enabled
	s.conf.InterfaceName = config.InterfaceName
	s.conf.HTTPRegister = config.HTTPRegister
	s.conf.ConfigModified = config.ConfigModified
	s.conf.DBFilePath = filepath.Join(config.WorkDir, dbFilename)

	if !webHandlersRegistered && s.conf.HTTPRegister != nil {
		webHandlersRegistered = true
		s.registerHandlers()
	}

	var err4, err6 error
	v4conf := config.Conf4
	v4conf.Enabled = s.conf.Enabled
	if len(v4conf.RangeStart) == 0 {
		v4conf.Enabled = false
	}
	v4conf.InterfaceName = s.conf.InterfaceName
	v4conf.notify = s.onNotify
	s.srv4, err4 = v4Create(v4conf)

	v6conf := config.Conf6
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

	if s.conf.Enabled && !v4conf.Enabled && !v6conf.Enabled {
		log.Error("Can't enable DHCP server because neither DHCPv4 nor DHCPv6 servers are configured")
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

// Leases returns the list of current DHCP leases (thread-safe)
func (s *Server) Leases(flags int) []Lease {
	result := s.srv4.GetLeases(flags)

	v6leases := s.srv6.GetLeases(flags)
	result = append(result, v6leases...)

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

// Parse option string
// Format:
// CODE TYPE VALUE
func parseOptionString(s string) (uint8, []byte) {
	s = strings.TrimSpace(s)
	scode := util.SplitNext(&s, ' ')
	t := util.SplitNext(&s, ' ')
	sval := util.SplitNext(&s, ' ')

	code, err := strconv.Atoi(scode)
	if err != nil || code <= 0 || code > 255 {
		return 0, nil
	}

	var val []byte

	switch t {
	case "hex":
		val, err = hex.DecodeString(sval)
		if err != nil {
			return 0, nil
		}

	case "ip":
		ip := net.ParseIP(sval)
		if ip == nil {
			return 0, nil
		}
		val = ip
		if ip.To4() != nil {
			val = ip.To4()
		}

	default:
		return 0, nil
	}

	return uint8(code), val
}
