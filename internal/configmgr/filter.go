package configmgr

import (
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/AdguardTeam/urlfilter/rules"
)

// Filter represents the on-disk filter list.
type Filter struct {
	// URL is the location of the filter list, which can be a URL or an absolute
	// file path.
	URL *urlutil.URL `yaml:"url"`

	// Name is the human-readable name of the filter list.
	Name string `yaml:"name"`

	// ID is automatically assigned when filter is added.
	ID rules.ListID `yaml:"id"`

	// Enabled indicates whether the filter list is used.
	Enabled bool `yaml:"enabled"`
}

// type check
var _ validate.Interface = (*Filter)(nil)

// Validate implements the [validate.Interface] interface for *Filter.
func (f *Filter) Validate() (err error) {
	if f == nil {
		return errors.ErrNoValue
	}

	// TODO(d.kolyshev):  Add more validations.

	return nil
}
