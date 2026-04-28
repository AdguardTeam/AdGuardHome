//go:build linux

package aghnet

import (
	"encoding/binary"
	"net/netip"
	"testing"

	"github.com/mdlayher/netlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestParseIPv6AddrStateNetlink(t *testing.T) {
	addr := netip.MustParseAddr("2001:db8::1234").As16()
	flags := make([]byte, 4)
	binary.NativeEndian.PutUint32(flags, unix.IFA_F_TEMPORARY|unix.IFA_F_TENTATIVE)

	cacheBytes := make([]byte, unix.SizeofIfaCacheinfo)
	binary.NativeEndian.PutUint32(cacheBytes[0:4], 600)
	binary.NativeEndian.PutUint32(cacheBytes[4:8], 1200)

	state, ok, err := parseIPv6AddrStateNetlink(unix.IfAddrmsg{
		Family:    unix.AF_INET6,
		Prefixlen: 64,
	}, []netlink.Attribute{{
		Type: unix.IFA_ADDRESS,
		Data: addr[:],
	}, {
		Type: unix.IFA_FLAGS,
		Data: flags,
	}, {
		Type: unix.IFA_CACHEINFO,
		Data: cacheBytes,
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
