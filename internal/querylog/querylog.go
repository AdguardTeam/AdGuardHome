package querylog

import (
	"net"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
)

// QueryLog - main interface
type QueryLog interface {
	Start()

	// Close query log object
	Close()

	// Add a log entry
	Add(params *AddParams)

	// WriteDiskConfig - write configuration
	WriteDiskConfig(c *Config)
}

// Config is the query log configuration structure.
type Config struct {
	// Anonymizer processes the IP addresses to anonymize those if needed.
	Anonymizer *aghnet.IPMut

	// ConfigModified is called when the configuration is changed, for
	// example by HTTP requests.
	ConfigModified func()

	// HTTPRegister registers an HTTP handler.
	HTTPRegister aghhttp.RegisterFunc

	// FindClient returns client information by their IDs.
	FindClient func(ids []string) (c *Client, err error)

	// BaseDir is the base directory for log files.
	BaseDir string

	// RotationIvl is the interval for log rotation.  After that period, the
	// old log file will be renamed, NOT deleted, so the actual log
	// retention time is twice the interval.  The value must be one of:
	//
	//    6 * time.Hour
	//    1 * timeutil.Day
	//    7 * timeutil.Day
	//   30 * timeutil.Day
	//   90 * timeutil.Day
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

// AddParams is the parameters for adding an entry.
type AddParams struct {
	Question *dns.Msg

	// ReqECS is the IP network extracted from EDNS Client-Subnet option of a
	// request.
	ReqECS *net.IPNet

	// Answer is the response which is sent to the client, if any.
	Answer *dns.Msg

	// OrigAnswer is the response from an upstream server.  It's only set if the
	// answer has been modified by filtering.
	OrigAnswer *dns.Msg

	// Result is the filtering result (optional).
	Result *filtering.Result

	ClientID string

	// Upstream is the URL of the upstream DNS server.
	Upstream string

	ClientProto ClientProto

	ClientIP net.IP

	// Elapsed is the time spent for processing the request.
	Elapsed time.Duration

	// Cached indicates if the response is served from cache.
	Cached bool

	// AuthenticatedData shows if the response had the AD bit set.
	AuthenticatedData bool
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

		logFile:    filepath.Join(conf.BaseDir, queryLogFileName),
		anonymizer: conf.Anonymizer,
	}

	l.conf = &Config{}
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
