package aghnet

import (
	"net"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIface is a stub implementation of aghnet.NetIface to simplify testing.
type fakeIface struct {
	err   error
	addrs []net.Addr
}

// Addrs implements the NetIface interface for *fakeIface.
func (iface *fakeIface) Addrs() (addrs []net.Addr, err error) {
	if iface.err != nil {
		return nil, iface.err
	}

	return iface.addrs, nil
}

func TestIfaceIPAddrs(t *testing.T) {
	const errTest errors.Error = "test error"

	ip4 := net.IP{1, 2, 3, 4}
	addr4 := &net.IPNet{IP: ip4}

	ip6 := net.IP{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6}
	addr6 := &net.IPNet{IP: ip6}

	testCases := []struct {
		iface      NetIface
		name       string
		wantErrMsg string
		want       []net.IP
		ipv        IPVersion
	}{{
		iface:      &fakeIface{addrs: []net.Addr{addr4}, err: nil},
		name:       "ipv4_success",
		wantErrMsg: "",
		want:       []net.IP{ip4},
		ipv:        IPVersion4,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		name:       "ipv4_success_with_ipv6",
		wantErrMsg: "",
		want:       []net.IP{ip4},
		ipv:        IPVersion4,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{addr4}, err: errTest},
		name:       "ipv4_error",
		wantErrMsg: errTest.Error(),
		want:       nil,
		ipv:        IPVersion4,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{addr6}, err: nil},
		name:       "ipv6_success",
		wantErrMsg: "",
		want:       []net.IP{ip6},
		ipv:        IPVersion6,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		name:       "ipv6_success_with_ipv4",
		wantErrMsg: "",
		want:       []net.IP{ip6},
		ipv:        IPVersion6,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{addr6}, err: errTest},
		name:       "ipv6_error",
		wantErrMsg: errTest.Error(),
		want:       nil,
		ipv:        IPVersion6,
	}, {
		iface:      &fakeIface{addrs: nil, err: nil},
		name:       "bad_proto",
		wantErrMsg: "invalid ip version 10",
		want:       nil,
		ipv:        IPVersion6 + IPVersion4,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{&net.IPAddr{IP: ip4}}, err: nil},
		name:       "ipaddr_v4",
		wantErrMsg: "",
		want:       []net.IP{ip4},
		ipv:        IPVersion4,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{&net.IPAddr{IP: ip6, Zone: ""}}, err: nil},
		name:       "ipaddr_v6",
		wantErrMsg: "",
		want:       []net.IP{ip6},
		ipv:        IPVersion6,
	}, {
		iface:      &fakeIface{addrs: []net.Addr{&net.UnixAddr{}}, err: nil},
		name:       "non-ipv4",
		wantErrMsg: "",
		want:       nil,
		ipv:        IPVersion4,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IfaceIPAddrs(tc.iface, tc.ipv)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.want, got)
		})
	}
}

type waitingFakeIface struct {
	err   error
	addrs []net.Addr
	n     int
}

// Addrs implements the NetIface interface for *waitingFakeIface.
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
	const errTest errors.Error = "test error"

	ip4 := net.IP{1, 2, 3, 4}
	addr4 := &net.IPNet{IP: ip4}

	ip6 := net.IP{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6}
	addr6 := &net.IPNet{IP: ip6}

	testCases := []struct {
		iface   NetIface
		wantErr error
		name    string
		want    []net.IP
		ipv     IPVersion
	}{{
		name:    "ipv4_success",
		iface:   &fakeIface{addrs: []net.Addr{addr4}, err: nil},
		ipv:     IPVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}, {
		name:    "ipv4_success_with_ipv6",
		iface:   &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		ipv:     IPVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}, {
		name:    "ipv4_error",
		iface:   &fakeIface{addrs: []net.Addr{addr4}, err: errTest},
		ipv:     IPVersion4,
		want:    nil,
		wantErr: errTest,
	}, {
		name:    "ipv4_wait",
		iface:   &waitingFakeIface{addrs: []net.Addr{addr4}, err: nil, n: 1},
		ipv:     IPVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}, {
		name:    "ipv6_success",
		iface:   &fakeIface{addrs: []net.Addr{addr6}, err: nil},
		ipv:     IPVersion6,
		want:    []net.IP{ip6, ip6},
		wantErr: nil,
	}, {
		name:    "ipv6_success_with_ipv4",
		iface:   &fakeIface{addrs: []net.Addr{addr6, addr4}, err: nil},
		ipv:     IPVersion6,
		want:    []net.IP{ip6, ip6},
		wantErr: nil,
	}, {
		name:    "ipv6_error",
		iface:   &fakeIface{addrs: []net.Addr{addr6}, err: errTest},
		ipv:     IPVersion6,
		want:    nil,
		wantErr: errTest,
	}, {
		name:    "ipv6_wait",
		iface:   &waitingFakeIface{addrs: []net.Addr{addr6}, err: nil, n: 1},
		ipv:     IPVersion6,
		want:    []net.IP{ip6, ip6},
		wantErr: nil,
	}, {
		name:    "empty",
		iface:   &fakeIface{addrs: nil, err: nil},
		ipv:     IPVersion4,
		want:    nil,
		wantErr: nil,
	}, {
		name:    "many",
		iface:   &fakeIface{addrs: []net.Addr{addr4, addr4}},
		ipv:     IPVersion4,
		want:    []net.IP{ip4, ip4},
		wantErr: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IfaceDNSIPAddrs(tc.iface, tc.ipv, 2, 0)
			require.ErrorIs(t, err, tc.wantErr)

			assert.Equal(t, tc.want, got)
		})
	}
}
