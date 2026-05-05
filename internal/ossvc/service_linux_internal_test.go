//go:build linux

package ossvc

import (
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
)

func TestSysvService_Install(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cmdErr     error
		sysSvcErr  error
		name       string
		wantErrMsg string
	}{{
		cmdErr:     nil,
		sysSvcErr:  nil,
		name:       "success",
		wantErrMsg: "",
	}, {
		cmdErr:     nil,
		sysSvcErr:  assert.AnError,
		name:       "sys_svc_error",
		wantErrMsg: assert.AnError.Error(),
	}, {
		cmdErr:     assert.AnError,
		sysSvcErr:  nil,
		name:       "cmd_error",
		wantErrMsg: `starting: ` + assert.AnError.Error() + `; stderr peek: ""; stdout peek: ""`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sysSvc := newTestService()
			sysSvc.OnInstall = func() (err error) {
				return tc.sysSvcErr
			}

			svc := &sysvService{
				cmdCons: newTestCmdConstructor(t, "", tc.cmdErr),
				Service: sysSvc,
				name:    testServiceName,
			}

			err := svc.Install()
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestSysvService_Uninstall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cmdErr     error
		sysSvcErr  error
		name       string
		wantErrMsg string
	}{{
		cmdErr:     nil,
		sysSvcErr:  nil,
		name:       "success",
		wantErrMsg: "",
	}, {
		cmdErr:     nil,
		sysSvcErr:  assert.AnError,
		name:       "sys_svc_error",
		wantErrMsg: assert.AnError.Error(),
	}, {
		cmdErr:     assert.AnError,
		sysSvcErr:  nil,
		name:       "cmd_error",
		wantErrMsg: `starting: ` + assert.AnError.Error() + `; stderr peek: ""; stdout peek: ""`,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sysSvc := newTestService()
			sysSvc.OnUninstall = func() (err error) {
				return tc.sysSvcErr
			}

			svc := &sysvService{
				cmdCons: newTestCmdConstructor(t, "", tc.cmdErr),
				Service: sysSvc,
				name:    testServiceName,
			}

			err := svc.Uninstall()
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestSystemdService_Status(t *testing.T) {
	t.Parallel()

	sysSvc := newTestService()

	testCases := []struct {
		cmdOutput  string
		name       string
		wantErrMsg string
		wantStatus service.Status
	}{{
		cmdOutput:  "LoadState=not-found\nActiveState=inactive\nSubState=dead\n",
		name:       "not_installed",
		wantErrMsg: "parsing command output: the service is not installed",
		wantStatus: service.StatusUnknown,
	}, {
		cmdOutput:  "LoadState=loaded\nActiveState=active\nSubState=running\n",
		name:       "running",
		wantErrMsg: "",
		wantStatus: service.StatusRunning,
	}, {
		cmdOutput:  "LoadState=loaded\nActiveState=inactive\nSubState=dead\n",
		name:       "stopped",
		wantErrMsg: "",
		wantStatus: service.StatusStopped,
	}, {
		cmdOutput:  "LoadState=loaded\nActiveState=activating\nSubState=auto-restart\n",
		name:       "restarting",
		wantErrMsg: "",
		wantStatus: statusRestartOnFail,
	}, {
		cmdOutput:  "LoadState=loaded\nActiveState=foo\nSubState=bar\n",
		name:       "unexpected_state",
		wantErrMsg: "parsing command output: unexpected state: LoadState=\"loaded\", ActiveState=\"foo\", SubState=\"bar\"",
		wantStatus: service.StatusUnknown,
	}, {
		cmdOutput:  "not_a_key_value_line\n",
		name:       "malformed_output",
		wantErrMsg: "parsing command output: unexpected line format: \"not_a_key_value_line\"",
		wantStatus: service.StatusUnknown,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &systemdService{
				cmdCons:  newTestCmdConstructor(t, tc.cmdOutput, nil),
				Service:  sysSvc,
				unitName: testServiceName,
			}

			s, err := svc.Status()
			assert.Equal(t, tc.wantStatus, s)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestSystemdService_Status_cmdError(t *testing.T) {
	t.Parallel()

	sysSvc := newTestService()

	svc := &systemdService{
		cmdCons:  newTestCmdConstructor(t, "", assert.AnError),
		Service:  sysSvc,
		unitName: testServiceName,
	}

	s, err := svc.Status()
	assert.Equal(t, service.StatusUnknown, s)
	testutil.AssertErrorMsg(t, "executing command: starting: "+assert.AnError.Error(), err)
}
