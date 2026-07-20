package dhcpsvc

import (
	"context"
)

// Database is the interface for storing DHCP leases.
type Database interface {
	// Load loads leases from the database.  If err is not nil, leases must be
	// nil.  It must be safe for concurrent use.
	Load(ctx context.Context) (leases []*Lease, err error)

	// Store stores leases to the database.  leases must be valid.  It must be
	// safe for concurrent use.
	Store(ctx context.Context, leases []*Lease) (err error)
}

// EmptyDatabase is a [Database] implementation that does nothing.
type EmptyDatabase struct{}

// type check
var _ Database = EmptyDatabase{}

// Load implements the [Database] interface for EmptyDatabase.  It always
// returns nil value and nil error.
func (EmptyDatabase) Load(_ context.Context) (leases []*Lease, err error) {
	return nil, nil
}

// Store implements the [Database] interface for EmptyDatabase.  It always
// returns nil.
func (EmptyDatabase) Store(_ context.Context, _ []*Lease) (err error) {
	return nil
}
