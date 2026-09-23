package configmgr

import (
	"github.com/AdguardTeam/golibs/validate"
)

// OSConfig is the on-disk OS-related configuration.
type OSConfig struct {
	// Group is the name of the group which AdGuard Home must switch to on
	// startup.  Empty string means no switching.
	Group string `yaml:"group"`

	// User is the name of the user which AdGuard Home must switch to on
	// startup.  Empty string means no switching.
	User string `yaml:"user"`

	// RlimitNoFile is the maximum number of opened fd's per process.  Zero
	// means to use the default value.
	RlimitNoFile uint64 `yaml:"rlimit_nofile"`
}

// type check
var _ validate.Interface = (*OSConfig)(nil)

// Validate implements the [validate.Interface] interface for *OSConfig.
func (c *OSConfig) Validate() (err error) {
	// TODO(d.kolyshev):  Validate.

	return nil
}
