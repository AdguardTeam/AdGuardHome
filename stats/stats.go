// Module for managing statistics for DNS filtering server

package stats

import (
	"net"
	"net/http"
)

type unitIDCallback func() uint32

// DiskConfig - configuration settings that are stored on disk
type DiskConfig struct {
	Interval uint32 `yaml:"statistics_interval"` // time interval for statistics (in days)
}

// Config - module configuration
type Config struct {
	Filename          string         // database file name
	LimitDays         uint32         // time limit (in days)
	UnitID            unitIDCallback // user function to get the current unit ID.  If nil, the current time hour is used.
	AnonymizeClientIP bool           // anonymize clients' IP addresses

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))

	limit uint32 // maximum time we need to keep data for (in hours)
}

// New - create object
func New(conf Config) (Stats, error) {
	return createObject(conf)
}

// Stats - main interface
type Stats interface {
	Start()

	// Close object.
	// This function is not thread safe
	//  (can't be called in parallel with any other function of this interface).
	Close()

	// Update counters
	Update(e Entry)

	// Get IP addresses of the clients with the most number of requests
	GetTopClientsIP(limit uint) []string

	// WriteDiskConfig - write configuration
	WriteDiskConfig(dc *DiskConfig)
}

// TimeUnit - time unit
type TimeUnit int

// Supported time units
const (
	Hours TimeUnit = iota
	Days
)

// Result of DNS request processing
type Result int

// Supported result values
const (
	RNotFiltered Result = iota + 1
	RFiltered
	RSafeBrowsing
	RSafeSearch
	RParental
	rLast
)

// Entry - data to add
type Entry struct {
	Domain string
	Client net.IP
	Result Result
	Time   uint32 // processing time (msec)
}
