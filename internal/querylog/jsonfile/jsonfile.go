package jsonfile

import (
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/querylog/logs"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
)

// New creates a new instance of the query log.
func New(conf logs.Config) logs.Api {
	return newQueryLog(conf)
}

// newQueryLog crates a new queryLog.
func newQueryLog(conf logs.Config) (l *queryLog) {
	findClient := conf.FindClient
	if findClient == nil {
		findClient = func(_ []string) (_ *logs.Client, _ error) {
			return nil, nil
		}
	}

	l = &queryLog{
		findClient: findClient,

		logFile:    filepath.Join(conf.BaseDir, queryLogFileName),
		anonymizer: conf.Anonymizer,
	}

	l.conf = &logs.Config{}
	*l.conf = conf

	if !checkInterval(conf.RotationIvl) {
		log.Info(
			"querylog: warning: unsupported rotation interval %s, setting to 1 day",
			conf.RotationIvl,
		)
		l.conf.RotationIvl = timeutil.Day
	}

	return l
}
