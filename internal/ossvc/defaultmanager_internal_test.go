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

	svc := newTestSvc(t)

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
			return service.StatusRunning, errors.Error("test")
		},
		name:           "error",
		wantStatus:     "",
		wantErrMessage: "getting service status: test",
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
	svc := newTestSvc(t)
	svc.OnStatus = func() (s service.Status, err error) {
		return service.StatusUnknown, errors.Error("test")
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
		cmdCons:    newTestCmdConstructor(t, "", errors.Error("test")),
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

// newTestSvc creates a new *testService with all methods set to panic, sets it
// in the chosen system, and returns it.  The caller can then set the desired
// behavior of the service via overriding its methods.
func newTestSvc(tb testing.TB) (svc *testService) {
	tb.Helper()

	svc = &testService{
		OnRun:       func() (err error) { panic(testutil.UnexpectedCall()) },
		OnStart:     func() (err error) { panic(testutil.UnexpectedCall()) },
		OnStop:      func() (err error) { panic(testutil.UnexpectedCall()) },
		OnRestart:   func() (err error) { panic(testutil.UnexpectedCall()) },
		OnInstall:   func() (err error) { panic(testutil.UnexpectedCall()) },
		OnUninstall: func() (err error) { panic(testutil.UnexpectedCall()) },
		OnStatus:    func() (s service.Status, err error) { panic(testutil.UnexpectedCall()) },
	}

	service.ChooseSystem(&testSystem{
		OnNew: func(_ service.Interface, _ *service.Config) (s service.Service, err error) {
			return svc, nil
		},
	})

	return svc
}
