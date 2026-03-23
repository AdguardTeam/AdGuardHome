//go:build darwin

package ossvc

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakeos/fakeexec"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCmdConstructor is a helper that creates a new command constructor. The
// returned constructor creates [fakeexec.Command] instances that print the
// given body to the command's standard output and return the error.
func newTestCmdConstructor(
	tb testing.TB,
	body string,
	returnErr error,
) (c executil.CommandConstructor) {
	tb.Helper()

	onNew := func(
		_ context.Context,
		conf *executil.CommandConfig,
	) (c executil.Command, err error) {
		cmd := fakeexec.NewCommand()
		cmd.OnStart = func(_ context.Context) (err error) {
			_, err = io.WriteString(conf.Stdout, body)
			require.NoError(tb, err)

			return returnErr
		}

		cmd.OnWait = func(_ context.Context) (err error) { return nil }

		return cmd, nil
	}

	return &fakeexec.CommandConstructor{
		OnNew: onNew,
	}
}

func TestDarwinService_Status(t *testing.T) {
	name := "AdGuardHome"
	plistDir := t.TempDir()
	svc := newDarwinService(&darwinServiceConfig{
		logger:   slogutil.NewDiscardLogger(),
		plistDir: plistDir,
		name:     name,
	})

	t.Run("not_installed", func(t *testing.T) {
		status, err := svc.Status()
		assert.Equal(t, service.StatusUnknown, status)
		assert.ErrorIs(t, err, service.ErrNotInstalled)
	})

	plistPath := path.Join(plistDir, name+".plist")
	file, err := os.Create(plistPath)
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, file.Close)

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
			svc.cmdCons = newTestCmdConstructor(t, tc.body, tc.cmdErr)
			var status service.Status
			status, err = svc.Status()
			require.NoError(t, err)

			assert.Equal(t, tc.wantStatus, status)
		})
	}
}
