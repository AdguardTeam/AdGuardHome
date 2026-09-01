// Package configmgr defines AdGuard Home on-disk configuration entities.
package configmgr

import (
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// Config is the top-level on-disk configuration structure.
//
// TODO(d.kolyshev): Use.
type Config struct {
	// Log is a block with log configuration settings.
	Log *LogConfig `yaml:"log"`
}

// type check
var _ validate.Interface = (*Config)(nil)

// Validate implements the [validate.Interface] interface for *Config.
func (c *Config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// Keep this in the same order as the fields in the config.
	validators := container.KeyValues[string, validate.Interface]{{
		Key:   "log",
		Value: c.Log,
	}}

	var errs []error
	for _, kv := range validators {
		errs = validate.Append(errs, kv.Key, kv.Value)
	}

	return errors.Join(errs...)
}
