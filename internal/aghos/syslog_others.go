//go:build !windows

package aghos

import (
	"log/syslog"

	"github.com/AdguardTeam/golibs/log"
)

// configureSyslog sets standard log output to syslog.
func configureSyslog(serviceName string) (err error) {
	w, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, serviceName)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	log.SetOutput(w)

	return nil
}
