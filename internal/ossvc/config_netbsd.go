//go:build netbsd

package ossvc

import "github.com/kardianos/service"

// configureOSOptions defines additional settings of the service configuration
// on NetBSD.  conf must not be nil.
func configureOSOptions(_ *service.Config) {}
