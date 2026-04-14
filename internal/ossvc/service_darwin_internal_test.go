//go:build darwin

package ossvc

import (
	"os"
	"path"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDarwinService_Status(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cmdErr     error
		name       string
		body       string
		wantStatus service.Status
	}{{
		name: "running",
		body: `
		{
			"PID" = 12345;
		};`,
		cmdErr:     nil,
		wantStatus: service.StatusRunning,
	}, {
		name: "restarting",
		body: `
		{
			"foo" = "bar";
		};`,
		cmdErr:     nil,
		wantStatus: statusRestartOnFail,
	}, {
		name:       "stopped",
		body:       "",
		cmdErr:     errors.Error("launchctl error"),
		wantStatus: service.StatusStopped,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plistDir := t.TempDir()
			plistPath := path.Join(plistDir, testServiceName+".plist")
			file, err := os.Create(plistPath)
			require.NoError(t, err)

			testutil.CleanupAndRequireSuccess(t, file.Close)

			svc := newDarwinService(&darwinServiceConfig{
				logger:   testLogger,
				cmdCons:  newTestCmdConstructor(t, tc.body, tc.cmdErr),
				plistDir: plistDir,
				name:     testServiceName,
			})

			var status service.Status
			status, err = svc.Status()
			require.NoError(t, err)

			assert.Equal(t, tc.wantStatus, status)
		})
	}
}

func TestDarwinService_Status_notInstalled(t *testing.T) {
	t.Parallel()

	svc := newDarwinService(&darwinServiceConfig{
		logger:  testLogger,
		cmdCons: executil.EmptyCommandConstructor{},
		name:    testServiceName,
	})

	status, err := svc.Status()
	assert.Equal(t, service.StatusUnknown, status)
	assert.ErrorIs(t, err, service.ErrNotInstalled)
}
