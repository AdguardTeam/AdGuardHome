// Package cmd is the AdGuard Home entry point.  It assembles the configuration
// file manager, sets up signal processing logic, and so on.
//
// TODO(a.garipov): Move to the upper-level internal/.
package cmd

import (
	"context"
	"io/fs"
	"os"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/configmgr"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/log"
)

// Main is the entry point of AdGuard Home.
func Main(frontend fs.FS) {
	start := time.Now()

	cmdName := os.Args[0]
	opts, err := parseOptions(cmdName, os.Args[1:])
	exitCode, needExit := processOptions(opts, cmdName, err)
	if needExit {
		os.Exit(exitCode)
	}

	err = setLog(opts)
	check(err)

	log.Info("starting adguard home, version %s, pid %d", version.Version(), os.Getpid())

	if opts.workDir != "" {
		log.Info("changing working directory to %q", opts.workDir)
		err = os.Chdir(opts.workDir)
		check(err)
	}

	confMgr, err := newConfigMgr(opts.confFile, frontend, start)
	check(err)

	web := confMgr.Web()
	err = web.Start()
	check(err)

	dns := confMgr.DNS()
	err = dns.Start()
	check(err)

	sigHdlr := newSignalHandler(
		opts.confFile,
		frontend,
		start,
		web,
		dns,
	)

	sigHdlr.handle()
}

// defaultTimeout is the timeout used for some operations where another timeout
// hasn't been defined yet.
const defaultTimeout = 5 * time.Second

// ctxWithDefaultTimeout is a helper function that returns a context with
// timeout set to defaultTimeout.
func ctxWithDefaultTimeout() (ctx context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// newConfigMgr returns a new configuration manager using defaultTimeout as the
// context timeout.
func newConfigMgr(confFile string, frontend fs.FS, start time.Time) (m *configmgr.Manager, err error) {
	ctx, cancel := ctxWithDefaultTimeout()
	defer cancel()

	return configmgr.New(ctx, confFile, frontend, start)
}

// check is a simple error-checking helper.  It must only be used within Main.
func check(err error) {
	if err != nil {
		panic(err)
	}
}
