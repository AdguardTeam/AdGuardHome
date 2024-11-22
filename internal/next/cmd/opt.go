package cmd

import (
	"context"
	"encoding"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/netip"
	"os"
	"slices"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/configmigrate"
	"github.com/AdguardTeam/AdGuardHome/internal/next/configmgr"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/osutil"
)

// options contains all command-line options for the AdGuardHome(.exe) binary.
type options struct {
	// confFile is the path to the configuration file.
	confFile string

	// logFile is the path to the log file.  Special values:
	//
	//   - "stdout":  Write to stdout (the default).
	//   - "stderr":  Write to stderr.
	//   - "syslog":  Write to the system log.
	logFile string

	// pidFile is the path to the file where to store the PID.
	pidFile string

	// serviceAction is the service control action to perform:
	//
	//   - "install":  Installs AdGuard Home as a system service.
	//   - "uninstall":  Uninstalls it.
	//   - "status":  Prints the service status.
	//   - "start":  Starts the previously installed service.
	//   - "stop":  Stops the previously installed service.
	//   - "restart":  Restarts the previously installed service.
	//   - "reload":  Reloads the configuration.
	//   - "run":  This is a special command that is not supposed to be used
	//     directly it is specified when we register a service, and it indicates
	//     to the app that it is being run as a service.
	//
	// TODO(a.garipov): Use.
	serviceAction string

	// workDir is the path to the working directory.  It is applied before all
	// other configuration is read, so all relative paths are relative to it.
	workDir string

	// webAddr contains the address on which to serve the web UI.
	webAddr netip.AddrPort

	// checkConfig, if true, instructs AdGuard Home to check the configuration
	// file, optionally print an error message to stdout, and exit with a
	// corresponding exit code.
	checkConfig bool

	// disableUpdate, if true, prevents AdGuard Home from automatically checking
	// for updates.
	//
	// TODO(a.garipov): Use.
	disableUpdate bool

	// glinetMode enables the GL-Inet compatibility mode.
	//
	// TODO(a.garipov): Use.
	glinetMode bool

	// help, if true, instructs AdGuard Home to print the command-line option
	// help message and quit with a successful exit-code.
	help bool

	// localFrontend, if true, instructs AdGuard Home to use the local frontend
	// directory instead of the files compiled into the binary.
	//
	// TODO(a.garipov): Use.
	localFrontend bool

	// performUpdate, if true, instructs AdGuard Home to update the current
	// binary and restart the service in case it's installed.
	//
	// TODO(a.garipov): Use.
	performUpdate bool

	// noPermCheck, if true, instructs AdGuard Home to skip checking and
	// migrating the permissions of its security-sensitive files.
	//
	// TODO(e.burkov):  Use.
	noPermCheck bool

	// verbose, if true, instructs AdGuard Home to enable verbose logging.
	verbose bool

	// version, if true, instructs AdGuard Home to print the version to stdout
	// and quit with a successful exit-code.  If verbose is also true, print a
	// more detailed version description.
	version bool
}

// Indexes to help with the [commandLineOptions] initialization.
const (
	confFileIdx = iota
	logFileIdx
	pidFileIdx
	serviceActionIdx
	workDirIdx
	webAddrIdx
	checkConfigIdx
	disableUpdateIdx
	glinetModeIdx
	helpIdx
	localFrontendIdx
	noPermCheckIdx
	performUpdateIdx
	verboseIdx
	versionIdx
)

// commandLineOption contains information about a command-line option: its long
// and, if there is one, short forms, the value type, the description, and the
// default value.
type commandLineOption struct {
	defaultValue any
	description  string
	long         string
	short        string
	valueType    string
}

