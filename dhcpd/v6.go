package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"
)

const valIAID = "ADGH"

func (s *Server) v6GetLeases(flags int) []Lease {
	var result []Lease
	s.v6LeasesLock.Lock()
	for _, lease := range s.v6Leases {
		if (flags&LeasesStatic) != 0 && lease.Expiry.Unix() == leaseExpireStatic {
			result = append(result, *lease)
		}
	}
	s.v6LeasesLock.Unlock()
	return result
}

func (s *Server) v6AddStaticLease(l Lease) error {
	if len(l.IP) != 16 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}

	l.Expiry = time.Unix(leaseExpireStatic, 0)

	s.v6LeasesLock.Lock()
	s.v6Leases = append(s.v6Leases, &l)
	s.dbStore()
	s.v6LeasesLock.Unlock()
	// s.notify(LeaseChangedAddedStatic)
	return nil
}

// Remove a lease
func (s *Server) v6RmLease(l Lease) error {
	var newLeases []*Lease
	for _, lease := range s.v6Leases {
		if net.IP.Equal(lease.IP, l.IP) {
			if !bytes.Equal(lease.HWAddr, l.HWAddr) {
				return fmt.Errorf("Lease not found")
			}
			continue
		}
		newLeases = append(newLeases, lease)
	}

	if len(newLeases) == len(s.v6Leases) {
		return fmt.Errorf("Lease not found: %s", l.IP)
	}

	s.v6Leases = newLeases
	return nil
}

func (s *Server) v6RemoveStaticLease(l Lease) error {
	if len(l.IP) != 16 {
		return fmt.Errorf("invalid IP")
	}
	if len(l.HWAddr) != 6 {
		return fmt.Errorf("invalid MAC")
	}

	s.v6LeasesLock.Lock()
	err := s.v6RmLease(l)
	if err != nil {
		s.v6LeasesLock.Unlock()
		return err
	}
	s.dbStore()
	s.v6LeasesLock.Unlock()
	// s.notify(LeaseChangedRemovedStatic)
	return nil
}

func (s *Server) v6FindLease(mac net.HardwareAddr) *Lease {
	s.v6LeasesLock.Lock()
	defer s.v6LeasesLock.Unlock()

	for i := range s.v6Leases {
		if bytes.Equal(mac, s.v6Leases[i].HWAddr) {
			return s.v6Leases[i]
		}
	}
	return nil
}

func (s *Server) v6Process(req dhcpv6.DHCPv6, resp dhcpv6.DHCPv6) {
	mac, err := dhcpv6.ExtractMAC(req)
	if err != nil {
		log.Debug("DHCPv6: dhcpv6.ExtractMAC: %s", err)
		return
	}

	lease := s.v6FindLease(mac)
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
			PreferredLifetime: s.leaseTime,
			ValidLifetime:     s.leaseTime,
		},
	}}
	resp.AddOption(oia)
}

func (s *Server) v6PacketHandler(conn net.PacketConn, peer net.Addr, req dhcpv6.DHCPv6) {
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

func (s *Server) v6Start(iface net.Interface) error {
	laddr := &net.UDPAddr{
		IP:   net.ParseIP("::"),
		Port: dhcpv6.DefaultServerPort,
	}
	server, err := server6.NewServer(iface.Name, laddr, s.v6PacketHandler, server6.WithDebugLogger())
	if err != nil {
		return err
	}

	go func() {
		err = server.Serve()
		log.Error("DHCPv6: %s", err)
	}()
	return nil
}
