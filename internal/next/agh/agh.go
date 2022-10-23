// Package agh contains common entities and interfaces of AdGuard Home.
//
// TODO(a.garipov): Move to the upper-level internal/.
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

// EmptyService is a Service that does nothing.
type EmptyService struct{}

// Start implements the Service interface for EmptyService.
func (EmptyService) Start() (err error) { return nil }

// Shutdown implements the Service interface for EmptyService.
func (EmptyService) Shutdown(_ context.Context) (err error) { return nil }
