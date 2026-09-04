package configmgr

import (
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// LogConfig is the on-disk logging configuration.
type LogConfig struct {
	// File is the path to the log file.  If empty, logs are written to stdout.
	// If "syslog", logs are written to syslog.
	File string `yaml:"file"`

	// MaxAge is the maximum duration for retaining old log files, in days.
	MaxAge int `yaml:"max_age"`

	// MaxBackups is the maximum number of old log files to retain.
	//
	// NOTE: MaxAge may still cause them to get deleted.
	MaxBackups int `yaml:"max_backups"`

	// MaxSize is the maximum size of the log file before it gets rotated, in
	// megabytes.
	MaxSize int `yaml:"max_size"`

	// Compress determines, if the rotated log files should be compressed using
	// gzip.
	Compress bool `yaml:"compress"`

	// Enabled indicates whether logging is enabled.
	Enabled bool `yaml:"enabled"`

	// LocalTime determines, if the time used for formatting the timestamps in
	// is the computer's local time.
	LocalTime bool `yaml:"local_time"`

	// Verbose determines, if verbose (aka debug) logging is enabled.
	Verbose bool `yaml:"verbose"`
}

// type check
var _ validate.Interface = (*LogConfig)(nil)

// Validate implements the [validate.Interface] interface for *LogConfig.
//
// TODO(d.kolyshev): Add more validations.
func (c *LogConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	return nil
}
