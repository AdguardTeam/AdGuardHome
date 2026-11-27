//go:build windows

package dhcpd

// 'u-root/u-root' package, a dependency of 'insomniacslk/dhcp' package, doesn't build on Windows

import (
	"context"
	"net"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
)

type winServer struct{}

// type check
var _ DHCPServer = winServer{}

func (winServer) ResetLeases(_ []*dhcpsvc.Lease) (err error)           { return nil }
func (winServer) GetLeases(_ GetLeasesFlags) (leases []*dhcpsvc.Lease) { return nil }
func (winServer) getLeasesRef() []*dhcpsvc.Lease                       { return nil }
func (winServer) AddStaticLease(_ *dhcpsvc.Lease) (err error)          { return nil }
func (winServer) RemoveStaticLease(_ *dhcpsvc.Lease) (err error)       { return nil }
func (winServer) UpdateStaticLease(_ *dhcpsvc.Lease) (err error)       { return nil }
func (winServer) FindMACbyIP(_ netip.Addr) (mac net.HardwareAddr)      { return nil }
func (winServer) WriteDiskConfig4(_ *configmgr.DHCPv4Config)           {}
func (winServer) WriteDiskConfig6(_ *configmgr.DHCPv6Config)           {}
func (winServer) Start(_ context.Context) (err error)                  { return nil }
func (winServer) Stop() (err error)                                    { return nil }
func (winServer) HostByIP(_ netip.Addr) (host string)                  { return "" }
func (winServer) IPByHost(_ string) (ip netip.Addr)                    { return netip.Addr{} }

func newV4Server(_ *V4ServerConf) (s DHCPServer, err error) { return winServer{}, nil }
func newV6Server(_ V6ServerConf) (s DHCPServer, err error)  { return winServer{}, nil }
