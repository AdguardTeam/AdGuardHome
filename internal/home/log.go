package home

import (
	"cmp"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

// configSyslog is used to indicate that syslog or eventlog (win) should be used
// for logger output.
const configSyslog = "syslog"

// newSlogLogger returns new [*slog.Logger] configured with the given settings.
func newSlogLogger(ls *logSettings) (l *slog.Logger) {
	if !ls.Enabled {
		return slogutil.NewDiscardLogger()
	}

	lvl := slog.LevelInfo
	if ls.Verbose {
		lvl = slog.LevelDebug
	}

	return slogutil.New(&slogutil.Config{
		Format:       slogutil.FormatAdGuardLegacy,
		Level:        lvl,
		AddTimestamp: true,
	})
}

// configureLogger configures logger level and output.
func configureLogger(ls *logSettings) (err error) {
	// Configure logger level.
	if !ls.Enabled {
		log.SetLevel(log.OFF)
	} else if ls.Verbose {
		log.SetLevel(log.DEBUG)
	}

	// Make sure that we see the microseconds in logs, as networking stuff can
	// happen pretty quickly.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Write logs to stdout by default.
	if ls.File == "" {
		return nil
	}

	if ls.File == configSyslog {
		// Use syslog where it is possible and eventlog on Windows.
		err = aghos.ConfigureSyslog(serviceName)
		if err != nil {
			return fmt.Errorf("cannot initialize syslog: %w", err)
		}

		return nil
	}

	logFilePath := ls.File
	if !filepath.IsAbs(logFilePath) {
		logFilePath = filepath.Join(Context.workDir, logFilePath)
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   logFilePath,
		Compress:   ls.Compress,
		LocalTime:  ls.LocalTime,
		MaxBackups: ls.MaxBackups,
		MaxSize:    ls.MaxSize,
		MaxAge:     ls.MaxAge,
	})

	return err
}

// getLogSettings returns a log settings object properly initialized from opts.
func getLogSettings(opts options) (ls *logSettings) {
	configLogSettings := config.Log

	ls = readLogSettings()
	if ls == nil {
		// Use default log settings.
		ls = &configLogSettings
	}

	// Command-line arguments can override config settings.
	if opts.verbose {
		ls.Verbose = true
	}

	ls.File = cmp.Or(opts.logFile, ls.File)

	if opts.runningAsService && ls.File == "" && runtime.GOOS == "windows" {
		// When running as a Windows service, use eventlog by default if
		// nothing else is configured.  Otherwise, we'll lose the log output.
		ls.File = configSyslog
	}

	return ls
}

// readLogSettings reads logging settings from the config file.  We do it in a
// separate method in order to configure logger before the actual configuration
// is parsed and applied.
func readLogSettings() (ls *logSettings) {
	// TODO(s.chzhen):  Add a helper function that returns default parameters
	// for this structure and for the global configuration structure [config].
	conf := &configuration{
		Log: logSettings{
			// By default, it is true if the property does not exist.
			Enabled: true,
		},
	}

	yamlFile, err := readConfigFile()
	if err != nil {
		return nil
	}

	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		log.Error("Couldn't get logging settings from the configuration: %s", err)
	}

	return &conf.Log
}
