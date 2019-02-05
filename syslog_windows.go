package main

import (
	"log"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/eventlog"
)

// should be the same as the service name!
const eventLogSrc = "AdGuardHome"

type eventLogWriter struct {
	el *eventlog.Log
}

// Write sends a log message to the Event Log.
func (w *eventLogWriter) Write(b []byte) (int, error) {
	return len(b), w.el.Info(1, string(b))
}

func configureSyslog() error {
	// Continue if we receive "registry key already exists" or if we get
	// ERROR_ACCESS_DENIED so that we can log without administrative permissions
	// for pre-existing eventlog sources.
	if err := eventlog.InstallAsEventCreate(eventLogSrc, eventlog.Info|eventlog.Warning|eventlog.Error); err != nil {
		if !strings.Contains(err.Error(), "registry key already exists") && err != windows.ERROR_ACCESS_DENIED {
			return err
		}
	}
	el, err := eventlog.Open(eventLogSrc)
	if err != nil {
		return err
	}

	log.SetOutput(&eventLogWriter{el: el})
	return nil
}