// commandLineOptions are all command-line options currently supported by
// AdGuard Home.
var commandLineOptions = []*commandLineOption{
	confFileIdx: {
		// TODO(a.garipov): Remove the directory when the new code is ready.
		defaultValue: "internal/next/AdGuardHome.yaml",
		description:  "Path to the config file.",
		long:         "config",
		short:        "c",
		valueType:    "path",
	},

	logFileIdx: {
		defaultValue: "stdout",
		description:  `Path to log file.  Special values include "stdout", "stderr", and "syslog".`,
		long:         "logfile",
		short:        "l",
		valueType:    "path",
	},

	pidFileIdx: {
		defaultValue: "",
		description:  "Path to the file where to store the PID.",
		long:         "pidfile",
		short:        "",
		valueType:    "path",
	},

	serviceActionIdx: {
		defaultValue: "",
		description: `Service control action: "status", "install" (as a service), ` +
			`"uninstall" (as a service), "start", "stop", "restart", "reload" (configuration).`,
		long:      "service",
		short:     "s",
		valueType: "action",
	},

	workDirIdx: {
		defaultValue: "",
		description: `Path to the working directory.  ` +
			`It is applied before all other configuration is read, ` +
			`so all relative paths are relative to it.`,
		long:      "work-dir",
		short:     "w",
		valueType: "path",
	},

	webAddrIdx: {
		defaultValue: netip.AddrPort{},
		description:  `Address to serve the web UI on, in the host:port format.`,
		long:         "web-addr",
		short:        "",
		valueType:    "host:port",
	},

	checkConfigIdx: {
		defaultValue: false,
		description:  "Check configuration, print errors to stdout, and quit.",
		long:         "check-config",
		short:        "",
		valueType:    "",
	},

	disableUpdateIdx: {
		defaultValue: false,
		description:  "Disable automatic update checking.",
		long:         "no-check-update",
		short:        "",
		valueType:    "",
	},

	glinetModeIdx: {
		defaultValue: false,
		description:  "Run in GL-Inet compatibility mode.",
		long:         "glinet",
		short:        "",
		valueType:    "",
	},

	helpIdx: {
		defaultValue: false,
		description:  "Print this help message and quit.",
		long:         "help",
		short:        "h",
		valueType:    "",
	},

	localFrontendIdx: {
		defaultValue: false,
		description:  "Use local frontend directories.",
		long:         "local-frontend",
		short:        "",
		valueType:    "",
	},

	noPermCheckIdx: {
		defaultValue: false,
		description:  "Skip checking the permissions of security-sensitive files.",
		long:         "no-permcheck",
		short:        "",
		valueType:    "",
	},

	performUpdateIdx: {
		defaultValue: false,
		description:  "Update the current binary and restart the service in case it's installed.",
		long:         "update",
		short:        "",
		valueType:    "",
	},

	verboseIdx: {
		defaultValue: false,
		description:  "Enable verbose logging.",
		long:         "verbose",
		short:        "v",
		valueType:    "",
	},

	versionIdx: {
		defaultValue: false,
		description: `Print the version to stdout and quit.  ` +
			`Print a more detailed version description with -v.`,
		long:      "version",
		short:     "",
		valueType: "",
	},
}

// parseOptions parses the command-line options for AdGuardHome.
func parseOptions(cmdName string, args []string) (opts *options, err error) {
	flags := flag.NewFlagSet(cmdName, flag.ContinueOnError)

	opts = &options{}
	for i, fieldPtr := range []any{
		confFileIdx:      &opts.confFile,
		logFileIdx:       &opts.logFile,
		pidFileIdx:       &opts.pidFile,
		serviceActionIdx: &opts.serviceAction,
		workDirIdx:       &opts.workDir,
		webAddrIdx:       &opts.webAddr,
		checkConfigIdx:   &opts.checkConfig,
		disableUpdateIdx: &opts.disableUpdate,
		glinetModeIdx:    &opts.glinetMode,
		helpIdx:          &opts.help,
		localFrontendIdx: &opts.localFrontend,
		noPermCheckIdx:   &opts.noPermCheck,
		performUpdateIdx: &opts.performUpdate,
		verboseIdx:       &opts.verbose,
		versionIdx:       &opts.version,
	} {
		addOption(flags, fieldPtr, commandLineOptions[i])
	}

	flags.Usage = func() { usage(cmdName, os.Stderr) }

	err = flags.Parse(args)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	return opts, nil
}

