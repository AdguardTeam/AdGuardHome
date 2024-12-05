package configmgr

import (
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/exp/constraints"
)

// validator is the interface for configuration entities that can validate
// themselves.
type validator interface {
	// validate returns an error if the entity isn't valid.
	validate() (err error)
}

// numberOrDuration is the constraint for integer types along with
// timeutil.Duration.
type numberOrDuration interface {
	constraints.Integer | timeutil.Duration
}

// newErrNotPositive returns an error about the value that must be positive but
// isn't.  prop is the name of the property to mention in the error message.
//
// TODO(a.garipov): Consider moving such helpers to golibs and use in AdGuardDNS
// as well.
func newErrNotPositive[T numberOrDuration](prop string, v T) (err error) {
	return fmt.Errorf("%s: %w, got %v", prop, errors.ErrNotPositive, v)
}
