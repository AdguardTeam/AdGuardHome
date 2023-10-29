package dhcpsvc

import "github.com/AdguardTeam/golibs/errors"

const (
	// errNilConfig is returned when a nil config met.
	errNilConfig errors.Error = "config is nil"

	// errNoInterfaces is returned when no interfaces found in configuration.
	errNoInterfaces errors.Error = "no interfaces specified"
)
