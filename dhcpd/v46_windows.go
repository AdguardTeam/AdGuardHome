package dhcpd

// 'u-root/u-root' package, a dependency of 'insomniacslk/dhcp' package, doesn't build on Windows

import "net"

type winServer struct {
}

func (s *winServer) ResetLeases(leases []*Lease) {
}
func (s *winServer) GetLeases(flags int) []Lease {
	return nil
}
func (s *winServer) GetLeasesRef() []*Lease {
	return nil
}
func (s *winServer) AddStaticLease(lease Lease) error {
	return nil
}
func (s *winServer) RemoveStaticLease(l Lease) error {
	return nil
}
func (s *winServer) FindMACbyIP(ip net.IP) net.HardwareAddr {
	return nil
}

func (s *winServer) WriteDiskConfig4(c *V4ServerConf) {
}
func (s *winServer) WriteDiskConfig6(c *V6ServerConf) {
}

func (s *winServer) Start() error {
	return nil
}
func (s *winServer) Stop() {
}
func (s *winServer) Reset() {
}

func v4Create(conf V4ServerConf) (DHCPServer, error) {
	return &winServer{}, nil
}

func v6Create(conf V6ServerConf) (DHCPServer, error) {
	return &winServer{}, nil
}