// addOption adds the command-line option described by o to flags using fieldPtr
// as the pointer to the value.
func addOption(flags *flag.FlagSet, fieldPtr any, o *commandLineOption) {
	switch fieldPtr := fieldPtr.(type) {
	case *string:
		flags.StringVar(fieldPtr, o.long, o.defaultValue.(string), o.description)
		if o.short != "" {
			flags.StringVar(fieldPtr, o.short, o.defaultValue.(string), o.description)
		}
	case *bool:
		flags.BoolVar(fieldPtr, o.long, o.defaultValue.(bool), o.description)
		if o.short != "" {
			flags.BoolVar(fieldPtr, o.short, o.defaultValue.(bool), o.description)
		}
	case encoding.TextUnmarshaler:
		flags.TextVar(fieldPtr, o.long, o.defaultValue.(encoding.TextMarshaler), o.description)
		if o.short != "" {
			flags.TextVar(fieldPtr, o.short, o.defaultValue.(encoding.TextMarshaler), o.description)
		}
	default:
		panic(fmt.Errorf("unexpected field pointer type %T", fieldPtr))
	}
}

// usage prints a usage message similar to the one printed by package flag but
// taking long vs. short versions into account as well as using more informative
// value hints.
func usage(cmdName string, output io.Writer) {
	options := slices.Clone(commandLineOptions)
	slices.SortStableFunc(options, func(a, b *commandLineOption) (res int) {
		return strings.Compare(a.long, b.long)
	})

	b := &strings.Builder{}
	_, _ = fmt.Fprintf(b, "Usage of %s:\n", cmdName)

	for _, o := range options {
		writeUsageLine(b, o)

		// Use four spaces before the tab to trigger good alignment for both 4-
		// and 8-space tab stops.
		if shouldIncludeDefault(o.defaultValue) {
			_, _ = fmt.Fprintf(b, "    \t%s  (Default value: %q)\n", o.description, o.defaultValue)
		} else {
			_, _ = fmt.Fprintf(b, "    \t%s\n", o.description)
		}
	}

	_, _ = io.WriteString(output, b.String())
}

// shouldIncludeDefault returns true if this default value should be printed.
func shouldIncludeDefault(v any) (ok bool) {
	switch v := v.(type) {
	case bool:
		return v
	case string:
		return v != ""
	default:
		return v == nil
	}
}

// writeUsageLine writes the usage line for the provided command-line option.
func writeUsageLine(b *strings.Builder, o *commandLineOption) {
	if o.short == "" {
		if o.valueType == "" {
			_, _ = fmt.Fprintf(b, "  --%s\n", o.long)
		} else {
			_, _ = fmt.Fprintf(b, "  --%s=%s\n", o.long, o.valueType)
		}

		return
	}

	if o.valueType == "" {
		_, _ = fmt.Fprintf(b, "  --%s/-%s\n", o.long, o.short)
	} else {
		_, _ = fmt.Fprintf(b, "  --%[1]s=%[3]s/-%[2]s %[3]s\n", o.long, o.short, o.valueType)
	}
}

// processOptions decides if AdGuard Home should exit depending on the results
// of command-line option parsing.
func processOptions(
	opts *options,
	cmdName string,
	parseErr error,
) (exitCode int, needExit bool) {
	if parseErr != nil {
		// Assume that usage has already been printed.
		return osutil.ExitCodeArgumentError, true
	}

	if opts.help {
		usage(cmdName, os.Stdout)

		return osutil.ExitCodeSuccess, true
	}

	if opts.version {
		if opts.verbose {
			fmt.Print(version.Verbose(configmigrate.LastSchemaVersion))
		} else {
			fmt.Printf("AdGuard Home %s\n", version.Version())
		}

		return osutil.ExitCodeSuccess, true
	}

	if opts.checkConfig {
		err := configmgr.Validate(opts.confFile)
		if err != nil {
			_, _ = io.WriteString(os.Stdout, err.Error()+"\n")

			return osutil.ExitCodeFailure, true
		}

		return osutil.ExitCodeSuccess, true
	}

	return 0, false
}

// frontendFromOpts returns the frontend to use based on the options.
func frontendFromOpts(
	ctx context.Context,
	logger *slog.Logger,
	opts *options,
	embeddedFrontend fs.FS,
) (frontend fs.FS, err error) {
	const frontendSubdir = "build/static"

	if opts.localFrontend {
		logger.WarnContext(ctx, "using local frontend files")

		return os.DirFS(frontendSubdir), nil
	}

	return fs.Sub(embeddedFrontend, frontendSubdir)
}
