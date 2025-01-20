package home

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/configmigrate"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/stringutil"
)

// TODO(a.garipov): Replace with package flag.

// options represents the command-line options.
type options struct {
	// confFilename is the path to the configuration file.
	confFilename string

	// workDir is the path to the working directory where AdGuard Home stores
	// filter data, the query log, and other data.
	workDir string

	// logFile is the path to the log file.  If empty, AdGuard Home writes to
	// stdout; if "syslog", to syslog.
	logFile string

	// pidFile is the file name for the file to which the PID is saved.
	pidFile string

	// serviceControlAction is the service action to perform.  See
	// [service.ControlAction] and [handleServiceControlAction].
	serviceControlAction string

	// bindHost is the address on which to serve the HTTP UI.
	//
	// Deprecated: Use bindAddr.
	bindHost netip.Addr

	// bindPort is the port on which to serve the HTTP UI.
	//
	// Deprecated: Use bindAddr.
	bindPort uint16

	// bindAddr is the address to serve the web UI on.
	bindAddr netip.AddrPort

	// checkConfig is true if the current invocation is only required to check
	// the configuration file and exit.
	checkConfig bool

	// disableUpdate, if set, makes AdGuard Home not check for updates.
	disableUpdate bool

	// performUpdate, if set, updates AdGuard Home without GUI and exits.
	performUpdate bool

	// verbose shows if verbose logging is enabled.
	verbose bool

	// runningAsService flag is set to true when options are passed from the
	// service runner
	//
	// TODO(a.garipov): Perhaps this could be determined by a non-empty
	// serviceControlAction?
	runningAsService bool

	// glinetMode shows if the GL-Inet compatibility mode is enabled.
	glinetMode bool

	// noEtcHosts flag should be provided when /etc/hosts file shouldn't be
	// used.
	noEtcHosts bool

	// localFrontend forces AdGuard Home to use the frontend files from disk
	// rather than the ones that have been compiled into the binary.
	localFrontend bool

	// noPermCheck disables checking and migration of permissions for the
	// security-sensitive files.
	noPermCheck bool
}

// initCmdLineOpts completes initialization of the global command-line option
// slice.  It must only be called once.
func initCmdLineOpts() {
	// The --help option cannot be put directly into cmdLineOpts, because that
	// causes initialization cycle due to printHelp referencing cmdLineOpts.
	cmdLineOpts = append(cmdLineOpts, cmdLineOpt{
		updateWithValue: nil,
		updateNoValue:   nil,
		effect: func(o options, exec string) (effect, error) {
			return func() error { printHelp(exec); exitWithError(); return nil }, nil
		},
		serialize:   func(o options) (val string, ok bool) { return "", false },
		description: "Print this help.",
		longName:    "help",
		shortName:   "",
	})
}

// effect is the type for functions used for their side-effects.
type effect func() (err error)

// cmdLineOpt contains the data for a single command-line option.  Only one of
// updateWithValue, updateNoValue, and effect must be present.
type cmdLineOpt struct {
	updateWithValue func(o options, v string) (updated options, err error)
	updateNoValue   func(o options) (updated options, err error)
	effect          func(o options, exec string) (eff effect, err error)

	// serialize is a function that encodes the option into a slice of
	// command-line arguments, if necessary.  If ok is false, this option should
	// be skipped.
	serialize func(o options) (val string, ok bool)

	description string
	longName    string
	shortName   string
}

