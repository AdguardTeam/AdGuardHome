//go:build windows

package dhcpd

// 'u-root/u-root' package, a dependency of 'insomniacslk/dhcp' package, doesn't build on Windows

import "net"

type winServer struct{}

// type check
var _ DHCPServer = winServer{}

func (winServer) ResetLeases(_ []*Lease) (err error)           { return nil }
func (winServer) GetLeases(_ GetLeasesFlags) (leases []*Lease) { return nil }
func (winServer) getLeasesRef() []*Lease                       { return nil }
func (winServer) AddStaticLease(_ *Lease) (err error)          { return nil }
func (winServer) RemoveStaticLease(_ *Lease) (err error)       { return nil }
func (winServer) FindMACbyIP(_ net.IP) (mac net.HardwareAddr)  { return nil }
func (winServer) WriteDiskConfig4(_ *V4ServerConf)             {}
func (winServer) WriteDiskConfig6(_ *V6ServerConf)             {}
func (winServer) Start() (err error)                           { return nil }
func (winServer) Stop() (err error)                            { return nil }

func v4Create(_ *V4ServerConf) (s DHCPServer, err error) { return winServer{}, nil }
func v6Create(_ V6ServerConf) (s DHCPServer, err error)  { return winServer{}, nil }
