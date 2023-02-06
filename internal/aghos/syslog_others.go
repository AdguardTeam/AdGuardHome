//go:build !windows

package aghos

import (
	"log/syslog"

	"github.com/AdguardTeam/golibs/log"
)

func configureSyslog(serviceName string) error {
	w, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, serviceName)
	if err != nil {
		return err
	}
	log.SetOutput(w)
	return nil
}
