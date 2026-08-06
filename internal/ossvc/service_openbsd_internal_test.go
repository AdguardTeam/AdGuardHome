//go:build openbsd

package ossvc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenbsdSystem(t *testing.T) {
	t.Parallel()

	sys := &openbsdSystem{
		cmdCons: newTestCmdConstructor(t, "", nil),
	}

	assert.Equal(t, sysVersion, sys.String())
	assert.True(t, sys.Detect())
	assert.Equal(t, os.Getppid() != 1, sys.Interactive())

	svc, err := sys.New(emptyInterface{}, &service.Config{
		Name: testServiceName,
	})
	require.NoError(t, err)
	require.NotNil(t, svc)

	assert.Equal(t, "openbsd", svc.Platform())
	assert.Equal(t, testServiceName, svc.String())

	l, err := svc.Logger(nil)
	require.NoError(t, err)

	assert.NotNil(t, l)

	l, err = svc.SystemLogger(nil)
	require.NoError(t, err)

	assert.NotNil(t, l)
}

func TestOpenbsdRunComService(t *testing.T) {
	t.Parallel()

	scriptsDir := t.TempDir()

	conf := &service.Config{
		Name: testServiceName,
	}
	svc := &openbsdRunComService{
		cmdCons:     executil.EmptyCommandConstructor{},
		i:           emptyInterface{},
		cfg:         conf,
		scriptsPath: scriptsDir,
	}

	err := svc.Install()
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(scriptsDir, testServiceName))

	err = svc.Start()
	require.NoError(t, err)

	err = svc.Stop()
	require.NoError(t, err)

	err = svc.Restart()
	require.NoError(t, err)

	err = svc.Uninstall()
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(scriptsDir, testServiceName))
}

func TestOpenbsdRunComService_Status(t *testing.T) {
	t.Parallel()

	conf := &service.Config{
		Name: testServiceName,
	}

	testCases := []struct {
		cmdErr     error
		name       string
		body       string
		wantErrMsg string
		wantStatus service.Status
	}{{
		name:       "running",
		body:       fmt.Sprintf("%s(ok)\n", testServiceName),
		cmdErr:     nil,
		wantErrMsg: "",
		wantStatus: service.StatusRunning,
	}, {
		name:       "stopped",
		body:       fmt.Sprintf("%s(failed)\n", testServiceName),
		cmdErr:     nil,
		wantErrMsg: "",
		wantStatus: service.StatusStopped,
	}, {
		name:   "unknown_cmd_error",
		body:   "",
		cmdErr: assert.AnError,
		wantErrMsg: "getting status of openbsd-runcom AdGuardHome service: starting: " +
			assert.AnError.Error(),
		wantStatus: service.StatusUnknown,
	}, {
		name:   "unknown",
		body:   "",
		cmdErr: nil,
		wantErrMsg: "getting status of openbsd-runcom AdGuardHome service: " +
			service.ErrNotInstalled.Error(),
		wantStatus: service.StatusUnknown,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &openbsdRunComService{
				cmdCons:     newTestCmdConstructor(t, tc.body, tc.cmdErr),
				i:           emptyInterface{},
				cfg:         conf,
				scriptsPath: t.TempDir(),
			}

			status, err := svc.Status()
			assert.Equal(t, tc.wantStatus, status)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}
