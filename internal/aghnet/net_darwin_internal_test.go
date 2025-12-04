//go:build darwin

package aghnet

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakeio/fakefs"
	"github.com/stretchr/testify/assert"
)

func TestIfaceHasStaticIP(t *testing.T) {
	testCases := []struct {
		name       string
		cmdCons    executil.CommandConstructor
		ifaceName  string
		wantHas    assert.BoolAssertionFunc
		wantErrMsg string
	}{{
		name: "success",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  nil,
			Out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		}),
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: ``,
	}, {
		name: "success_static",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd: "networksetup -getinfo hwport",
			Err: nil,
			Out: "Manual Configuration\nIP address: 1.2.3.4\n" +
				"Subnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		}),
		ifaceName:  "en0",
		wantHas:    assert.True,
		wantErrMsg: ``,
	}, {
		name: "reports_error",
		cmdCons: agh.NewCommandConstructor(
			"networksetup -listallhardwareports",
			0,
			"",
			errors.Error("can't list"),
		),
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: `could not find hardware port for en0`,
	}, {
		name: "port_error",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  errors.Error("can't get"),
			Out:  ``,
			Code: 0,
		}),
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: `command "networksetup" failed: running: can't get: `,
	}, {
		name: "port_bad_output",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  nil,
			Out:  "nothing meaningful",
			Code: 0,
		}),
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: `could not find hardware port info`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.ContextWithTimeout(t, testTimeout)
			has, err := IfaceHasStaticIP(ctx, tc.cmdCons, tc.ifaceName)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)

			tc.wantHas(t, has)
		})
	}
}

func TestIfaceSetStaticIP(t *testing.T) {
	succFsys := fstest.MapFS{
		"etc/resolv.conf": &fstest.MapFile{
			Data: []byte(`nameserver 1.1.1.1`),
		},
	}
	panicFsys := &fakefs.FS{
		OnOpen: func(name string) (_ fs.File, _ error) { panic(testutil.UnexpectedCall(name)) },
	}

	testCases := []struct {
		name       string
		cmdCons    executil.CommandConstructor
		fsys       fs.FS
		wantErrMsg string
	}{{
		name: "success",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  nil,
			Out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -setdnsservers hwport 1.1.1.1",
			Err:  nil,
			Out:  "",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -setmanual hwport 1.2.3.4 255.255.255.0 1.2.3.1",
			Err:  nil,
			Out:  "",
			Code: 0,
		}),
		fsys:       succFsys,
		wantErrMsg: ``,
	}, {
		name: "static_already",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd: "networksetup -getinfo hwport",
			Err: nil,
			Out: "Manual Configuration\nIP address: 1.2.3.4\n" +
				"Subnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		}),
		fsys:       panicFsys,
		wantErrMsg: `ip address is already static`,
	}, {
		name: "reports_error",
		cmdCons: agh.NewCommandConstructor(
			"networksetup -listallhardwareports",
			0,
			"",
			errors.Error("can't list"),
		),
		fsys:       panicFsys,
		wantErrMsg: `could not find hardware port for en0`,
	}, {
		name: "resolv_conf_error",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  nil,
			Out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		},
		),
		fsys: fstest.MapFS{
			"etc/resolv.conf": &fstest.MapFile{
				Data: []byte("this resolv.conf is invalid"),
			},
		},
		wantErrMsg: `found no dns servers in etc/resolv.conf`,
	}, {
		name: "set_dns_error",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  nil,
			Out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -setdnsservers hwport 1.1.1.1",
			Err:  errors.Error("can't set"),
			Out:  "",
			Code: 0,
		}),
		fsys:       succFsys,
		wantErrMsg: `command "networksetup" failed: running: can't set: `,
	}, {
		name: "set_manual_error",
		cmdCons: agh.NewMultipleCommandConstructor(agh.ExternalCommand{
			Cmd:  "networksetup -listallhardwareports",
			Err:  nil,
			Out:  "Hardware Port: hwport\nDevice: en0\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -getinfo hwport",
			Err:  nil,
			Out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -setdnsservers hwport 1.1.1.1",
			Err:  nil,
			Out:  "",
			Code: 0,
		}, agh.ExternalCommand{
			Cmd:  "networksetup -setmanual hwport 1.2.3.4 255.255.255.0 1.2.3.1",
			Err:  errors.Error("can't set"),
			Out:  "",
			Code: 0,
		}),
		fsys:       succFsys,
		wantErrMsg: `command "networksetup" failed: running: can't set: `,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substRootDirFS(t, tc.fsys)

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := IfaceSetStaticIP(ctx, testLogger, tc.cmdCons, "en0")
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}