// cmdLineOpts are all command-line options of AdGuard Home.
var cmdLineOpts = []cmdLineOpt{{
	updateWithValue: func(o options, v string) (options, error) {
		o.confFilename = v
		return o, nil
	},
	updateNoValue: nil,
	effect:        nil,
	serialize: func(o options) (val string, ok bool) {
		return o.confFilename, o.confFilename != ""
	},
	description: "Path to the config file.",
	longName:    "config",
	shortName:   "c",
}, {
	updateWithValue: func(o options, v string) (options, error) { o.workDir = v; return o, nil },
	updateNoValue:   nil,
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return o.workDir, o.workDir != "" },
	description:     "Path to the working directory.",
	longName:        "work-dir",
	shortName:       "w",
}, {
	updateWithValue: func(o options, v string) (oo options, err error) {
		o.bindHost, err = netip.ParseAddr(v)

		return o, err
	},
	updateNoValue: nil,
	effect:        nil,
	serialize: func(o options) (val string, ok bool) {
		if !o.bindHost.IsValid() {
			return "", false
		}

		return o.bindHost.String(), true
	},
	description: "Deprecated. Host address to bind HTTP server on. Use --web-addr. " +
		"The short -h will work as --help in the future.",
	longName:  "host",
	shortName: "h",
}, {
	updateWithValue: func(o options, v string) (options, error) {
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			err = fmt.Errorf("parsing port: %w", err)
		} else {
			o.bindPort = uint16(p)
		}

		return o, err
	},
	updateNoValue: nil,
	effect:        nil,
	serialize: func(o options) (val string, ok bool) {
		if o.bindPort == 0 {
			return "", false
		}

		return strconv.Itoa(int(o.bindPort)), true
	},
	description: "Deprecated. Port to serve HTTP pages on. Use --web-addr.",
	longName:    "port",
	shortName:   "p",
}, {
	updateWithValue: func(o options, v string) (oo options, err error) {
		o.bindAddr, err = netip.ParseAddrPort(v)

		return o, err
	},
	updateNoValue: nil,
	effect:        nil,
	serialize: func(o options) (val string, ok bool) {
		return o.bindAddr.String(), o.bindAddr.IsValid()
	},
	description: "Address to serve the web UI on, in the host:port format.",
	longName:    "web-addr",
	shortName:   "",
}, {
	updateWithValue: func(o options, v string) (options, error) {
		o.serviceControlAction = v
		return o, nil
	},
	updateNoValue: nil,
	effect:        nil,
	serialize: func(o options) (val string, ok bool) {
		return o.serviceControlAction, o.serviceControlAction != ""
	},
	description: `Service control action: status, install (as a service), ` +
		`uninstall (as a service), start, stop, restart, reload (configuration).`,
	longName:  "service",
	shortName: "s",
}, {
	updateWithValue: func(o options, v string) (options, error) { o.logFile = v; return o, nil },
	updateNoValue:   nil,
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return o.logFile, o.logFile != "" },
	description: `Path to log file.  If empty, write to stdout; ` +
		`if "syslog", write to system log.`,
	longName:  "logfile",
	shortName: "l",
}, {
	updateWithValue: func(o options, v string) (options, error) { o.pidFile = v; return o, nil },
	updateNoValue:   nil,
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return o.pidFile, o.pidFile != "" },
	description:     "Path to a file where PID is stored.",
	longName:        "pidfile",
	shortName:       "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.checkConfig = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.checkConfig },
	description:     "Check configuration and exit.",
	longName:        "check-config",
	shortName:       "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.disableUpdate = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.disableUpdate },
	description:     "Don't check for updates.",
	longName:        "no-check-update",
	shortName:       "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.performUpdate = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.performUpdate },
	description:     "Update the current binary and restart the service in case it's installed.",
	longName:        "update",
	shortName:       "",
}, {
	updateWithValue: nil,
	updateNoValue:   nil,
	effect: func(_ options, _ string) (f effect, err error) {
		log.Info("warning: using --no-mem-optimization flag has no effect and is deprecated")

		return nil, nil
	},
	serialize:   func(o options) (val string, ok bool) { return "", false },
	description: "Deprecated.  Disable memory optimization.",
	longName:    "no-mem-optimization",
	shortName:   "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.noEtcHosts = true; return o, nil },
	effect: func(_ options, _ string) (f effect, err error) {
		log.Info(
			"warning: --no-etc-hosts flag is deprecated " +
				"and will be removed in the future versions; " +
				"set clients.runtime_sources.hosts and dns.hostsfile_enabled " +
				"in the configuration file to false instead",
		)

		return nil, nil
	},
	serialize: func(o options) (val string, ok bool) { return "", o.noEtcHosts },
	description: "Deprecated: use clients.runtime_sources.hosts and dns.hostsfile_enabled " +
		"instead.  Do not use the OS-provided hosts.",
	longName:  "no-etc-hosts",
	shortName: "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.localFrontend = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.localFrontend },
	description:     "Use local frontend directories.",
	longName:        "local-frontend",
	shortName:       "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.verbose = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.verbose },
	description:     "Enable verbose output.",
	longName:        "verbose",
	shortName:       "v",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.glinetMode = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.glinetMode },
	description:     "Run in GL-Inet compatibility mode.",
	longName:        "glinet",
	shortName:       "",
}, {
	updateWithValue: nil,
	updateNoValue:   func(o options) (options, error) { o.noPermCheck = true; return o, nil },
	effect:          nil,
	serialize:       func(o options) (val string, ok bool) { return "", o.noPermCheck },
	description: "Skip checking and migration of permissions " +
		"of security-sensitive files.",
	longName:  "no-permcheck",
	shortName: "",
}, {
	updateWithValue: nil,
	updateNoValue:   nil,
	effect: func(o options, exec string) (effect, error) {
		return func() error {
			if o.verbose {
				fmt.Print(version.Verbose(configmigrate.LastSchemaVersion))
			} else {
				fmt.Println(version.Full())
			}

			os.Exit(osutil.ExitCodeSuccess)

			return nil
		}, nil
	},
	serialize:   func(o options) (val string, ok bool) { return "", false },
	description: "Show the version and exit.  Show more detailed version description with -v.",
	longName:    "version",
	shortName:   "",
}}

