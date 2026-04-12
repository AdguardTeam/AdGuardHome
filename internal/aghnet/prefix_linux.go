//go:build linux

package aghnet

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

// ObserveIPv6Addrs returns IPv6 interface address state for ifaceName.
//
// ctx is accepted to match the BSD implementations, which run ifconfig under
// an [executil.CommandConstructor] and can be canceled.  Linux uses a netlink
// socket instead, so we still accept ctx for API compatibility but do not
// consult it while receiving the dump reply.
func ObserveIPv6Addrs(
	_ context.Context,
	_ *slog.Logger,
	_ executil.CommandConstructor,
	ifaceName string,
) (states []IPv6AddrState, err error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("finding interface %s: %w", ifaceName, err)
	}

	conn, err := netlink.Dial(unix.NETLINK_ROUTE, nil)
	if err != nil {
		return nil, fmt.Errorf("dialing rtnetlink: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, conn.Close()) }()

	msgs, err := conn.Execute(netlink.Message{
		Header: netlink.Header{
			Type:  netlink.HeaderType(unix.RTM_GETADDR),
			Flags: netlink.Request | netlink.Dump,
		},
		Data: []byte{unix.AF_INET6},
	})
	if err != nil {
		return nil, fmt.Errorf("querying rtnetlink addrs: %w", err)
	}

	return parseIPv6AddrStatesNetlink(msgs, iface.Index)
}

// parseIPv6AddrStatesNetlink parses IPv6 address state from netlink messages.
func parseIPv6AddrStatesNetlink(
	msgs []netlink.Message,
	ifIndex int,
) (states []IPv6AddrState, err error) {
	for _, msg := range msgs {
		state, done, ok, err := parseIPv6AddrStateMessage(msg, ifIndex)
		if err != nil {
			return nil, err
		}
		if done {
			return states, nil
		}
		if ok {
			states = append(states, state)
		}
	}

	return states, nil
}

// parseIPv6AddrStateMessage parses one netlink message carrying IPv6 address
// state.
func parseIPv6AddrStateMessage(
	msg netlink.Message,
	ifIndex int,
) (state IPv6AddrState, done, ok bool, err error) {
	switch msg.Header.Type {
	case netlink.Done:
		return IPv6AddrState{}, true, false, nil
	case netlink.HeaderType(unix.RTM_NEWADDR):
		// Go on.
	default:
		return IPv6AddrState{}, false, false, nil
	}

	ifam, err := parseIfAddrmsg(msg.Data)
	if err != nil {
		return IPv6AddrState{}, false, false, err
	}
	if ifam.Family != unix.AF_INET6 || int(ifam.Index) != ifIndex {
		return IPv6AddrState{}, false, false, nil
	}

	attrs, err := netlink.UnmarshalAttributes(msg.Data[unix.SizeofIfAddrmsg:])
	if err != nil {
		return IPv6AddrState{}, false, false, fmt.Errorf("parsing route attrs: %w", err)
	}

	state, ok, err = parseIPv6AddrStateNetlink(ifam, attrs)

	return state, false, ok, err
}

// parseIPv6AddrStateNetlink parses one IPv6 address state from the netlink
// message data.
func parseIPv6AddrStateNetlink(
	ifam unix.IfAddrmsg,
	attrs []netlink.Attribute,
) (state IPv6AddrState, ok bool, err error) {
	var addr netip.Addr
	var cache *ipv6AddrCacheInfo
	flags := uint32(ifam.Flags)

	for _, attr := range attrs {
		addr, flags, cache, err = parseIPv6AddrStateNetlinkAttr(attr, addr, flags, cache)
		if err != nil {
			return IPv6AddrState{}, false, err
		}
	}

	if !addr.IsValid() {
		return IPv6AddrState{}, false, nil
	}

	preferred, valid := uint32(^uint32(0)), uint32(^uint32(0))
	if cache != nil {
		preferred = cache.preferredLifetimeSec
		valid = cache.validLifetimeSec
	}

	return IPv6AddrState{
		Addr:                 addr,
		Prefix:               netip.PrefixFrom(addr, int(ifam.Prefixlen)).Masked(),
		PreferredLifetimeSec: preferred,
		ValidLifetimeSec:     valid,
		Temporary:            flags&unix.IFA_F_TEMPORARY != 0,
		Tentative:            flags&unix.IFA_F_TENTATIVE != 0,
	}, true, nil
}

// parseIPv6AddrStateNetlinkAttr parses one IPv6 address attribute from a
// netlink message.
func parseIPv6AddrStateNetlinkAttr(
	attr netlink.Attribute,
	addr netip.Addr,
	flags uint32,
	cache *ipv6AddrCacheInfo,
) (nextAddr netip.Addr, nextFlags uint32, nextCache *ipv6AddrCacheInfo, err error) {
	switch attr.Type {
	case unix.IFA_LOCAL:
		return parseIPv6AddrStateLocalAttr(attr.Data, flags, cache)
	case unix.IFA_ADDRESS:
		return parseIPv6AddrStateAddressAttr(attr.Data, addr, flags, cache)
	case unix.IFA_FLAGS:
		return parseIPv6AddrStateFlagsAttr(attr.Data, addr, cache)
	case unix.IFA_CACHEINFO:
		return parseIPv6AddrStateCacheAttr(attr.Data, addr, flags)
	default:
		return addr, flags, cache, nil
	}
}

// parseIPv6AddrStateLocalAttr parses an IFA_LOCAL attribute.
func parseIPv6AddrStateLocalAttr(
	data []byte,
	flags uint32,
	cache *ipv6AddrCacheInfo,
) (addr netip.Addr, nextFlags uint32, nextCache *ipv6AddrCacheInfo, err error) {
	addr, err = parseIPv6AddrAttr(data)
	if err != nil {
		return netip.Addr{}, 0, nil, fmt.Errorf("parsing ifa_local: %w", err)
	}

	return addr, flags, cache, nil
}

// parseIPv6AddrStateAddressAttr parses an IFA_ADDRESS attribute.
func parseIPv6AddrStateAddressAttr(
	data []byte,
	addr netip.Addr,
	flags uint32,
	cache *ipv6AddrCacheInfo,
) (nextAddr netip.Addr, nextFlags uint32, nextCache *ipv6AddrCacheInfo, err error) {
	if addr.IsValid() {
		return addr, flags, cache, nil
	}

	nextAddr, err = parseIPv6AddrAttr(data)
	if err != nil {
		return netip.Addr{}, 0, nil, fmt.Errorf("parsing ifa_address: %w", err)
	}

	return nextAddr, flags, cache, nil
}

// parseIPv6AddrStateFlagsAttr parses an IFA_FLAGS attribute.
func parseIPv6AddrStateFlagsAttr(
	data []byte,
	addr netip.Addr,
	cache *ipv6AddrCacheInfo,
) (nextAddr netip.Addr, nextFlags uint32, nextCache *ipv6AddrCacheInfo, err error) {
	if len(data) < 4 {
		return netip.Addr{}, 0, nil, fmt.Errorf("short ifa_flags attribute")
	}

	return addr, binary.NativeEndian.Uint32(data[:4]), cache, nil
}

// parseIPv6AddrStateCacheAttr parses an IFA_CACHEINFO attribute.
func parseIPv6AddrStateCacheAttr(
	data []byte,
	addr netip.Addr,
	flags uint32,
) (nextAddr netip.Addr, nextFlags uint32, nextCache *ipv6AddrCacheInfo, err error) {
	ifaCacheInfo, err := parseIfaCacheinfo(data)
	if err != nil {
		return netip.Addr{}, 0, nil, err
	}

	return addr, flags, &ifaCacheInfo, nil
}

// ipv6AddrCacheInfo is the lifetime subset of Linux ifa_cacheinfo used by
// IPv6 address observation.
type ipv6AddrCacheInfo struct {
	preferredLifetimeSec uint32
	validLifetimeSec     uint32
}

// parseIfAddrmsg parses one Linux ifaddrmsg structure.
func parseIfAddrmsg(b []byte) (ifam unix.IfAddrmsg, err error) {
	if len(b) < unix.SizeofIfAddrmsg {
		return unix.IfAddrmsg{}, fmt.Errorf("short ifaddrmsg payload")
	}

	return unix.IfAddrmsg{
		Family:    b[0],
		Prefixlen: b[1],
		Flags:     b[2],
		Scope:     b[3],
		Index:     binary.NativeEndian.Uint32(b[4:8]),
	}, nil
}

// parseIfaCacheinfo parses one Linux ifa_cacheinfo structure.
func parseIfaCacheinfo(b []byte) (cache ipv6AddrCacheInfo, err error) {
	if len(b) < unix.SizeofIfaCacheinfo {
		return ipv6AddrCacheInfo{}, fmt.Errorf("short ifa_cacheinfo attribute")
	}

	return ipv6AddrCacheInfo{
		preferredLifetimeSec: binary.NativeEndian.Uint32(b[0:4]),
		validLifetimeSec:     binary.NativeEndian.Uint32(b[4:8]),
	}, nil
}

// parseIPv6AddrAttr parses one IPv6 address attribute.
func parseIPv6AddrAttr(b []byte) (addr netip.Addr, err error) {
	if len(b) < net.IPv6len {
		return netip.Addr{}, fmt.Errorf("short ipv6 address")
	}

	var arr [net.IPv6len]byte
	copy(arr[:], b[:net.IPv6len])

	return netip.AddrFrom16(arr), nil
}
