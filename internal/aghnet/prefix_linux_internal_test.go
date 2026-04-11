//go:build linux

package aghnet

import (
	"encoding/binary"
	"net/netip"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestParseIPv6AddrStateNetlink(t *testing.T) {
	addr := netip.MustParseAddr("2001:db8::1234").As16()
	flags := make([]byte, 4)
	binary.NativeEndian.PutUint32(flags, unix.IFA_F_TEMPORARY|unix.IFA_F_TENTATIVE)

	cache := unix.IfaCacheinfo{
		Prefered: 600,
		Valid:    1200,
	}
	cacheBytes := *(*[unix.SizeofIfaCacheinfo]byte)(unsafe.Pointer(&cache))

	state, ok, err := parseIPv6AddrStateNetlink(&syscall.IfAddrmsg{
		Family:    syscall.AF_INET6,
		Prefixlen: 64,
	}, []syscall.NetlinkRouteAttr{{
		Attr:  syscall.RtAttr{Type: unix.IFA_ADDRESS},
		Value: addr[:],
	}, {
		Attr:  syscall.RtAttr{Type: unix.IFA_FLAGS},
		Value: flags,
	}, {
		Attr:  syscall.RtAttr{Type: unix.IFA_CACHEINFO},
		Value: cacheBytes[:],
	}})
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, netip.MustParseAddr("2001:db8::1234"), state.Addr)
	assert.Equal(t, netip.MustParsePrefix("2001:db8::/64"), state.Prefix)
	assert.Equal(t, uint32(600), state.PreferredLifetimeSec)
	assert.Equal(t, uint32(1200), state.ValidLifetimeSec)
	assert.True(t, state.Temporary)
	assert.True(t, state.Tentative)
}
