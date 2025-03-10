package aghnet

import (
	"net"
	"testing"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/stretchr/testify/assert"
)

func TestIPMut(t *testing.T) {
	testIPs := []net.IP{{
		127, 0, 0, 1,
	}, {
		192, 168, 0, 1,
	}, {
		8, 8, 8, 8,
	}}

	t.Run("nil_no_mut", func(t *testing.T) {
		ipmut := NewIPMut(nil)

		ips := netutil.CloneIPs(testIPs)
		for i := range ips {
			ipmut.Load()(ips[i])
			assert.True(t, ips[i].Equal(testIPs[i]))
		}
	})

	t.Run("not_nil_mut", func(t *testing.T) {
		ipmut := NewIPMut(func(ip net.IP) {
			for i := range ip {
				ip[i] = 0
			}
		})
		want := netutil.IPv4Zero()

		ips := netutil.CloneIPs(testIPs)
		for i := range ips {
			ipmut.Load()(ips[i])
			assert.True(t, ips[i].Equal(want))
		}
	})
}
