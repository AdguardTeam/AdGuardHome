package cmd

import (
	"fmt"
	"os"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/log"
)

// syslogServiceName is the name of the AdGuard Home service used for writing
// logs to the system log.
const syslogServiceName = "AdGuardHome"

// setLog sets up the text logging.
//
// TODO(a.garipov): Add parameters from configuration file.
func setLog(opts *options) (err error) {
	switch opts.confFile {
	case "stdout":
		log.SetOutput(os.Stdout)
	case "stderr":
		log.SetOutput(os.Stderr)
	case "syslog":
		err = aghos.ConfigureSyslog(syslogServiceName)
		if err != nil {
			return fmt.Errorf("initializing syslog: %w", err)
		}
	default:
		// TODO(a.garipov): Use the path.
	}

	if opts.verbose {
		log.SetLevel(log.DEBUG)
		log.Debug("verbose logging enabled")
	}

	return nil
}
