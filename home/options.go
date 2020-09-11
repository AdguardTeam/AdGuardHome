package home

import (
	"fmt"
	"os"
	"strconv"
)

// options passed from command-line arguments
type options struct {
	verbose        bool   // is verbose logging enabled
	configFilename string // path to the config file
	workDir        string // path to the working directory where we will store the filters data and the querylog
	bindHost       string // host address to bind HTTP server on
	bindPort       int    // port to serve HTTP pages on
	logFile        string // Path to the log file. If empty, write to stdout. If "syslog", writes to syslog
	pidFile        string // File name to save PID to
	checkConfig    bool   // Check configuration and exit
	disableUpdate  bool   // If set, don't check for updates

	// service control action (see service.ControlAction array + "status" command)
	serviceControlAction string

	// runningAsService flag is set to true when options are passed from the service runner
	runningAsService bool

	// disableMemoryOptimization - disables memory optimization hacks
	// see memoryUsage() function for the details
	disableMemoryOptimization bool

	glinetMode bool // Activate GL-Inet compatibility mode
}

// functions used for their side-effects
type effect func() error

type arg struct {
	description string // a short, English description of the argument
	longName    string // the name of the argument used after '--'
	shortName   string // the name of the argument used after '-'

	// only one of updateWithValue, updateNoValue, and effect should be present

	updateWithValue func(o options, v string) (options, error)         // the mutator for arguments with parameters
	updateNoValue   func(o options) (options, error)                   // the mutator for arguments without parameters
	effect          func(o options, exec string) (f effect, err error) // the side-effect closure generator

	serialize func(o options) []string // the re-serialization function back to arguments (return nil for omit)
}

// {type}SliceOrNil functions check their parameter of type {type}
// against its zero value and return nil if the parameter value is
// zero otherwise they return a string slice of the parameter

