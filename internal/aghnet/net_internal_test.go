package aghnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// substRootDirFS replaces the aghos.RootDirFS function used throughout the
// package with fsys for tests ran under t.
func substRootDirFS(t testing.TB, fsys fs.FS) {
	t.Helper()

	prev := rootDirFS
	t.Cleanup(func() { rootDirFS = prev })
	rootDirFS = fsys
}

// RunCmdFunc is the signature of aghos.RunCommand function.
type RunCmdFunc func(cmd string, args ...string) (code int, out []byte, err error)

// substShell replaces the the aghos.RunCommand function used throughout the
// package with rc for tests ran under t.
func substShell(t testing.TB, rc RunCmdFunc) {
	t.Helper()

	prev := aghosRunCommand
	t.Cleanup(func() { aghosRunCommand = prev })
	aghosRunCommand = rc
}

// mapShell is a substitution of aghos.RunCommand that maps the command to it's
// execution result.  It's only needed to simplify testing.
//
// TODO(e.burkov):  Perhaps put all the shell interactions behind an interface.
type mapShell map[string]struct {
	err  error
	out  string
	code int
}

// theOnlyCmd returns mapShell that only handles a single command and arguments
// combination from cmd.
func theOnlyCmd(cmd string, code int, out string, err error) (s mapShell) {
	return mapShell{cmd: {code: code, out: out, err: err}}
}

// RunCmd is a RunCmdFunc handled by s.
func (s mapShell) RunCmd(cmd string, args ...string) (code int, out []byte, err error) {
	key := strings.Join(append([]string{cmd}, args...), " ")
	ret, ok := s[key]
	if !ok {
		return 0, nil, fmt.Errorf("unexpected shell command %q", key)
	}

	return ret.code, []byte(ret.out), ret.err
}

// ifaceAddrsFunc is the signature of net.InterfaceAddrs function.
type ifaceAddrsFunc func() (ifaces []net.Addr, err error)

// substNetInterfaceAddrs replaces the the net.InterfaceAddrs function used
// throughout the package with f for tests ran under t.
func substNetInterfaceAddrs(t *testing.T, f ifaceAddrsFunc) {
	t.Helper()

	prev := netInterfaceAddrs
	t.Cleanup(func() { netInterfaceAddrs = prev })
	netInterfaceAddrs = f
}

func TestGatewayIP(t *testing.T) {
	const ifaceName = "ifaceName"
	const cmd = "ip route show dev " + ifaceName

	testCases := []struct {
		shell mapShell
		want  netip.Addr
		name  string
	}{{
		shell: theOnlyCmd(cmd, 0, `default via 1.2.3.4 onlink`, nil),
		want:  netip.MustParseAddr("1.2.3.4"),
		name:  "success_v4",
	}, {
		shell: theOnlyCmd(cmd, 0, `default via ::ffff onlink`, nil),
		want:  netip.MustParseAddr("::ffff"),
		name:  "success_v6",
	}, {
		shell: theOnlyCmd(cmd, 0, `non-default via 1.2.3.4 onlink`, nil),
		want:  netip.Addr{},
		name:  "bad_output",
	}, {
		shell: theOnlyCmd(cmd, 0, "", errors.Error("can't run command")),
		want:  netip.Addr{},
		name:  "err_runcmd",
	}, {
		shell: theOnlyCmd(cmd, 1, "", nil),
		want:  netip.Addr{},
		name:  "bad_code",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substShell(t, tc.shell.RunCmd)

			assert.Equal(t, tc.want, GatewayIP(ifaceName))
		})
	}
}

func TestInterfaceByIP(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.NoError(t, err)
	require.NotEmpty(t, ifaces)

	for _, iface := range ifaces {
		t.Run(iface.Name, func(t *testing.T) {
			require.NotEmpty(t, iface.Addresses)

			for _, ip := range iface.Addresses {
				ifaceName := InterfaceByIP(ip)
				require.Equal(t, iface.Name, ifaceName)
			}
		})
	}
}

func TestBroadcastFromIPNet(t *testing.T) {
	known4 := netip.MustParseAddr("192.168.0.1")
	fullBroadcast4 := netip.MustParseAddr("255.255.255.255")

	known6 := netip.MustParseAddr("102:304:506:708:90a:b0c:d0e:f10")

	testCases := []struct {
		pref netip.Prefix
		want netip.Addr
		name string
	}{{
		pref: netip.PrefixFrom(known4, 0),
		want: fullBroadcast4,
		name: "full",
	}, {
		pref: netip.PrefixFrom(known4, 20),
		want: netip.MustParseAddr("192.168.15.255"),
		name: "full",
	}, {
		pref: netip.PrefixFrom(known6, netutil.IPv6BitLen),
		want: known6,
		name: "ipv6_no_mask",
	}, {
		pref: netip.PrefixFrom(known4, netutil.IPv4BitLen),
		want: known4,
		name: "ipv4_no_mask",
	}, {
		pref: netip.PrefixFrom(netip.IPv4Unspecified(), 0),
		want: fullBroadcast4,
		name: "unspecified",
	}, {
		pref: netip.Prefix{},
		want: netip.Addr{},
		name: "invalid",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, BroadcastFromPref(tc.pref))
		})
	}
}

