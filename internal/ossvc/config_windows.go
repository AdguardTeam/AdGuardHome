//go:build windows

package ossvc

import "github.com/kardianos/service"

// configureOSOptions defines additional settings of the service
// configuration on Windows.
func configureOSOptions(_ *service.Config) {}