func stringSliceOrNil(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func intSliceOrNil(i int) []string {
	if i == 0 {
		return nil
	}
	return []string{strconv.Itoa(i)}
}

func boolSliceOrNil(b bool) []string {
	if b {
		return []string{}
	}
	return nil
}

var args []arg

var configArg = arg{
	"Path to the config file",
	"config", "c",
	func(o options, v string) (options, error) { o.configFilename = v; return o, nil },
	nil,
	nil,
	func(o options) []string { return stringSliceOrNil(o.configFilename) },
}

var workDirArg = arg{
	"Path to the working directory",
	"work-dir", "w",
	func(o options, v string) (options, error) { o.workDir = v; return o, nil }, nil, nil,
	func(o options) []string { return stringSliceOrNil(o.workDir) },
}

var hostArg = arg{
	"Host address to bind HTTP server on",
	"host", "h",
	func(o options, v string) (options, error) { o.bindHost = v; return o, nil }, nil, nil,
	func(o options) []string { return stringSliceOrNil(o.bindHost) },
}

var portArg = arg{
	"Port to serve HTTP pages on",
	"port", "p",
	func(o options, v string) (options, error) {
		var err error
		var p int
		minPort, maxPort := 0, 1<<16-1
		if p, err = strconv.Atoi(v); err != nil {
			err = fmt.Errorf("port '%s' is not a number", v)
		} else if p < minPort || p > maxPort {
			err = fmt.Errorf("port %d not in range %d - %d", p, minPort, maxPort)
		} else {
			o.bindPort = p
		}
		return o, err
	}, nil, nil,
	func(o options) []string { return intSliceOrNil(o.bindPort) },
}

var serviceArg = arg{
	"Service control action: status, install, uninstall, start, stop, restart, reload (configuration)",
	"service", "s",
	func(o options, v string) (options, error) {
		o.serviceControlAction = v
		return o, nil
	}, nil, nil,
	func(o options) []string { return stringSliceOrNil(o.serviceControlAction) },
}

var logfileArg = arg{
	"Path to log file. If empty: write to stdout; if 'syslog': write to system log",
	"logfile", "l",
	func(o options, v string) (options, error) { o.logFile = v; return o, nil }, nil, nil,
	func(o options) []string { return stringSliceOrNil(o.logFile) },
}

var pidfileArg = arg{
	"Path to a file where PID is stored",
	"pidfile", "",
	func(o options, v string) (options, error) { o.pidFile = v; return o, nil }, nil, nil,
	func(o options) []string { return stringSliceOrNil(o.pidFile) },
}

var checkConfigArg = arg{
	"Check configuration and exit",
	"check-config", "",
	nil, func(o options) (options, error) { o.checkConfig = true; return o, nil }, nil,
	func(o options) []string { return boolSliceOrNil(o.checkConfig) },
}

var noCheckUpdateArg = arg{
	"Don't check for updates",
	"no-check-update", "",
	nil, func(o options) (options, error) { o.disableUpdate = true; return o, nil }, nil,
	func(o options) []string { return boolSliceOrNil(o.disableUpdate) },
}

var disableMemoryOptimizationArg = arg{
	"Disable memory optimization",
	"no-mem-optimization", "",
	nil, func(o options) (options, error) { o.disableMemoryOptimization = true; return o, nil }, nil,
	func(o options) []string { return boolSliceOrNil(o.disableMemoryOptimization) },
}

var verboseArg = arg{
	"Enable verbose output",
	"verbose", "v",
	nil, func(o options) (options, error) { o.verbose = true; return o, nil }, nil,
	func(o options) []string { return boolSliceOrNil(o.verbose) },
}

var glinetArg = arg{
	"Run in GL-Inet compatibility mode",
	"glinet", "",
	nil, func(o options) (options, error) { o.glinetMode = true; return o, nil }, nil,
	func(o options) []string { return boolSliceOrNil(o.glinetMode) },
}

var versionArg = arg{
	"Show the version and exit",
	"version", "",
	nil, nil, func(o options, exec string) (effect, error) {
		return func() error { fmt.Println(version()); os.Exit(0); return nil }, nil
	},
	func(o options) []string { return nil },
}

var helpArg = arg{
	"Print this help",
	"help", "",
	nil, nil, func(o options, exec string) (effect, error) {
		return func() error { _ = printHelp(exec); os.Exit(64); return nil }, nil
	},
	func(o options) []string { return nil },
}

func init() {
	args = []arg{
		configArg,
		workDirArg,
		hostArg,
		portArg,
		serviceArg,
		logfileArg,
		pidfileArg,
		checkConfigArg,
		noCheckUpdateArg,
		disableMemoryOptimizationArg,
		verboseArg,
		glinetArg,
		versionArg,
		helpArg,
	}
}

func getUsageLines(exec string, args []arg) []string {
	usage := []string{
		"Usage:",
		"",
		fmt.Sprintf("%s [options]", exec),
		"",
		"Options:",
	}
	for _, arg := range args {
		val := ""
		if arg.updateWithValue != nil {
			val = " VALUE"
		}
		if arg.shortName != "" {
			usage = append(usage, fmt.Sprintf("  -%s, %-30s %s",
				arg.shortName,
				"--"+arg.longName+val,
				arg.description))
		} else {
			usage = append(usage, fmt.Sprintf("  %-34s %s",
				"--"+arg.longName+val,
				arg.description))
		}
	}
	return usage
}

func printHelp(exec string) error {
	for _, line := range getUsageLines(exec, args) {
		_, err := fmt.Println(line)
		if err != nil {
			return err
		}
	}
	return nil
}

func argMatches(a arg, v string) bool {
	return v == "--"+a.longName || (a.shortName != "" && v == "-"+a.shortName)
}

func parse(exec string, ss []string) (o options, f effect, err error) {
	for i := 0; i < len(ss); i++ {
		v := ss[i]
		knownParam := false
		for _, arg := range args {
			if argMatches(arg, v) {
				if arg.updateWithValue != nil {
					if i+1 >= len(ss) {
						return o, f, fmt.Errorf("got %s without argument", v)
					}
					i++
					o, err = arg.updateWithValue(o, ss[i])
					if err != nil {
						return
					}
				} else if arg.updateNoValue != nil {
					o, err = arg.updateNoValue(o)
					if err != nil {
						return
					}
				} else if arg.effect != nil {
					var eff effect
					eff, err = arg.effect(o, exec)
					if err != nil {
						return
					}
					if eff != nil {
						prevf := f
						f = func() error {
							var err error
							if prevf != nil {
								err = prevf()
							}
							if err == nil {
								err = eff()
							}
							return err
						}
					}
				}
				knownParam = true
				break
			}
		}
		if !knownParam {
			return o, f, fmt.Errorf("unknown option %v", v)
		}
	}

	return
}

func shortestFlag(a arg) string {
	if a.shortName != "" {
		return "-" + a.shortName
	}
	return "--" + a.longName
}

func serialize(o options) []string {
	ss := []string{}
	for _, arg := range args {
		s := arg.serialize(o)
		if s != nil {
			ss = append(ss, append([]string{shortestFlag(arg)}, s...)...)
		}
	}
	return ss
}
