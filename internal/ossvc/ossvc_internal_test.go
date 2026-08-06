package ossvc

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakeos/fakeexec"
	"github.com/kardianos/service"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testServiceName is a common service name for tests.
const testServiceName = "AdGuardHome"

// testLogger is a common logger for tests.
var testLogger = slogutil.NewDiscardLogger()

// testSystem is a mock implementation of [service.System] for tests.
type testSystem struct {
	OnNew func(_ service.Interface, _ *service.Config) (svc service.Service, err error)
}

// type check
var _ service.System = (*testSystem)(nil)

// New implements the [service.System] interface for *testSystem.
func (ts *testSystem) New(i service.Interface, c *service.Config) (svc service.Service, err error) {
	return ts.OnNew(i, c)
}

// String implements the [service.System] interface for *testSystem.
func (ts *testSystem) String() (s string) {
	return "testSystem"
}

// Detect implements the [service.System] interface for *testSystem.
func (ts *testSystem) Detect() (ok bool) {
	return true
}

// Interactive implements the [service.System] interface for *testSystem.
func (ts *testSystem) Interactive() (ok bool) {
	return true
}

// testService is a mock implementation of [service.Service] for tests.
type testService struct {
	OnRun       func() (err error)
	OnStart     func() (err error)
	OnStop      func() (err error)
	OnRestart   func() (err error)
	OnInstall   func() (err error)
	OnUninstall func() (err error)
	OnStatus    func() (s service.Status, err error)
}

// type check
var _ service.Service = (*testService)(nil)

// Run implements the [service.Service] interface for *testService.
func (t *testService) Run() (err error) {
	return t.OnRun()
}

// Start implements the [service.Service] interface for *testService.
func (t *testService) Start() (err error) {
	return t.OnStart()
}

// Stop implements the [service.Service] interface for *testService.
func (t *testService) Stop() (err error) {
	return t.OnStop()
}

// Restart implements the [service.Service] interface for *testService.
func (t *testService) Restart() (err error) {
	return t.OnRestart()
}

// Install implements the [service.Service] interface for *testService.
func (t *testService) Install() (err error) {
	return t.OnInstall()
}

// Uninstall implements the [service.Service] interface for *testService.
func (t *testService) Uninstall() (err error) {
	return t.OnUninstall()
}

// Logger implements the [service.Service] interface for *testService.
func (t *testService) Logger(errs chan<- error) (l service.Logger, err error) {
	return nil, nil
}

// SystemLogger implements the [service.Service] interface for *testService.
func (t *testService) SystemLogger(errs chan<- error) (l service.Logger, err error) {
	return nil, nil
}

// String implements the [service.Service] interface for *testService.
func (t *testService) String() (s string) {
	return testServiceName
}

// Platform implements the [service.Service] interface for *testService.
func (t *testService) Platform() (s string) {
	return "testPlatform"
}

// Status implements the [service.Service] interface for *testService.
func (t *testService) Status() (s service.Status, err error) {
	return t.OnStatus()
}

// newTestService returns a new [*testService] with all methods set to panic,
// since they should be overridden in tests as needed.
func newTestService() (ts *testService) {
	return &testService{
		OnRun:       func() (err error) { panic(testutil.UnexpectedCall()) },
		OnStart:     func() (err error) { panic(testutil.UnexpectedCall()) },
		OnStop:      func() (err error) { panic(testutil.UnexpectedCall()) },
		OnRestart:   func() (err error) { panic(testutil.UnexpectedCall()) },
		OnInstall:   func() (err error) { panic(testutil.UnexpectedCall()) },
		OnUninstall: func() (err error) { panic(testutil.UnexpectedCall()) },
		OnStatus:    func() (s service.Status, err error) { panic(testutil.UnexpectedCall()) },
	}
}

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
