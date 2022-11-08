package logs

import (
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/miekg/dns"
)

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
func (p *AddParams) Validate() (err error) {
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
