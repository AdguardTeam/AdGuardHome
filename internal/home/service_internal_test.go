package home

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/ossvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServiceManager records service actions performed by install.
type fakeServiceManager struct {
	installErr   error
	startErr     error
	uninstallErr error
	actions      []ossvc.ActionName
}

// type check
var _ ossvc.Manager = (*fakeServiceManager)(nil)

// Perform implements [ossvc.Manager] for *fakeServiceManager.
func (f *fakeServiceManager) Perform(_ context.Context, action ossvc.Action) (err error) {
	f.actions = append(f.actions, action.Name())

	switch action.(type) {
	case *ossvc.ActionInstall:
		return f.installErr
	case *ossvc.ActionStart:
		return f.startErr
	case *ossvc.ActionUninstall:
		return f.uninstallErr
	default:
		return nil
	}
}

// Status implements [ossvc.Manager] for *fakeServiceManager.
func (f *fakeServiceManager) Status(
	_ context.Context,
	_ ossvc.ServiceName,
) (status ossvc.Status, err error) {
	return ossvc.StatusNotInstalled, nil
}

func TestHandleServiceInstallCmd_rollback(t *testing.T) {
	workDir := t.TempDir()
	confPath := filepath.Join(workDir, "AdGuardHome.yaml")
	require.NoError(t, os.WriteFile(confPath, nil, 0o600))

	testCases := []struct {
		installErr   error
		startErr     error
		uninstallErr error
		name         string
		wantErrMsg   string
		wantActions  []ossvc.ActionName
	}{{
		name:        "install_error",
		installErr:  assert.AnError,
		wantErrMsg:  "installing service: " + assert.AnError.Error(),
		wantActions: []ossvc.ActionName{ossvc.ActionNameInstall},
	}, {
		name:       "start_error",
		startErr:   assert.AnError,
		wantErrMsg: "starting service: " + assert.AnError.Error(),
		wantActions: []ossvc.ActionName{
			ossvc.ActionNameInstall,
			ossvc.ActionNameStart,
			ossvc.ActionNameUninstall,
		},
	}, {
		name:         "start_and_uninstall_error",
		startErr:     assert.AnError,
		uninstallErr: assert.AnError,
		wantErrMsg: "starting service: " + assert.AnError.Error() +
			" (uninstalling: " + assert.AnError.Error() + ")",
		wantActions: []ossvc.ActionName{
			ossvc.ActionNameInstall,
			ossvc.ActionNameStart,
			ossvc.ActionNameUninstall,
		},
	}, {
		name:        "success",
		wantActions: []ossvc.ActionName{ossvc.ActionNameInstall, ossvc.ActionNameStart},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := &fakeServiceManager{
				installErr:   tc.installErr,
				startErr:     tc.startErr,
				uninstallErr: tc.uninstallErr,
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			err := handleServiceInstallCmd(
				ctx,
				slogutil.NewDiscardLogger(),
				mgr,
				options{},
				workDir,
				confPath,
			)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
			assert.Equal(t, tc.wantActions, mgr.actions)
		})
	}
}
