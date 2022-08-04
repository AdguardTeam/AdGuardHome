// Package stats provides units for managing statistics of the filtering DNS
// server.
package stats

import (
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// UnitIDGenFunc is the signature of a function that generates a unique ID for
// the statistics unit.
type UnitIDGenFunc func() (id uint32)

// DiskConfig is the configuration structure that is stored in file.
type DiskConfig struct {
	// Interval is the number of days for which the statistics are collected
	// before flushing to the database.
	Interval uint32 `yaml:"statistics_interval"`
}

// Config is the configuration structure for the statistics collecting.
type Config struct {
	// UnitID is the function to generate the identifier for current unit.  If
	// nil, the default function is used, see newUnitID.
	UnitID UnitIDGenFunc

	// ConfigModified will be called each time the configuration changed via web
	// interface.
	ConfigModified func()

	// HTTPRegister is the function that registers handlers for the stats
	// endpoints.
	HTTPRegister aghhttp.RegisterFunc

	// Filename is the name of the database file.
	Filename string

	// LimitDays is the maximum number of days to collect statistics into the
	// current unit.
	LimitDays uint32
}

// Interface is the statistics interface to be used by other packages.
type Interface interface {
	// Start begins the statistics collecting.
	Start()

	// Close stops the statistics collecting.
	Close()

	// Update collects the incoming statistics data.
	Update(e Entry)

	// GetTopClientIP returns at most limit IP addresses corresponding to the
	// clients with the most number of requests.
	GetTopClientsIP(limit uint) []net.IP

	// WriteDiskConfig puts the Interface's configuration to the dc.
	WriteDiskConfig(dc *DiskConfig)
}

// TimeUnit is the unit of measuring time while aggregating the statistics.
type TimeUnit int

// Supported TimeUnit values.
const (
	Hours TimeUnit = iota
	Days
)

// Result is the resulting code of processing the DNS request.
type Result int

// Supported Result values.
//
// TODO(e.burkov):  Think about better naming.
const (
	RNotFiltered Result = iota + 1
	RFiltered
	RSafeBrowsing
	RSafeSearch
	RParental

	resultLast = RParental + 1
)

// Entry is a statistics data entry.
type Entry struct {
	// Clients is the client's primary ID.
	//
	// TODO(a.garipov): Make this a {net.IP, string} enum?
	Client string

	// Domain is the domain name requested.
	Domain string

	// Result is the result of processing the request.
	Result Result

	// Time is the duration of the request processing in milliseconds.
	Time uint32
}
