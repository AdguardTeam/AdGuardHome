package ossvc_test

import (
	"os"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/ossvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/require"
)

func TestManagerConfig_Validate(t *testing.T) {
	t.Parallel()

	root, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, root.Close)

	valid := &ossvc.ManagerConfig{
		Clock:              timeutil.SystemClock{},
		Logger:             slogutil.NewDiscardLogger(),
		CommandConstructor: executil.EmptyCommandConstructor{},
		RootDir:            root,
	}

	testCases := []struct {
		conf       *ossvc.ManagerConfig
		name       string
		wantErrMsg string
	}{{
		conf:       nil,
		name:       "nil",
		wantErrMsg: "no value",
	}, {
		conf:       valid,
		name:       "valid",
		wantErrMsg: "",
	}, {
		conf: &ossvc.ManagerConfig{
			Clock:              nil,
			Logger:             valid.Logger,
			CommandConstructor: valid.CommandConstructor,
			RootDir:            valid.RootDir,
		},
		name:       "nil_clock",
		wantErrMsg: "Clock: no value",
	}, {
		conf: &ossvc.ManagerConfig{
			Clock:              valid.Clock,
			Logger:             nil,
			CommandConstructor: valid.CommandConstructor,
			RootDir:            valid.RootDir,
		},
		name:       "nil_logger",
		wantErrMsg: "Logger: no value",
	}, {
		conf: &ossvc.ManagerConfig{
			Clock:              valid.Clock,
			Logger:             valid.Logger,
			CommandConstructor: nil,
			RootDir:            valid.RootDir,
		},
		name:       "nil_command_constructor",
		wantErrMsg: "CommandConstructor: no value",
	}, {
		conf: &ossvc.ManagerConfig{
			Clock:              valid.Clock,
			Logger:             valid.Logger,
			CommandConstructor: valid.CommandConstructor,
			RootDir:            nil,
		},
		name:       "nil_root_dir",
		wantErrMsg: "RootDir: no value",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testutil.AssertErrorMsg(t, tc.wantErrMsg, tc.conf.Validate())
		})
	}
}
