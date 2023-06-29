package configmgr

import (
	"fmt"

	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/exp/constraints"
)

// numberOrDuration is the constraint for integer types along with
// timeutil.Duration.
type numberOrDuration interface {
	constraints.Integer | timeutil.Duration
}

// newMustBePositiveError returns an error about the value that must be positive
// but isn't.  prop is the name of the property to mention in the error message.
//
// TODO(a.garipov): Consider moving such helpers to golibs and use in AdGuardDNS
// as well.
func newMustBePositiveError[T numberOrDuration](prop string, v T) (err error) {
	if s, ok := any(v).(fmt.Stringer); ok {
		return fmt.Errorf("%s must be positive, got %s", prop, s)
	}

	return fmt.Errorf("%s must be positive, got %d", prop, v)
}
