// Package agh contains common entities and interfaces of AdGuard Home.
package agh

import (
	"context"
	"fmt"
	"strings"

	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil/fakeos/fakeexec"
)

// DefaultOutputLimit is the default limit of bytes for commands' standard
// output and standard error.
const DefaultOutputLimit = 512

// ConfigModifier defines an interface for updating the global configuration.
type ConfigModifier interface {
	// Apply applies changes to the global configuration.
	Apply(ctx context.Context)
}

// EmptyConfigModifier is an empty [ConfigModifier] implementation that does
// nothing.
type EmptyConfigModifier struct{}

// type check
var _ ConfigModifier = EmptyConfigModifier{}

// Apply implements the [ConfigModifier] for EmptyConfigModifier.
func (em EmptyConfigModifier) Apply(ctx context.Context) {}

// exitErr implements [executil.ExitCodeError] for tests to simulate non-zero
// process exit codes.
//
// TODO(s.chzhen):  Consider constructing an [exec.ExitError] instead.
type exitErr struct {
	code osutil.ExitCode
}

// newExitErr returns a properly initialized exitErr with the provided code.
func newExitErr(code osutil.ExitCode) (err exitErr) {
	return exitErr{code: code}
}

// type check
var _ executil.ExitCodeError = exitErr{}

// Error implements the [executil.ExitCodeError] for exitErr.
func (e exitErr) Error() (s string) {
	return fmt.Sprintf("exit code %d", e.code)
}

// ExitCode implements the [executil.ExitCodeError] for exitErr.
func (e exitErr) ExitCode() (code osutil.ExitCode) {
	return e.code
}

// ExternalCommand is a fake command used by [NewMultipleCommandConstructor].
type ExternalCommand struct {
	// Err is the error returned, if non-nil.
	Err error

	// Cmd contains the command path and arguments.
	Cmd string

	// Out is written to stdout if non-empty.
	Out string

	// Code is returned as the exit code if non-zero.
	Code osutil.ExitCode
}

// keyCommand builds a key for a command lookup.
func keyCommand(path string, args []string) (k string) {
	if len(args) == 0 {
		return path
	}

	return path + " " + strings.Join(args, " ")
}

// parseCommand splits a command string into the executable path and args.
func parseCommand(s string) (path string, args []string) {
	f := strings.Fields(s)
	if len(f) == 0 {
		return "", nil
	}

	return f[0], f[1:]
}

// NewMultipleCommandConstructor is a helper function that returns a mock
// [executil.CommandConstructor] for tests that supports multiple commands.
//
// TODO(s.chzhen):  Move to aghtest once the import cycle is resolved, since it
// will be called from the aghnet package, which imports the whois package,
// which in turn imports aghnet.
func NewMultipleCommandConstructor(cmds ...ExternalCommand) (cs executil.CommandConstructor) {
	table := make(map[string]ExternalCommand, len(cmds))
	for _, ec := range cmds {
		p, a := parseCommand(ec.Cmd)
		table[keyCommand(p, a)] = ec
	}

	onNew := func(_ context.Context, conf *executil.CommandConfig) (c executil.Command, err error) {
		ec := table[keyCommand(conf.Path, conf.Args)]

		cmd := fakeexec.NewCommand()
		cmd.OnStart = func(_ context.Context) (err error) {
			if ec.Out != "" {
				_, _ = conf.Stdout.Write([]byte(ec.Out))
			}

			return nil
		}

		cmd.OnWait = func(_ context.Context) (err error) {
			if ec.Err != nil {
				return ec.Err
			}

			if ec.Code != 0 {
				return newExitErr(ec.Code)
			}

			return nil
		}

		return cmd, nil
	}

	return &fakeexec.CommandConstructor{OnNew: onNew}
}

// NewCommandConstructor is a helper function that returns a mock
// [executil.CommandConstructor] for tests.
func NewCommandConstructor(
	_ string,
	code osutil.ExitCode,
	stdout string,
	cmdErr error,
) (cs executil.CommandConstructor) {
	onNew := func(_ context.Context, conf *executil.CommandConfig) (c executil.Command, err error) {
		cmd := fakeexec.NewCommand()
		cmd.OnStart = func(_ context.Context) (err error) {
			if conf.Stdout != nil {
				_, _ = conf.Stdout.Write([]byte(stdout))
			}

			return nil
		}

		cmd.OnWait = func(_ context.Context) (err error) {
			if cmdErr != nil {
				return cmdErr
			}

			if code != 0 {
				return newExitErr(code)
			}

			return nil
		}

		return cmd, nil
	}

	return &fakeexec.CommandConstructor{OnNew: onNew}
}
