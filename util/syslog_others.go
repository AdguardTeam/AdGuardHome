// +build !windows,!nacl,!plan9

package util

import (
	"log"
	"log/syslog"
)

// ConfigureSyslog reroutes standard logger output to syslog
func ConfigureSyslog(serviceName string) error {
	w, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, serviceName)
	if err != nil {
		return err
	}
	log.SetOutput(w)
	return nil
}
