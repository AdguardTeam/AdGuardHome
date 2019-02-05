// +build !windows,!nacl,!plan9

package main

import (
	"log"
	"log/syslog"
)

// configureSyslog reroutes standard logger output to syslog
func configureSyslog() error {
	w, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, serviceName)
	if err != nil {
		return err
	}
	log.SetOutput(w)
	return nil
}
