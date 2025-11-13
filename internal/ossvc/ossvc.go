// Package ossvc contains abstractions and utilities for platform-independent
// service management.
//
// TODO(e.burkov):  Add tests.
package ossvc

// ServiceName is the name of a service.
//
// TODO(e.burkov):  Validate for each platform.
type ServiceName string

// Status represents the status of a service.
type Status string

const (
	// StatusNotInstalled means that the service is not installed.
	StatusNotInstalled Status = "not installed"

	// StatusStopped means that the service is stopped.
	StatusStopped Status = "stopped"

	// StatusRunning means that the service is running.
	StatusRunning Status = "running"
)