func TestCheckPort(t *testing.T) {
	laddr := netip.AddrPortFrom(netutil.IPv4Localhost(), 0)

	t.Run("tcp_bound", func(t *testing.T) {
		l, err := net.Listen("tcp", laddr.String())
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, l.Close)

		ipp := testutil.RequireTypeAssert[*net.TCPAddr](t, l.Addr()).AddrPort()
		require.Equal(t, laddr.Addr(), ipp.Addr())
		require.NotZero(t, ipp.Port())

		err = CheckPort("tcp", ipp)
		target := &net.OpError{}
		require.ErrorAs(t, err, &target)

		assert.Equal(t, "listen", target.Op)
	})

	t.Run("udp_bound", func(t *testing.T) {
		conn, err := net.ListenPacket("udp", laddr.String())
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, conn.Close)

		ipp := testutil.RequireTypeAssert[*net.UDPAddr](t, conn.LocalAddr()).AddrPort()
		require.Equal(t, laddr.Addr(), ipp.Addr())
		require.NotZero(t, ipp.Port())

		err = CheckPort("udp", ipp)
		target := &net.OpError{}
		require.ErrorAs(t, err, &target)

		assert.Equal(t, "listen", target.Op)
	})

	t.Run("bad_network", func(t *testing.T) {
		err := CheckPort("bad_network", netip.AddrPortFrom(netip.Addr{}, 0))
		assert.NoError(t, err)
	})

	t.Run("can_bind", func(t *testing.T) {
		err := CheckPort("udp", netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
		assert.NoError(t, err)
	})
}

func TestCollectAllIfacesAddrs(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		addrs      []net.Addr
		wantAddrs  []netip.Addr
	}{{
		name:       "success",
		wantErrMsg: ``,
		addrs: []net.Addr{&net.IPNet{
			IP:   net.IP{1, 2, 3, 4},
			Mask: net.CIDRMask(24, netutil.IPv4BitLen),
		}, &net.IPNet{
			IP:   net.IP{4, 3, 2, 1},
			Mask: net.CIDRMask(16, netutil.IPv4BitLen),
		}},
		wantAddrs: []netip.Addr{
			netip.MustParseAddr("1.2.3.4"),
			netip.MustParseAddr("4.3.2.1"),
		},
	}, {
		name:       "not_cidr",
		wantErrMsg: `netip.ParsePrefix("1.2.3.4"): no '/'`,
		addrs: []net.Addr{&net.IPAddr{
			IP: net.IP{1, 2, 3, 4},
		}},
		wantAddrs: nil,
	}, {
		name:       "empty",
		wantErrMsg: ``,
		addrs:      []net.Addr{},
		wantAddrs:  nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substNetInterfaceAddrs(t, func() ([]net.Addr, error) { return tc.addrs, nil })

			addrs, err := CollectAllIfacesAddrs()
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			assert.Equal(t, tc.wantAddrs, addrs)
		})
	}

	t.Run("internal_error", func(t *testing.T) {
		const errAddrs errors.Error = "can't get addresses"

		substNetInterfaceAddrs(t, func() ([]net.Addr, error) { return nil, errAddrs })

		_, err := CollectAllIfacesAddrs()
		assert.ErrorIs(t, err, errAddrs)
	})
}

func TestIsAddrInUse(t *testing.T) {
	t.Run("addr_in_use", func(t *testing.T) {
		l, err := net.Listen("tcp", "0.0.0.0:0")
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, l.Close)

		_, err = net.Listen(l.Addr().Network(), l.Addr().String())
		assert.True(t, IsAddrInUse(err))
	})

	t.Run("another", func(t *testing.T) {
		const anotherErr errors.Error = "not addr in use"

		assert.False(t, IsAddrInUse(anotherErr))
	})
}

func TestNetInterface_MarshalJSON(t *testing.T) {
	const want = `{` +
		`"hardware_address":"aa:bb:cc:dd:ee:ff",` +
		`"flags":"up|multicast",` +
		`"ip_addresses":["1.2.3.4","aaaa::1"],` +
		`"name":"iface0",` +
		`"mtu":1500` +
		`}` + "\n"

	ip4, ok := netip.AddrFromSlice([]byte{1, 2, 3, 4})
	require.True(t, ok)

	ip6, ok := netip.AddrFromSlice([]byte{0xAA, 0xAA, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	require.True(t, ok)

	net4 := netip.PrefixFrom(ip4, 24)
	net6 := netip.PrefixFrom(ip6, 8)

	iface := &NetInterface{
		Addresses:    []netip.Addr{ip4, ip6},
		Subnets:      []netip.Prefix{net4, net6},
		Name:         "iface0",
		HardwareAddr: net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		Flags:        net.FlagUp | net.FlagMulticast,
		MTU:          1500,
	}

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(iface)
	require.NoError(t, err)

	assert.Equal(t, want, b.String())
}
