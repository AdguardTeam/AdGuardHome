package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"
)

const valIAID = "ADGH"

// V6Server - DHCPv6 server
type V6Server struct {
	s4         *Server // for dbStore()
	srv        *server6.Server
	leases     []*Lease
	leasesLock sync.Mutex

	conf V6ServerConf
}

// V6ServerConf - server configuration
type V6ServerConf struct {
	Enabled       bool   `yaml:"enabled"`
	RangeStart    string `yaml:"range_start"`
	LeaseDuration uint32 `yaml:"lease_duration"` // in seconds
	leaseTime     time.Duration
}

// GetLeases - get current leases
func (s *V6Server) GetLeases(flags int) []Lease {
	var result []Lease
	s.leasesLock.Lock()
	for _, lease := range s.leases {
		if (flags&LeasesStatic) != 0 && lease.Expiry.Unix() == leaseExpireStatic {
			result = append(result, *lease)
		}
	}
	s.leasesLock.Unlock()
	return result
}

// AddStaticLease - add a static lease
func (s *V6Server) AddStaticLease(l Lease) error {
	if len(l.IP) != 16 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}

	l.Expiry = time.Unix(leaseExpireStatic, 0)

	s.leasesLock.Lock()
	err := s.addLease(l)
	if err != nil {
		s.leasesLock.Unlock()
		return err
	}
	s.s4.dbStore()
	s.leasesLock.Unlock()
	// s.notify(LeaseChangedAddedStatic)
	return nil
}

// RemoveStaticLease - remove a static lease
func (s *V6Server) RemoveStaticLease(l Lease) error {
	if len(l.IP) != 16 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}

	s.leasesLock.Lock()
	err := s.rmLease(l)
	if err != nil {
		s.leasesLock.Unlock()
		return err
	}
	s.s4.dbStore()
	s.leasesLock.Unlock()
	// s.notify(LeaseChangedRemovedStatic)
	return nil
}

// Add a lease
func (s *V6Server) addLease(l Lease) error {
	s.leases = append(s.leases, &l)
	return nil
}

// Remove a lease
func (s *V6Server) rmLease(l Lease) error {
	var newLeases []*Lease
	for _, lease := range s.leases {
		if net.IP.Equal(lease.IP, l.IP) {
			if !bytes.Equal(lease.HWAddr, l.HWAddr) {
				return fmt.Errorf("Lease not found")
			}
			continue
		}
		newLeases = append(newLeases, lease)
	}

	if len(newLeases) == len(s.leases) {
		return fmt.Errorf("Lease not found: %s", l.IP)
	}

	s.leases = newLeases
	return nil
}

func (s *V6Server) findLease(mac net.HardwareAddr) *Lease {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for i := range s.leases {
		if bytes.Equal(mac, s.leases[i].HWAddr) {
			return s.leases[i]
		}
	}
	return nil
}

func (s *V6Server) v6Process(req dhcpv6.DHCPv6, resp dhcpv6.DHCPv6) {
	mac, err := dhcpv6.ExtractMAC(req)
	if err != nil {
		log.Debug("DHCPv6: dhcpv6.ExtractMAC: %s", err)
		return
	}

	lease := s.findLease(mac)
	if lease == nil {
		log.Debug("DHCPv6: no lease for: %s", mac)
		return
	}

	osid := dhcpv6.OptServerID(dhcpv6.Duid{
		Type:          dhcpv6.DUID_LLT,
		HwType:        iana.HWTypeEthernet,
		LinkLayerAddr: []byte{1, 2, 3, 4, 5, 6},
	})
	resp.AddOption(osid)

	oia := &dhcpv6.OptIANA{}
	copy(oia.IaId[:], []byte(valIAID))
	oia.Options = dhcpv6.IdentityOptions{Options: []dhcpv6.Option{
		&dhcpv6.OptIAAddress{
			IPv6Addr:          lease.IP,
			PreferredLifetime: s.conf.leaseTime,
			ValidLifetime:     s.conf.leaseTime,
		},
	}}
	resp.AddOption(oia)
}

func (s *V6Server) packetHandler(conn net.PacketConn, peer net.Addr, req dhcpv6.DHCPv6) {
	msg, err := req.GetInnerMessage()
	if err != nil {
		log.Error("DHCPv6: %s", err)
		return
	}

	log.Debug("DHCPv6: received: %s", req.Summary())

	if msg.GetOneOption(dhcpv6.OptionClientID) == nil {
		log.Error("DHCPv6: no ClientID option in request")
		return
	}

	var resp dhcpv6.DHCPv6

	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		if msg.GetOneOption(dhcpv6.OptionRapidCommit) != nil {
			resp, err = dhcpv6.NewReplyFromMessage(msg)
		} else {
			resp, err = dhcpv6.NewAdvertiseFromSolicit(msg)
		}

	case dhcpv6.MessageTypeRequest,
		dhcpv6.MessageTypeConfirm,
		dhcpv6.MessageTypeRenew,
		dhcpv6.MessageTypeRebind,
		dhcpv6.MessageTypeRelease,
		dhcpv6.MessageTypeInformationRequest:
		resp, err = dhcpv6.NewReplyFromMessage(msg)

	default:
		err = fmt.Errorf("message type %d not supported", msg.Type())
	}

	if err != nil {
		log.Error("DHCPv6: %s", err)
		return
	}

	s.v6Process(req, resp)

	log.Debug("DHCPv6: sending: %s", resp.Summary())

	_, err = conn.WriteTo(resp.ToBytes(), peer)
	if err != nil {
		log.Error("DHCPv6: conn.Write to %s failed: %s", peer, err)
		return
	}
}

// Start - start server
func (s *V6Server) Start(iface net.Interface) error {
	laddr := &net.UDPAddr{
		IP:   net.ParseIP("::"),
		Port: dhcpv6.DefaultServerPort,
	}
	server, err := server6.NewServer(iface.Name, laddr, s.packetHandler, server6.WithDebugLogger())
	if err != nil {
		return err
	}

	go func() {
		err = server.Serve()
		log.Error("DHCPv6: %s", err)
	}()
	return nil
}

func v6Create(conf V6ServerConf) *V6Server {
	s := &V6Server{}
	s.conf = conf

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = time.Hour * 2
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	return s
}
