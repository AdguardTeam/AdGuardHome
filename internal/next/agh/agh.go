// Package agh contains common entities and interfaces of AdGuard Home.
package agh

import "context"

// Service is the interface for API servers.
//
// TODO(a.garipov): Consider adding a context to Start.
//
// TODO(a.garipov): Consider adding a Wait method or making an extension
// interface for that.
type Service interface {
	// Start starts the service.  It does not block.
	Start() (err error)

	// Shutdown gracefully stops the service.  ctx is used to determine
	// a timeout before trying to stop the service less gracefully.
	Shutdown(ctx context.Context) (err error)
}

// type check
var _ Service = EmptyService{}

// EmptyService is a [Service] that does nothing.
//
// TODO(a.garipov): Remove if unnecessary.
type EmptyService struct{}

// Start implements the [Service] interface for EmptyService.
func (EmptyService) Start() (err error) { return nil }

// Shutdown implements the [Service] interface for EmptyService.
func (EmptyService) Shutdown(_ context.Context) (err error) { return nil }

// ServiceWithConfig is an extension of the [Service] interface for services
// that can return their configuration.
//
// TODO(a.garipov): Consider removing this generic interface if we figure out
// how to make it testable in a better way.
type ServiceWithConfig[ConfigType any] interface {
	Service

	Config() (c ConfigType)
}

// type check
var _ ServiceWithConfig[struct{}] = (*EmptyServiceWithConfig[struct{}])(nil)

// EmptyServiceWithConfig is a ServiceWithConfig that does nothing.  Its Config
// method returns Conf.
//
// TODO(a.garipov): Remove if unnecessary.
type EmptyServiceWithConfig[ConfigType any] struct {
	EmptyService

	Conf ConfigType
}

// Config implements the [ServiceWithConfig] interface for
// *EmptyServiceWithConfig.
func (s *EmptyServiceWithConfig[ConfigType]) Config() (conf ConfigType) {
	return s.Conf
}
