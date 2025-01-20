// Package agh contains common entities and interfaces of AdGuard Home.
package agh

import (
	"github.com/AdguardTeam/golibs/service"
)

// ServiceWithConfig is an extension of the [Service] interface for services
// that can return their configuration.
//
// TODO(a.garipov): Consider removing this generic interface if we figure out
// how to make it testable in a better way.
type ServiceWithConfig[ConfigType any] interface {
	service.Interface

	// Config returns a deep clone of the configuration of the service.
	Config() (c ConfigType)
}

// type check
var _ ServiceWithConfig[struct{}] = (*EmptyServiceWithConfig[struct{}])(nil)

// EmptyServiceWithConfig is a ServiceWithConfig that does nothing.  Its Config
// method returns Conf.
//
// TODO(a.garipov): Remove if unnecessary.
type EmptyServiceWithConfig[ConfigType any] struct {
	service.Empty

	Conf ConfigType
}

// Config implements the [ServiceWithConfig] interface for
// *EmptyServiceWithConfig.
func (s *EmptyServiceWithConfig[ConfigType]) Config() (conf ConfigType) {
	return s.Conf
}
