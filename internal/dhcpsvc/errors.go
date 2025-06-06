package dhcpsvc

import (
	"fmt"
)

// newMustErr returns an error that indicates that valName must be as must
// describes.
//
// TODO(e.burkov):  Use [validate] and remove this function.
func newMustErr(valName, must string, val fmt.Stringer) (err error) {
	return fmt.Errorf("%s %s must %s", valName, val, must)
}
