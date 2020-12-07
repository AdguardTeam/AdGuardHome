// +build !windows,!nacl,!plan9

package sysutil

import (
	"log"
	"log/syslog"
)

func configureSyslog(serviceName string) error {
	w, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, serviceName)
	if err != nil {
		return err
	}
	log.SetOutput(w)
	return nil
}
