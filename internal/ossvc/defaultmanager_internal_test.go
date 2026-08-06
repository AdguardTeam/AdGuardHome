package ossvc

import (
	"fmt"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
)

func TestManager_Status(t *testing.T) {
	const invalidSvcStatus = statusRestartOnFail + 1

	svc := newTestServiceWithSystem(t)

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     aghos.IsOpenWrt(),
		isUnixSystemV: false,
	}

	testCases := []struct {
		onStatus       func() (s service.Status, err error)
		name           string
		wantErrMessage string
		wantStatus     Status
	}{{
		onStatus: func() (s service.Status, err error) {
			return service.StatusUnknown, service.ErrNotInstalled
		},
		name:           "not_installed",
		wantStatus:     StatusNotInstalled,
		wantErrMessage: "",
	}, {
		onStatus: func() (s service.Status, err error) {
			return service.StatusRunning, assert.AnError
		},
		name:           "error",
		wantStatus:     "",
		wantErrMessage: "getting service status: " + assert.AnError.Error(),
	}, {
		onStatus: func() (s service.Status, err error) {
			return service.StatusRunning, nil
		},
		name:           "running",
		wantStatus:     StatusRunning,
		wantErrMessage: "",
	}, {
		onStatus: func() (s service.Status, err error) {
			return service.StatusStopped, nil
		},
		name:           "stopped",
		wantStatus:     StatusStopped,
		wantErrMessage: "",
	}, {
		onStatus: func() (s service.Status, err error) {
			return statusRestartOnFail, nil
		},
		name:           "restart_on_fail",
		wantStatus:     StatusRestartOnFail,
		wantErrMessage: "",
	}, {
		onStatus: func() (s service.Status, err error) {
			return invalidSvcStatus, nil
		},
		name:       "invalid_status",
		wantStatus: "",
		wantErrMessage: fmt.Sprintf(
			"service status: %v: %v",
			errors.ErrBadEnumValue,
			invalidSvcStatus,
		),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc.OnStatus = tc.onStatus

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			status, err := m.Status(ctx, testServiceName)
			assert.Equal(t, tc.wantStatus, status)
			testutil.AssertErrorMsg(t, tc.wantErrMessage, err)
		})
	}
}

func TestManager_Status_unixSystemV(t *testing.T) {
	svc := newTestServiceWithSystem(t)
	svc.OnStatus = func() (s service.Status, err error) {
		return service.StatusUnknown, assert.AnError
	}

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     aghos.IsOpenWrt(),
		isUnixSystemV: true,
	}

	testCases := []struct {
		cmdCons    executil.CommandConstructor
		name       string
		wantStatus Status
	}{{
		cmdCons:    newTestCmdConstructor(t, "", nil),
		name:       "running",
		wantStatus: StatusRunning,
	}, {
		cmdCons:    newTestCmdConstructor(t, "", assert.AnError),
		name:       "stopped",
		wantStatus: StatusStopped,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m.cmdCons = tc.cmdCons

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			status, err := m.Status(ctx, testServiceName)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantStatus, status)
		})
	}
}

