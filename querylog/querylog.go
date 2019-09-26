package querylog

import (
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
)

// QueryLog - main interface
type QueryLog interface {
	// Close query log object
	Close()

	// Set new configuration at runtime
	// Currently only 'Interval' field is supported.
	Configure(conf Config)

	// Add a log entry
	Add(question *dns.Msg, answer *dns.Msg, result *dnsfilter.Result, elapsed time.Duration, addr net.Addr, upstream string)

	// Get log entries
	GetData(params GetDataParams) []map[string]interface{}

	// Clear memory buffer and remove log files
	Clear()
}

// Config - configuration object
type Config struct {
	BaseDir  string // directory where log file is stored
	Interval uint32 // interval to rotate logs (in hours)
}

// New - create a new instance of the query log
func New(conf Config) QueryLog {
	return newQueryLog(conf)
}

// GetDataParams - parameters for GetData()
type GetDataParams struct {
	OlderThan         time.Time          // return entries that are older than this value
	Domain            string             // filter by domain name in question
	Client            string             // filter by client IP
	QuestionType      uint16             // filter by question type
	ResponseStatus    ResponseStatusType // filter by response status
	StrictMatchDomain bool               // if Domain value must be matched strictly
	StrictMatchClient bool               // if Client value must be matched strictly
}

// ResponseStatusType - response status
type ResponseStatusType int32

// Response status constants
const (
	ResponseStatusAll ResponseStatusType = iota + 1
	ResponseStatusFiltered
)
