// Package ossvc contains abstractions and utilities for platform-independent
// service management.
//
// TODO(e.burkov):  Add tests.
package ossvc

import (
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/kardianos/service"
)

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

	// StatusRestartOnFail means that the service is restarting after failed
	// start.
	StatusRestartOnFail Status = "restart on fail"
)

// statusToInternal converts a service.Status to a Status.
//
// TODO(e.burkov):  Get rid of [service] package dependency and remove this
// function.
func statusToInternal(status service.Status) (s Status, err error) {
	switch status {
	case service.StatusRunning:
		return StatusRunning, nil
	case service.StatusStopped:
		return StatusStopped, nil
	case statusRestartOnFail:
		return StatusRestartOnFail, nil
	default:
		return "", fmt.Errorf("service status: %w: %v", errors.ErrBadEnumValue, status)
	}
}
