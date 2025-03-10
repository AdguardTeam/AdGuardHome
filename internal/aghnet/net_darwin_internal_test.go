package aghnet

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakefs"
	"github.com/stretchr/testify/assert"
)

func TestIfaceHasStaticIP(t *testing.T) {
	testCases := []struct {
		name       string
		shell      mapShell
		ifaceName  string
		wantHas    assert.BoolAssertionFunc
		wantErrMsg string
	}{{
		name: "success",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  nil,
				out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
		},
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: ``,
	}, {
		name: "success_static",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err: nil,
				out: "Manual Configuration\nIP address: 1.2.3.4\n" +
					"Subnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
		},
		ifaceName:  "en0",
		wantHas:    assert.True,
		wantErrMsg: ``,
	}, {
		name: "reports_error",
		shell: theOnlyCmd(
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
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  errors.Error("can't get"),
				out:  ``,
				code: 0,
			},
		},
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: `can't get`,
	}, {
		name: "port_bad_output",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  nil,
				out:  "nothing meaningful",
				code: 0,
			},
		},
		ifaceName:  "en0",
		wantHas:    assert.False,
		wantErrMsg: `could not find hardware port info`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substShell(t, tc.shell.RunCmd)

			has, err := IfaceHasStaticIP(tc.ifaceName)
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
		OnOpen: func(name string) (fs.File, error) { panic("not implemented") },
	}

	testCases := []struct {
		name       string
		shell      mapShell
		fsys       fs.FS
		wantErrMsg string
	}{{
		name: "success",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  nil,
				out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
			"networksetup -setdnsservers hwport 1.1.1.1": {
				err:  nil,
				out:  "",
				code: 0,
			},
			"networksetup -setmanual hwport 1.2.3.4 255.255.255.0 1.2.3.1": {
				err:  nil,
				out:  "",
				code: 0,
			},
		},
		fsys:       succFsys,
		wantErrMsg: ``,
	}, {
		name: "static_already",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err: nil,
				out: "Manual Configuration\nIP address: 1.2.3.4\n" +
					"Subnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
		},
		fsys:       panicFsys,
		wantErrMsg: `ip address is already static`,
	}, {
		name: "reports_error",
		shell: theOnlyCmd(
			"networksetup -listallhardwareports",
			0,
			"",
			errors.Error("can't list"),
		),
		fsys:       panicFsys,
		wantErrMsg: `could not find hardware port for en0`,
	}, {
		name: "resolv_conf_error",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  nil,
				out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
		},
		fsys: fstest.MapFS{
			"etc/resolv.conf": &fstest.MapFile{
				Data: []byte("this resolv.conf is invalid"),
			},
		},
		wantErrMsg: `found no dns servers in etc/resolv.conf`,
	}, {
		name: "set_dns_error",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  nil,
				out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
			"networksetup -setdnsservers hwport 1.1.1.1": {
				err:  errors.Error("can't set"),
				out:  "",
				code: 0,
			},
		},
		fsys:       succFsys,
		wantErrMsg: `can't set`,
	}, {
		name: "set_manual_error",
		shell: mapShell{
			"networksetup -listallhardwareports": {
				err:  nil,
				out:  "Hardware Port: hwport\nDevice: en0\n",
				code: 0,
			},
			"networksetup -getinfo hwport": {
				err:  nil,
				out:  "IP address: 1.2.3.4\nSubnet mask: 255.255.255.0\nRouter: 1.2.3.1\n",
				code: 0,
			},
			"networksetup -setdnsservers hwport 1.1.1.1": {
				err:  nil,
				out:  "",
				code: 0,
			},
			"networksetup -setmanual hwport 1.2.3.4 255.255.255.0 1.2.3.1": {
				err:  errors.Error("can't set"),
				out:  "",
				code: 0,
			},
		},
		fsys:       succFsys,
		wantErrMsg: `can't set`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substShell(t, tc.shell.RunCmd)
			substRootDirFS(t, tc.fsys)

			err := IfaceSetStaticIP("en0")
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}
