package dhcpsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/google/renameio/v2/maybe"
)

// jsonDataVersion is the current version of the stored DHCP leases structure.
const jsonDataVersion = 1

// jsonDatabasePerm is the permissions for the database file.
const jsonDatabasePerm fs.FileMode = 0o640

// jsonLeasesData is the structure of the data stored in the [JSONDatabase].
type jsonLeasesData struct {
	// Leases is the list containing stored DHCP leases.
	Leases []*jsonLease `json:"leases"`

	// Version is the current version of the structure.
	Version int `json:"version"`
}

// jsonLease is the structure of a lease stored in [JSONDatabase].
//
// TODO(e.burkov):  Migrate to add DUID and IAID fields for DHCPv6 leases.
type jsonLease struct {
	// Expiry is the expiration time of the lease in RFC 3339 format.  It is
	// empty for static leases.
	Expiry string `json:"expires"`

	// IP is the IP address leased to the client.  It must not be empty.
	IP netip.Addr `json:"ip"`

	// Hostname is the hostname of the client.
	Hostname string `json:"hostname"`

	// HWAddr is the MAC address of the client.  It must be a valid hardware
	// address string according to [netutil.IsValidMACString].
	HWAddr string `json:"mac"`

	// IsStatic defines if the lease is static.
	IsStatic bool `json:"static"`
}

// compareNames returns the result of comparing the hostnames of jl and other
// lexicographically.
func (jl *jsonLease) compareNames(other *jsonLease) (res int) {
	return strings.Compare(jl.Hostname, other.Hostname)
}

// toJSONLease converts *Lease to *jsonLease.  l must not be nil.
func toJSONLease(l *Lease) (jl *jsonLease) {
	var expiryStr string
	if !l.IsStatic {
		// The front-end is waiting for RFC 3999 format of the time value.  It
		// also shouldn't got an Expiry field for static leases.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2692.
		expiryStr = l.Expiry.Format(time.RFC3339)
	}

	return &jsonLease{
		Expiry:   expiryStr,
		Hostname: l.Hostname,
		HWAddr:   l.HWAddr.String(),
		IP:       l.IP,
		IsStatic: l.IsStatic,
	}
}

// toInternal converts jl to *Lease.
func (jl *jsonLease) toInternal() (l *Lease, err error) {
	mac, err := net.ParseMAC(jl.HWAddr)
	if err != nil {
		return nil, fmt.Errorf("parsing hardware address: %w", err)
	}

	expiry := time.Time{}
	if !jl.IsStatic {
		expiry, err = time.Parse(time.RFC3339, jl.Expiry)
		if err != nil {
			return nil, fmt.Errorf("parsing expiry time: %w", err)
		}
	}

	return &Lease{
		Expiry:   expiry,
		IP:       jl.IP,
		Hostname: jl.Hostname,
		HWAddr:   mac,
		IsStatic: jl.IsStatic,
	}, nil
}

// JSONDatabaseConfig is the configuration for [JSONDatabase].
type JSONDatabaseConfig struct {
	// Logger is the logger for the database operations.  It must not be nil.
	Logger *slog.Logger

	// FilePath is the path to the JSON file where leases are stored.  It must
	// not be empty.
	//
	// TODO(e.burkov):  Use [os.Root].
	FilePath string
}

// type check
var _ validate.Interface = (*JSONDatabaseConfig)(nil)

// Validate implements the [validate.Interface] for *JSONDatabaseConfig.
func (c *JSONDatabaseConfig) Validate() (err error) {
	return errors.Join(
		validate.NotNil("c.Logger", c.Logger),
		validate.NotEmpty("c.FilePath", c.FilePath),
	)
}

// JSONDatabase is a [Database] implementation that stores leases in a JSON
// file.
//
// TODO(e.burkov):  Test.
type JSONDatabase struct {
	// mu protects the database file from concurrent access.
	mu       *sync.RWMutex
	logger   *slog.Logger
	filePath string
}

// NewJSONDatabase returns a new [JSONDatabase] instance.  c must be valid.
func NewJSONDatabase(c *JSONDatabaseConfig) (db *JSONDatabase) {
	return &JSONDatabase{
		mu:       &sync.RWMutex{},
		logger:   c.Logger.With("file", c.FilePath),
		filePath: c.FilePath,
	}
}

// Load implements the [Database] interface for *JSONDatabase.
func (db *JSONDatabase) Load(ctx context.Context) (leases []*Lease, err error) {
	defer func() { err = errors.Annotate(err, "loading db: %w") }()

	db.logger.DebugContext(ctx, "loading leases")

	db.mu.RLock()
	defer db.mu.RUnlock()

	file, err := os.Open(db.filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("reading db: %w", err)
		}

		db.logger.DebugContext(ctx, "no db file found")

		return nil, nil
	}
	defer func() { err = errors.WithDeferred(err, file.Close()) }()

	dl := &jsonLeasesData{}
	err = json.NewDecoder(file).Decode(dl)
	if err != nil {
		return nil, fmt.Errorf("decoding db: %w", err)
	}

	var errs []error
	for i, l := range dl.Leases {
		var lease *Lease
		lease, err = l.toInternal()
		if err != nil {
			errs = append(errs, fmt.Errorf("converting lease: at index %d: %w", i, err))

			continue
		}

		leases = append(leases, lease)
	}

	return leases, errors.Join(errs...)
}

// Store implements the [Database] interface for *JSONDatabase.
func (db *JSONDatabase) Store(ctx context.Context, leases []*Lease) (err error) {
	defer func() { err = errors.Annotate(err, "writing db: %w") }()

	dl := &jsonLeasesData{
		// Avoid writing null into the database file if there are no leases.
		Leases:  make([]*jsonLease, 0, len(leases)),
		Version: jsonDataVersion,
	}

	for _, l := range leases {
		lease := toJSONLease(l)
		i, _ := slices.BinarySearchFunc(dl.Leases, lease, (*jsonLease).compareNames)
		dl.Leases = slices.Insert(dl.Leases, i, lease)
	}

	// TODO(e.burkov):  Consider pooling buffers.
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")

	err = enc.Encode(dl)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	err = maybe.WriteFile(db.filePath, buf.Bytes(), jsonDatabasePerm)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	db.logger.InfoContext(ctx, "stored leases", "num", len(dl.Leases))

	return nil
}
