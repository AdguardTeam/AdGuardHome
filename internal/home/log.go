package home

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	yaml "go.yaml.in/yaml/v4"
	"gopkg.in/natefinch/lumberjack.v2"
)

// configSyslog is used to indicate that syslog or eventlog (win) should be used
// for logger output.
const configSyslog = "syslog"

// logSettings are the logging settings part of the configuration file.
type logSettings struct {
	// file is the path to the log file.  If empty, logs are written to stdout.
	// If "syslog", logs are written to syslog.
	file string

	// maxAge is the maximum duration for retaining old log files, in days.
	maxAge int

	// maxBackups is the maximum number of old log files to retain.
	//
	// NOTE: maxAge may still cause them to get deleted.
	maxBackups int

	// maxSize is the maximum size of the log file before it gets rotated, in
	// megabytes.  The default value is 100 MB.
	maxSize int

	// compress determines, if the rotated log files should be compressed using
	// gzip.
	compress bool

	// enabled indicates whether logging is enabled.
	enabled bool

	// localTime determines, if the time used for formatting the timestamps in
	// is the computer's local time.
	localTime bool

	// verbose determines, if verbose (aka debug) logging is enabled.
	verbose bool
}

// newSlogLogger returns new [*slog.Logger] configured with the given settings.
// ls must not be nil.
func newSlogLogger(ls *logSettings) (l *slog.Logger) {
	if !ls.enabled {
		return slogutil.NewDiscardLogger()
	}

	lvl := slog.LevelInfo
	if ls.verbose {
		lvl = slog.LevelDebug

		log.SetLevel(log.DEBUG)
	}

	return slogutil.New(&slogutil.Config{
		Format:       slogutil.FormatAdGuardLegacy,
		Level:        lvl,
		AddTimestamp: true,
	})
}

// configureLogger configures logger output.  ls must not be nil.
func configureLogger(ls *logSettings, workDir string) (err error) {
	// Make sure that we see the microseconds in logs, as networking stuff can
	// happen pretty quickly.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Write logs to stdout by default.
	if ls.file == "" {
		return nil
	}

	if ls.file == configSyslog {
		// Use syslog where it is possible and eventlog on Windows.
		err = aghos.ConfigureSyslog(serviceName)
		if err != nil {
			return fmt.Errorf("cannot initialize syslog: %w", err)
		}

		return nil
	}

	logFilePath := ls.file
	if !filepath.IsAbs(logFilePath) {
		logFilePath = filepath.Join(workDir, logFilePath)
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   logFilePath,
		Compress:   ls.compress,
		LocalTime:  ls.localTime,
		MaxBackups: ls.maxBackups,
		MaxSize:    ls.maxSize,
		MaxAge:     ls.maxAge,
	})

	return err
}

// Default log constants.
const (
	defaultLogMaxAge  = 3
	defaultLogMaxSize = 100
)

// newLogSettings returns a *logSettings properly initialized from opts.  l must
// not be nil.
func newLogSettings(
	ctx context.Context,
	l *slog.Logger,
	opts options,
	workDir string,
	confPath string,
) (ls *logSettings) {
	ls = readLogSettings(ctx, l, workDir, confPath)
	if ls == nil {
		// Use default log settings.
		ls = &logSettings{
			enabled: true,
			maxAge:  defaultLogMaxAge,
			maxSize: defaultLogMaxSize,
		}
	}

	config.Log = ls.toLogConf()

	// Command-line arguments can override config settings.
	if opts.verbose {
		ls.verbose = true
	}

	ls.file = cmp.Or(opts.logFile, ls.file)

	if opts.runningAsService && ls.file == "" && runtime.GOOS == "windows" {
		// When running as a Windows service, use eventlog by default if
		// nothing else is configured.  Otherwise, we'll lose the log output.
		ls.file = configSyslog
	}

	return ls
}

// readLogSettings reads logging settings from the config file.  We do it in a
// separate method in order to configure logger before the actual configuration
// is parsed and applied.  l must not be nil.
func readLogSettings(
	ctx context.Context,
	l *slog.Logger,
	workDir string,
	confPath string,
) (ls *logSettings) {
	yamlFile, err := readConfigFile(ctx, l, workDir, confPath)
	if err != nil {
		l.DebugContext(ctx, "reading config file", slogutil.KeyError, err)

		return nil
	}

	conf := &configuration{}
	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		l.ErrorContext(ctx, "getting logging settings from config", slogutil.KeyError, err)
	}

	err = conf.Log.Validate()
	if err != nil {
		l.ErrorContext(ctx, "reading logging settings from config", slogutil.KeyError, err)

		return nil
	}

	return logConfToInternal(conf.Log)
}

// logConfToInternal converts c to the log settings.  c must be valid.
func logConfToInternal(c *configmgr.LogConfig) (s *logSettings) {
	if c == nil {
		return &logSettings{
			enabled: true,
		}
	}

	return &logSettings{
		enabled:    c.Enabled,
		file:       c.File,
		maxAge:     c.MaxAge,
		maxBackups: c.MaxBackups,
		maxSize:    c.MaxSize,
		compress:   c.Compress,
		localTime:  c.LocalTime,
		verbose:    c.Verbose,
	}
}

// toLogConf converts s to the on-disk logging configuration.  s must be valid.
func (s *logSettings) toLogConf() (c *configmgr.LogConfig) {
	return &configmgr.LogConfig{
		File:       s.file,
		MaxAge:     s.maxAge,
		MaxBackups: s.maxBackups,
		MaxSize:    s.maxSize,
		Compress:   s.compress,
		Enabled:    s.enabled,
		LocalTime:  s.localTime,
		Verbose:    s.verbose,
	}
}
