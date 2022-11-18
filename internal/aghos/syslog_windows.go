//go:build windows

package aghos

import (
	"strings"

	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/eventlog"
)

type eventLogWriter struct {
	el *eventlog.Log
}

// Write implements io.Writer interface for eventLogWriter.
func (w *eventLogWriter) Write(b []byte) (int, error) {
	return len(b), w.el.Info(1, string(b))
}

func configureSyslog(serviceName string) error {
	// Note that the eventlog src is the same as the service name
	// Otherwise, we will get "the description for event id cannot be found" warning in every log record

	// Continue if we receive "registry key already exists" or if we get
	// ERROR_ACCESS_DENIED so that we can log without administrative permissions
	// for pre-existing eventlog sources.
	if err := eventlog.InstallAsEventCreate(serviceName, eventlog.Info|eventlog.Warning|eventlog.Error); err != nil {
		if !strings.Contains(err.Error(), "registry key already exists") && err != windows.ERROR_ACCESS_DENIED {
			return err
		}
	}
	el, err := eventlog.Open(serviceName)
	if err != nil {
		return err
	}

	log.SetOutput(&eventLogWriter{el: el})
	return nil
}
