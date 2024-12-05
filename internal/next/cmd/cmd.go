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
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/service"
)

// Main is the entry point of AdGuard Home.
func Main(embeddedFrontend fs.FS) {
	ctx := context.Background()

	start := time.Now()

	cmdName := os.Args[0]
	opts, err := parseOptions(cmdName, os.Args[1:])
	exitCode, needExit := processOptions(opts, cmdName, err)
	if needExit {
		os.Exit(exitCode)
	}

	baseLogger := newBaseLogger(opts)

	baseLogger.InfoContext(
		ctx,
		"starting adguard home",
		"version", version.Version(),
		"pid", os.Getpid(),
	)

	if opts.workDir != "" {
		baseLogger.InfoContext(ctx, "changing working directory", "dir", opts.workDir)

		err = os.Chdir(opts.workDir)
		errors.Check(err)
	}

	frontend, err := frontendFromOpts(ctx, baseLogger, opts, embeddedFrontend)
	errors.Check(err)

	startCtx, startCancel := context.WithTimeout(ctx, defaultTimeoutStart)
	defer startCancel()

	confMgrConf := &configmgr.Config{
		BaseLogger: baseLogger,
		Logger:     baseLogger.With(slogutil.KeyPrefix, "configmgr"),
		Frontend:   frontend,
		WebAddr:    opts.webAddr,
		Start:      start,
		FileName:   opts.confFile,
	}

	confMgr, err := configmgr.New(startCtx, confMgrConf)
	errors.Check(err)

	web := confMgr.Web()
	err = web.Start(startCtx)
	errors.Check(err)

	dns := confMgr.DNS()
	err = dns.Start(startCtx)
	errors.Check(err)

	sigHdlr := newSignalHandler(
		baseLogger.With(slogutil.KeyPrefix, service.SignalHandlerPrefix),
		confMgrConf,
		opts.pidFile,
		web,
		dns,
	)

	os.Exit(sigHdlr.handle(ctx))
}

// Default timeouts.
//
// TODO(a.garipov):  Make configurable.
const (
	defaultTimeoutStart    = 1 * time.Minute
	defaultTimeoutShutdown = 5 * time.Second
)

// newConfigMgr returns a new configuration manager using defaultTimeout as the
// context timeout.
func newConfigMgr(ctx context.Context, c *configmgr.Config) (m *configmgr.Manager, err error) {
	return configmgr.New(ctx, c)
}
