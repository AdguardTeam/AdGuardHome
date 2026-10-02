package configmgr

import (
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
)

// StatsConfig is the on-disk statistics configuration.
type StatsConfig struct {
	// DirPath is the custom directory for statistics.  If it's empty the
	// default directory is used.
	DirPath string `yaml:"dir_path"`

	// Ignored is the list of host names, which should not be counted.
	Ignored []string `yaml:"ignored"`

	// Interval is the retention interval for statistics.
	Interval timeutil.Duration `yaml:"interval"`

	// Enabled defines if the statistics are enabled.
	Enabled bool `yaml:"enabled"`

	// IgnoredEnabled defines whether hosts from the ignored list should be
	// ignored.
	IgnoredEnabled bool `yaml:"ignored_enabled"`
}

// type check
var _ validate.Interface = (*StatsConfig)(nil)

// Validate implements the [validate.Interface] interface for *StatsConfig.
func (c *StatsConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(d.kolyshev):  Add more validations.

	return nil
}
