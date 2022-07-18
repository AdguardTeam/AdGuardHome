package aghnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

// testdata is the filesystem containing data for testing the package.
var testdata fs.FS = os.DirFS("./testdata")

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
		name  string
		shell mapShell
		want  net.IP
	}{{
		name:  "success_v4",
		shell: theOnlyCmd(cmd, 0, `default via 1.2.3.4 onlink`, nil),
		want:  net.IP{1, 2, 3, 4}.To16(),
	}, {
		name:  "success_v6",
		shell: theOnlyCmd(cmd, 0, `default via ::ffff onlink`, nil),
		want: net.IP{
			0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0xFF, 0xFF,
		},
	}, {
		name:  "bad_output",
		shell: theOnlyCmd(cmd, 0, `non-default via 1.2.3.4 onlink`, nil),
		want:  nil,
	}, {
		name:  "err_runcmd",
		shell: theOnlyCmd(cmd, 0, "", errors.Error("can't run command")),
		want:  nil,
	}, {
		name:  "bad_code",
		shell: theOnlyCmd(cmd, 1, "", nil),
		want:  nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substShell(t, tc.shell.RunCmd)

			assert.Equal(t, tc.want, GatewayIP(ifaceName))
		})
	}
}

func TestGetInterfaceByIP(t *testing.T) {
	ifaces, err := GetValidNetInterfacesForWeb()
	require.NoError(t, err)
	require.NotEmpty(t, ifaces)

	for _, iface := range ifaces {
		t.Run(iface.Name, func(t *testing.T) {
			require.NotEmpty(t, iface.Addresses)

			for _, ip := range iface.Addresses {
				ifaceName := GetInterfaceByIP(ip)
				require.Equal(t, iface.Name, ifaceName)
			}
		})
	}
}

func TestBroadcastFromIPNet(t *testing.T) {
	known6 := net.IP{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	testCases := []struct {
		name   string
		subnet *net.IPNet
		want   net.IP
	}{{
		name: "full",
		subnet: &net.IPNet{
			IP:   net.IP{192, 168, 0, 1},
			Mask: net.IPMask{255, 255, 15, 0},
		},
		want: net.IP{192, 168, 240, 255},
	}, {
		name: "ipv6_no_mask",
		subnet: &net.IPNet{
			IP: known6,
		},
		want: known6,
	}, {
		name: "ipv4_no_mask",
		subnet: &net.IPNet{
			IP: net.IP{192, 168, 1, 2},
		},
		want: net.IP{192, 168, 1, 255},
	}, {
		name: "unspecified",
		subnet: &net.IPNet{
			IP:   net.IP{0, 0, 0, 0},
			Mask: net.IPMask{0, 0, 0, 0},
		},
		want: net.IPv4bcast,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bc := BroadcastFromIPNet(tc.subnet)
			assert.True(t, bc.Equal(tc.want), bc)
		})
	}
}

func TestCheckPort(t *testing.T) {
	t.Run("tcp_bound", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")

		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, l.Close)

		ip, err := netip.ParseAddrPort(l.Addr().String())
		require.Nil(t, err)
		require.NotNil(t, ip)
		require.True(t, ip.IsValid())
		require.NotZero(t, ip.Port())

		err = CheckPort("tcp", ip.Addr(), int(ip.Port()))
		target := &net.OpError{}
		require.ErrorAs(t, err, &target)
		assert.Equal(t, "listen", target.Op)
	})

	t.Run("udp_bound", func(t *testing.T) {
		conn, err := net.ListenPacket("udp", "127.0.0.1:")
		require.NoError(t, err)
		testutil.CleanupAndRequireSuccess(t, conn.Close)

		ip, err := netip.ParseAddrPort(conn.LocalAddr().String())
		require.Nil(t, err)
		require.NotNil(t, ip)
		require.True(t, ip.IsValid())
		require.NotZero(t, ip.Port())

		err = CheckPort("udp", ip.Addr(), int(ip.Port()))
		target := &net.OpError{}
		require.ErrorAs(t, err, &target)

		assert.Equal(t, "listen", target.Op)
	})

	t.Run("bad_network", func(t *testing.T) {
		err := CheckPort("bad_network", netip.Addr{}, 0)
		assert.NoError(t, err)
	})

	t.Run("can_bind", func(t *testing.T) {
		err := CheckPort("udp", netip.IPv4Unspecified(), 0)
		assert.NoError(t, err)
	})
}

func TestCollectAllIfacesAddrs(t *testing.T) {
	testCases := []struct {
		name       string
		wantErrMsg string
		addrs      []net.Addr
		wantAddrs  []string
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
		wantAddrs: []string{"1.2.3.4", "4.3.2.1"},
	}, {
		name:       "not_cidr",
		wantErrMsg: `parsing cidr: invalid CIDR address: 1.2.3.4`,
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
		const wantErrMsg string = `getting interfaces addresses: ` + string(errAddrs)

		substNetInterfaceAddrs(t, func() ([]net.Addr, error) { return nil, errAddrs })

		_, err := CollectAllIfacesAddrs()
		testutil.AssertErrorMsg(t, wantErrMsg, err)
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

	ip4, ip6 := netip.AddrFrom4([4]byte{1, 2, 3, 4}), netip.AddrFrom16([16]byte{0xAA, 0xAA, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	v4Subnet, v6Subnet := netip.PrefixFrom(ip4, 24), netip.PrefixFrom(ip6, 8)

	iface := &NetInterface{
		Addresses:    []netip.Addr{ip4, ip6},
		Subnets:      []*netip.Prefix{&v4Subnet, &v6Subnet},
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