// printHelp prints the entire help message.  It exits with an error code if
// there are any I/O errors.
func printHelp(exec string) {
	b := &strings.Builder{}

	stringutil.WriteToBuilder(
		b,
		"Usage:\n\n",
		fmt.Sprintf("%s [options]\n\n", exec),
		"Options:\n",
	)

	var err error
	for _, opt := range cmdLineOpts {
		val := ""
		if opt.updateWithValue != nil {
			val = " VALUE"
		}

		longDesc := opt.longName + val
		if opt.shortName != "" {
			_, err = fmt.Fprintf(b, "  -%s, --%-28s %s\n", opt.shortName, longDesc, opt.description)
		} else {
			_, err = fmt.Fprintf(b, "  --%-32s %s\n", longDesc, opt.description)
		}

		if err != nil {
			// The only error here can be from incorrect Fprintf usage, which is
			// a programmer error.
			panic(err)
		}
	}

	_, err = fmt.Print(b)
	if err != nil {
		// Exit immediately, since not being able to print out a help message
		// essentially means that the I/O is very broken at the moment.
		exitWithError()
	}
}

// parseCmdOpts parses the command-line arguments into options and effects.
func parseCmdOpts(cmdName string, args []string) (o options, eff effect, err error) {
	// Don't use range since the loop changes the loop variable.
	argsLen := len(args)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		isKnown := false
		for _, opt := range cmdLineOpts {
			isKnown = argMatches(opt, arg)
			if !isKnown {
				continue
			}

			if opt.updateWithValue != nil {
				i++
				if i >= argsLen {
					return o, eff, fmt.Errorf("got %s without argument", arg)
				}

				o, err = opt.updateWithValue(o, args[i])
			} else {
				o, eff, err = updateOptsNoValue(o, eff, opt, cmdName)
			}

			if err != nil {
				return o, eff, fmt.Errorf("applying option %s: %w", arg, err)
			}

			break
		}

		if !isKnown {
			return o, eff, fmt.Errorf("unknown option %s", arg)
		}
	}

	return o, eff, err
}

// argMatches returns true if arg matches command-line option opt.
func argMatches(opt cmdLineOpt, arg string) (ok bool) {
	if arg == "" || arg[0] != '-' {
		return false
	}

	arg = arg[1:]
	if arg == "" {
		return false
	}

	return (opt.shortName != "" && arg == opt.shortName) ||
		(arg[0] == '-' && arg[1:] == opt.longName)
}

// updateOptsNoValue sets values or effects from opt into o or prev.
func updateOptsNoValue(
	o options,
	prev effect,
	opt cmdLineOpt,
	cmdName string,
) (updated options, chained effect, err error) {
	if opt.updateNoValue != nil {
		o, err = opt.updateNoValue(o)
		if err != nil {
			return o, prev, err
		}

		return o, prev, nil
	}

	next, err := opt.effect(o, cmdName)
	if err != nil {
		return o, prev, err
	}

	chained = chainEffect(prev, next)

	return o, chained, nil
}

// chainEffect chans the next effect after the prev one.  If prev is nil, eff
// only calls next.  If next is nil, eff is prev; if prev is nil, eff is next.
func chainEffect(prev, next effect) (eff effect) {
	if prev == nil {
		return next
	} else if next == nil {
		return prev
	}

	eff = func() (err error) {
		err = prev()
		if err != nil {
			return err
		}

		return next()
	}

	return eff
}

// optsToArgs converts command line options into a list of arguments.
func optsToArgs(o options) (args []string) {
	for _, opt := range cmdLineOpts {
		val, ok := opt.serialize(o)
		if !ok {
			continue
		}

		if opt.shortName != "" {
			args = append(args, "-"+opt.shortName)
		} else {
			args = append(args, "--"+opt.longName)
		}

		if val != "" {
			args = append(args, val)
		}
	}

	return args
}
