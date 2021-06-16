//go:build windows
// +build windows

package dhcpd

// 'u-root/u-root' package, a dependency of 'insomniacslk/dhcp' package, doesn't build on Windows

import "net"

type winServer struct{}

func (s *winServer) ResetLeases(_ []*Lease) (err error)           { return nil }
func (s *winServer) GetLeases(_ GetLeasesFlags) (leases []*Lease) { return nil }
func (s *winServer) getLeasesRef() []*Lease                       { return nil }
func (s *winServer) AddStaticLease(_ *Lease) (err error)          { return nil }
func (s *winServer) RemoveStaticLease(_ *Lease) (err error)       { return nil }
func (s *winServer) FindMACbyIP(ip net.IP) (mac net.HardwareAddr) { return nil }
func (s *winServer) WriteDiskConfig4(c *V4ServerConf)             {}
func (s *winServer) WriteDiskConfig6(c *V6ServerConf)             {}
func (s *winServer) Start() (err error)                           { return nil }
func (s *winServer) Stop() (err error)                            { return nil }
func v4Create(conf V4ServerConf) (DHCPServer, error)              { return &winServer{}, nil }
func v6Create(conf V6ServerConf) (DHCPServer, error)              { return &winServer{}, nil }