func TestManager_Install(t *testing.T) {
	svc := newTestServiceWithSystem(t)

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     false,
		isUnixSystemV: false,
	}

	testCases := []struct {
		installErr error
		name       string
		wantErrMsg string
	}{{
		installErr: nil,
		name:       "success",
		wantErrMsg: "",
	}, {
		installErr: assert.AnError,
		name:       "error",
		wantErrMsg: "installing service: " + assert.AnError.Error(),
	}}

	action := &ActionInstall{ServiceName: testServiceName}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc.OnInstall = func() (err error) {
				return tc.installErr
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := m.Perform(ctx, action)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestManager_Install_unixSystemV(t *testing.T) {
	svc := newTestServiceWithSystem(t)
	svc.OnInstall = func() (err error) {
		return nil
	}

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     aghos.IsOpenWrt(),
		isUnixSystemV: true,
	}

	testCases := []struct {
		cmdCons    executil.CommandConstructor
		name       string
		wantErrMsg string
	}{{
		cmdCons:    newTestCmdConstructor(t, "", nil),
		name:       "success",
		wantErrMsg: "",
	}, {
		cmdCons:    newTestCmdConstructor(t, "", assert.AnError),
		name:       "error",
		wantErrMsg: "",
	}}

	action := &ActionInstall{ServiceName: testServiceName}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m.cmdCons = tc.cmdCons

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := m.Perform(ctx, action)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestManager_Restart(t *testing.T) {
	svc := newTestServiceWithSystem(t)

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     false,
		isUnixSystemV: false,
	}

	testCases := []struct {
		cmdCons       executil.CommandConstructor
		restartErr    error
		name          string
		wantErrMsg    string
		isUnixSystemV bool
	}{{
		cmdCons:       executil.EmptyCommandConstructor{},
		restartErr:    nil,
		name:          "success",
		wantErrMsg:    "",
		isUnixSystemV: false,
	}, {
		cmdCons:       executil.EmptyCommandConstructor{},
		restartErr:    assert.AnError,
		name:          "error",
		wantErrMsg:    assert.AnError.Error(),
		isUnixSystemV: false,
	}, {
		cmdCons:       newTestCmdConstructor(t, "", nil),
		restartErr:    assert.AnError,
		name:          "unix_systemv_restart",
		wantErrMsg:    assert.AnError.Error(),
		isUnixSystemV: true,
	}, {
		cmdCons:    newTestCmdConstructor(t, "", errors.Error("initd_test")),
		restartErr: assert.AnError,
		name:       "unix_systemv_restart_error",
		wantErrMsg: assert.AnError.Error() + ` (restarting via init.d: starting: initd_test;` +
			` stderr peek: ""; stdout peek: "")`,
		isUnixSystemV: true,
	}}

	action := &ActionRestart{ServiceName: testServiceName}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m.cmdCons = tc.cmdCons
			m.isUnixSystemV = tc.isUnixSystemV

			svc.OnRestart = func() (err error) {
				return tc.restartErr
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := m.Perform(ctx, action)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestManager_Stop(t *testing.T) {
	svc := newTestServiceWithSystem(t)

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     false,
		isUnixSystemV: false,
	}

	testCases := []struct {
		cmdCons       executil.CommandConstructor
		stopErr       error
		name          string
		wantErrMsg    string
		isUnixSystemV bool
	}{{
		cmdCons:       executil.EmptyCommandConstructor{},
		stopErr:       nil,
		name:          "success",
		wantErrMsg:    "",
		isUnixSystemV: false,
	}, {
		cmdCons:       executil.EmptyCommandConstructor{},
		stopErr:       assert.AnError,
		name:          "error",
		wantErrMsg:    assert.AnError.Error(),
		isUnixSystemV: false,
	}, {
		cmdCons:       newTestCmdConstructor(t, "", nil),
		stopErr:       assert.AnError,
		name:          "unix_systemv_stop",
		wantErrMsg:    assert.AnError.Error(),
		isUnixSystemV: true,
	}, {
		cmdCons: newTestCmdConstructor(t, "", errors.Error("initd_test")),
		stopErr: assert.AnError,
		name:    "unix_systemv_stop_error",
		wantErrMsg: assert.AnError.Error() + ` (stopping via init.d: starting: initd_test;` +
			` stderr peek: ""; stdout peek: "")`,
		isUnixSystemV: true,
	}}

	action := &ActionStop{ServiceName: testServiceName}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m.cmdCons = tc.cmdCons
			m.isUnixSystemV = tc.isUnixSystemV

			svc.OnStop = func() (err error) {
				return tc.stopErr
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := m.Perform(ctx, action)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestManager_Uninstall(t *testing.T) {
	svc := newTestServiceWithSystem(t)

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     false,
		isUnixSystemV: false,
	}

	testCases := []struct {
		stopErr      error
		uninstallErr error
		name         string
		wantErrMsg   string
	}{{
		stopErr:      nil,
		uninstallErr: nil,
		name:         "success",
		wantErrMsg:   "",
	}, {
		stopErr:      errors.Error("stop_test"),
		uninstallErr: assert.AnError,
		name:         "error_stop",
		wantErrMsg:   "uninstalling service: " + assert.AnError.Error(),
	}, {
		stopErr:      nil,
		uninstallErr: assert.AnError,
		name:         "error_uninstall",
		wantErrMsg:   "uninstalling service: " + assert.AnError.Error(),
	}}

	action := &ActionUninstall{ServiceName: testServiceName}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc.OnStop = func() (err error) {
				return tc.stopErr
			}
			svc.OnUninstall = func() (err error) {
				return tc.uninstallErr
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := m.Perform(ctx, action)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestManager_Uninstall_openWrt(t *testing.T) {
	svc := newTestServiceWithSystem(t)
	svc.OnStop = func() (err error) { return nil }
	svc.OnUninstall = func() (err error) { return nil }

	m := &manager{
		logger:        testLogger,
		cmdCons:       executil.EmptyCommandConstructor{},
		isOpenWrt:     true,
		isUnixSystemV: false,
	}

	testCases := []struct {
		cmdCons    executil.CommandConstructor
		name       string
		wantErrMsg string
	}{{
		cmdCons:    newTestCmdConstructor(t, "", nil),
		name:       "success",
		wantErrMsg: "",
	}, {
		cmdCons: newTestCmdConstructor(t, "", assert.AnError),
		name:    "error_disable",
		wantErrMsg: "disabling service on openwrt: starting: " +
			assert.AnError.Error() +
			`; stderr peek: ""; stdout peek: ""`,
	}}

	action := &ActionUninstall{ServiceName: testServiceName}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m.cmdCons = tc.cmdCons

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := m.Perform(ctx, action)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

// newTestServiceWithSystem creates a new *testService with all methods set to
// panic, sets it in the chosen system, and returns it.  The caller can then set
// the desired behavior of the service via overriding its methods.
func newTestServiceWithSystem(tb testing.TB) (svc *testService) {
	tb.Helper()

	svc = newTestService()

	service.ChooseSystem(&testSystem{
		OnNew: func(_ service.Interface, _ *service.Config) (s service.Service, err error) {
			return svc, nil
		},
	})

	return svc
}
