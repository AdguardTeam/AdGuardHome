package dhcpsvc

import (
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
)

const (
	// errNilConfig is returned when a nil config met.
	errNilConfig errors.Error = "config is nil"

	// errNoInterfaces is returned when no interfaces found in configuration.
	errNoInterfaces errors.Error = "no interfaces specified"
)

// newMustErr returns an error that indicates that valName must be as must
// describes.
func newMustErr(valName, must string, val fmt.Stringer) (err error) {
	return fmt.Errorf("%s %s must %s", valName, val, must)
}
