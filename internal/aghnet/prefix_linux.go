//go:build linux

package aghnet

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"syscall"
	"unsafe"

	"github.com/AdguardTeam/golibs/osutil/executil"
	"golang.org/x/sys/unix"
)

// ObserveIPv6Addrs returns IPv6 interface address state for ifaceName.
//
// ctx is accepted to match the BSD implementations (which run ifconfig under
// an [executil.CommandConstructor] and can be cancelled) but is not honored
// here: syscall.NetlinkRIB is synchronous and uncancellable from outside the
// call.  Wrapping it in a goroutine that selects on ctx.Done() would only
// hide a stuck kernel from the caller while leaking the blocked goroutine on
// every retry, which is strictly worse than failing fast on the caller side
// and letting the operator notice a genuinely broken environment.
// rtnetlink responds in microseconds under normal conditions, so the lack of
// cancellation is acceptable in practice.
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

	rib, err := syscall.NetlinkRIB(syscall.RTM_GETADDR, syscall.AF_INET6)
	if err != nil {
		return nil, fmt.Errorf("querying rtnetlink addrs: %w", err)
	}

	msgs, err := syscall.ParseNetlinkMessage(rib)
	if err != nil {
		return nil, fmt.Errorf("parsing rtnetlink addrs: %w", err)
	}

	return parseIPv6AddrStatesNetlink(msgs, iface.Index)
}

// parseIPv6AddrStatesNetlink parses IPv6 address state from netlink messages.
func parseIPv6AddrStatesNetlink(
	msgs []syscall.NetlinkMessage,
	ifIndex int,
) (states []IPv6AddrState, err error) {
loop:
	for _, msg := range msgs {
		switch msg.Header.Type {
		case syscall.NLMSG_DONE:
			break loop
		case syscall.RTM_NEWADDR:
			// Go on.
		default:
			continue
		}

		if len(msg.Data) < syscall.SizeofIfAddrmsg {
			return nil, fmt.Errorf("short ifaddrmsg payload")
		}

		ifam := (*syscall.IfAddrmsg)(unsafe.Pointer(&msg.Data[0]))
		if ifam.Family != syscall.AF_INET6 || int(ifam.Index) != ifIndex {
			continue
		}

		attrs, err := syscall.ParseNetlinkRouteAttr(&msg)
		if err != nil {
			return nil, fmt.Errorf("parsing route attrs: %w", err)
		}

		state, ok, err := parseIPv6AddrStateNetlink(ifam, attrs)
		if err != nil {
			return nil, err
		} else if ok {
			states = append(states, state)
		}
	}

	return states, nil
}

// parseIPv6AddrStateNetlink parses one IPv6 address state from the netlink
// message data.
func parseIPv6AddrStateNetlink(
	ifam *syscall.IfAddrmsg,
	attrs []syscall.NetlinkRouteAttr,
) (state IPv6AddrState, ok bool, err error) {
	var addr netip.Addr
	var cache *unix.IfaCacheinfo
	flags := uint32(ifam.Flags)

	for _, attr := range attrs {
		switch attr.Attr.Type {
		case unix.IFA_LOCAL:
			addr, err = parseIPv6AddrAttr(attr.Value)
			if err != nil {
				return IPv6AddrState{}, false, fmt.Errorf("parsing ifa_local: %w", err)
			}
		case unix.IFA_ADDRESS:
			if addr.IsValid() {
				continue
			}

			addr, err = parseIPv6AddrAttr(attr.Value)
			if err != nil {
				return IPv6AddrState{}, false, fmt.Errorf("parsing ifa_address: %w", err)
			}
		case unix.IFA_FLAGS:
			if len(attr.Value) < 4 {
				return IPv6AddrState{}, false, fmt.Errorf("short ifa_flags attribute")
			}

			flags = binary.NativeEndian.Uint32(attr.Value[:4])
		case unix.IFA_CACHEINFO:
			if len(attr.Value) < unix.SizeofIfaCacheinfo {
				return IPv6AddrState{}, false, fmt.Errorf("short ifa_cacheinfo attribute")
			}

			cache = (*unix.IfaCacheinfo)(unsafe.Pointer(&attr.Value[0]))
		}
	}

	if !addr.IsValid() {
		return IPv6AddrState{}, false, nil
	}

	preferred, valid := uint32(^uint32(0)), uint32(^uint32(0))
	if cache != nil {
		preferred = cache.Prefered
		valid = cache.Valid
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

// parseIPv6AddrAttr parses one IPv6 address attribute.
func parseIPv6AddrAttr(b []byte) (addr netip.Addr, err error) {
	if len(b) < net.IPv6len {
		return netip.Addr{}, fmt.Errorf("short ipv6 address")
	}

	var arr [net.IPv6len]byte
	copy(arr[:], b[:net.IPv6len])

	return netip.AddrFrom16(arr), nil
}
