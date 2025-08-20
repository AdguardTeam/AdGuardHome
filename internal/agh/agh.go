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

// TODO(s.chzhen): !! Is there another way?
//
// TODO(s.chzhen): !! Docs, naming.
//
// TODO(s.chzhen):  Move to aghtest once the import cycle is resolved.
type exitErr struct {
	code osutil.ExitCode
}

// type check
var _ executil.ExitCodeError = exitErr{}

func (e exitErr) Error() (s string) {
	return fmt.Sprintf("exit code %d", e.code)
}

func (e exitErr) ExitCode() (code osutil.ExitCode) {
	return e.code
}

type ExternalCommand struct {
	Err  error
	Cmd  string
	Out  string
	Code int
}

func keyCommand(path string, args []string) (k string) {
	return path + " " + strings.Join(args, " ")
}

func parseCommand(s string) (path string, args []string) {
	f := strings.Fields(s)
	if len(f) == 0 {
		return "", nil
	}

	return f[0], f[1:]
}

// NewMultipleCommandConstructor is a helper function that returns a mock
// [executil.CommandConstructor] for tests.
func NewMultipleCommandConstructor(cmds ...ExternalCommand) (cs executil.CommandConstructor) {
	table := make(map[string]ExternalCommand, len(cmds))
	for _, ec := range cmds {
		p, a := parseCommand(ec.Cmd)
		table[keyCommand(p, a)] = ec
	}

	return &fakeexec.CommandConstructor{
		OnNew: func(
			_ context.Context,
			conf *executil.CommandConfig,
		) (c executil.Command, err error) {
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
					return exitErr{code: ec.Code}
				}

				return nil
			}

			return cmd, nil
		},
	}
}

// NewCommandConstructor is a helper function that returns a mock
// [executil.CommandConstructor] for tests.
func NewCommandConstructor(
	_ string,
	code int,
	stdout string,
	cmdErr error,
) (cs executil.CommandConstructor) {
	return &fakeexec.CommandConstructor{
		OnNew: func(
			_ context.Context,
			conf *executil.CommandConfig,
		) (c executil.Command, err error) {
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
					return exitErr{code: code}
				}

				return nil
			}

			return cmd, nil
		},
	}
}
