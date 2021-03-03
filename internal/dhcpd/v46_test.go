package dhcpd

import (
	"errors"
	"net"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeIface struct {
	addrs []net.Addr
	err   error
}

// Addrs implements the netIface interface for *fakeIface.
func (iface *fakeIface) Addrs() (addrs []net.Addr, err error) {
	if iface.err != nil {
		return nil, iface.err
	}

	return iface.addrs, nil
}

func TestIfaceIPAddrs(t *testing.T) {
	const errTest agherr.Error = "test error"

	ip4 := net.IP{1, 2, 3, 4}
	addr4 := &net.IPNet{IP: ip4}

	ip6 := net.IP{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6}
	addr6 := &net.IPNet{IP: ip6}

	testCases := []struct {
		name    string
		iface   netIface
		ipv     ipVersion
		want    []net.IP
		wantErr error
	}{{
		name:    "ipv4_success",
		iface:   &fakeIface{addrs: []net.Addr{addr4}, err: nil},
		ipv:     ipVersion4,
		want:    []net.IP{ip4},
		wantErr: nil,
	}, {
		name:    "ipv4_success_with_ipv6",
		iface:   &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		ipv:     ipVersion4,
		want:    []net.IP{ip4},
		wantErr: nil,
	}, {
		name:    "ipv4_error",
		iface:   &fakeIface{addrs: []net.Addr{addr4}, err: errTest},
		ipv:     ipVersion4,
		want:    nil,
		wantErr: errTest,
	}, {
		name:    "ipv6_success",
		iface:   &fakeIface{addrs: []net.Addr{addr6}, err: nil},
		ipv:     ipVersion6,
		want:    []net.IP{ip6},
		wantErr: nil,
	}, {
		name:    "ipv6_success_with_ipv4",
		iface:   &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		ipv:     ipVersion6,
		want:    []net.IP{ip6},
		wantErr: nil,
	}, {
		name:    "ipv6_error",
		iface:   &fakeIface{addrs: []net.Addr{addr6}, err: errTest},
		ipv:     ipVersion6,
		want:    nil,
		wantErr: errTest,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := ifaceIPAddrs(tc.iface, tc.ipv)
			require.True(t, errors.Is(gotErr, tc.wantErr))
			assert.Equal(t, tc.want, got)
		})
	}
}

type waitingFakeIface struct {
	addrs []net.Addr
	err   error
	n     int
}

// Addrs implements the netIface interface for *waitingFakeIface.
func (iface *waitingFakeIface) Addrs() (addrs []net.Addr, err error) {
	if iface.err != nil {
		return nil, iface.err
	}

	if iface.n == 0 {
		return iface.addrs, nil
	}

	iface.n--

	return nil, nil
}

func TestIfaceDNSIPAddrs(t *testing.T) {
	const errTest agherr.Error = "test error"

	ip4 := net.IP{1, 2, 3, 4}
	addr4 := &net.IPNet{IP: ip4}

	ip6 := net.IP{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6}
	addr6 := &net.IPNet{IP: ip6}

	testCases := []struct {
		name    string
		iface   netIface
		ipv     ipVersion
		want    []net.IP
		wantErr error
	}{{
		name:    "ipv4_success",
		iface:   &fakeIface{addrs: []net.Addr{addr4}, err: nil},
		ipv:     ipVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}, {
		name:    "ipv4_success_with_ipv6",
		iface:   &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		ipv:     ipVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}, {
		name:    "ipv4_error",
		iface:   &fakeIface{addrs: []net.Addr{addr4}, err: errTest},
		ipv:     ipVersion4,
		want:    nil,
		wantErr: errTest,
	}, {
		name:    "ipv4_wait",
		iface:   &waitingFakeIface{addrs: []net.Addr{addr4}, err: nil, n: 1},
		ipv:     ipVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}, {
		name:    "ipv6_success",
		iface:   &fakeIface{addrs: []net.Addr{addr6}, err: nil},
		ipv:     ipVersion6,
		want:    []net.IP{ip6, ip6},
		wantErr: nil,
	}, {
		name:    "ipv6_success_with_ipv4",
		iface:   &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		ipv:     ipVersion6,
		want:    []net.IP{ip6, ip6},
		wantErr: nil,
	}, {
		name:    "ipv6_error",
		iface:   &fakeIface{addrs: []net.Addr{addr6}, err: errTest},
		ipv:     ipVersion6,
		want:    nil,
		wantErr: errTest,
	}, {
		name:    "ipv6_wait",
		iface:   &waitingFakeIface{addrs: []net.Addr{addr6}, err: nil, n: 1},
		ipv:     ipVersion6,
		want:    []net.IP{ip6, ip6},
		wantErr: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := ifaceDNSIPAddrs(tc.iface, tc.ipv, 2, 0)
			require.True(t, errors.Is(gotErr, tc.wantErr))
			assert.Equal(t, tc.want, got)
		})
	}
}
