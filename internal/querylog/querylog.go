package querylog

import (
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// QueryLog - main interface
type QueryLog interface {
	Start()

	// Close query log object
	Close()

	// Add a log entry
	Add(params AddParams)

	// WriteDiskConfig - write configuration
	WriteDiskConfig(c *Config)
}

// Config - configuration object
type Config struct {
	// ConfigModified is called when the configuration is changed, for
	// example by HTTP requests.
	ConfigModified func()

	// HTTPRegister registers an HTTP handler.
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))

	// FindClient returns client information by their IDs.
	FindClient func(ids []string) (c *Client, err error)

	// BaseDir is the base directory for log files.
	BaseDir string

	// RotationIvl is the interval for log rotation.  After that period, the
	// old log file will be renamed, NOT deleted, so the actual log
	// retention time is twice the interval.  The value must be one of:
	//
	//         6 * time.Hour
	//        24 * time.Hour
	//    7 * 24 * time.Hour
	//   30 * 24 * time.Hour
	//   90 * 24 * time.Hour
	//
	RotationIvl time.Duration

	// MemSize is the number of entries kept in a memory buffer before they
	// are flushed to disk.
	MemSize uint32

	// Enabled tells if the query log is enabled.
	Enabled bool

	// FileEnabled tells if the query log writes logs to files.
	FileEnabled bool

	// AnonymizeClientIP tells if the query log should anonymize clients' IP
	// addresses.
	AnonymizeClientIP bool
}

// AddParams - parameters for Add()
type AddParams struct {
	Question    *dns.Msg
	Answer      *dns.Msg          // The response we sent to the client (optional)
	OrigAnswer  *dns.Msg          // The response from an upstream server (optional)
	Result      *filtering.Result // Filtering result (optional)
	Elapsed     time.Duration     // Time spent for processing the request
	ClientID    string
	ClientIP    net.IP
	Upstream    string // Upstream server URL
	ClientProto ClientProto
}

// validate returns an error if the parameters aren't valid.
func (p *AddParams) validate() (err error) {
	switch {
	case p.Question == nil:
		return errors.Error("question is nil")
	case len(p.Question.Question) != 1:
		return errors.Error("more than one question")
	case len(p.Question.Question[0].Name) == 0:
		return errors.Error("no host in question")
	case p.ClientIP == nil:
		return errors.Error("no client ip")
	default:
		return nil
	}
}

// New creates a new instance of the query log.
func New(conf Config) (ql QueryLog) {
	return newQueryLog(conf)
}

// newQueryLog crates a new queryLog.
func newQueryLog(conf Config) (l *queryLog) {
	findClient := conf.FindClient
	if findClient == nil {
		findClient = func(_ []string) (_ *Client, _ error) {
			return nil, nil
		}
	}

	l = &queryLog{
		findClient: findClient,

		logFile: filepath.Join(conf.BaseDir, queryLogFileName),
	}

	l.conf = &Config{}
	*l.conf = conf

	if !checkInterval(conf.RotationIvl) {
		log.Info(
			"querylog: warning: unsupported rotation interval %d, setting to 1 day",
			conf.RotationIvl,
		)
		l.conf.RotationIvl = 24 * time.Hour
	}

	return l
}
