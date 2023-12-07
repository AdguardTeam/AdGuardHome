package dhcpd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

const (
	// leaseExpireStatic is used to define the Expiry field for static
	// leases.
	//
	// Deprecated:  Remove it when migration of DHCP leases will be not needed.
	leaseExpireStatic = 1

	// dbFilename contains saved leases.
	//
	// Deprecated:  Use dataFilename.
	dbFilename = "leases.db"
)

// leaseJSON is the structure of stored lease in a legacy database.
//
// Deprecated:  Use [dbLease].
type leaseJSON struct {
	HWAddr   []byte `json:"mac"`
	IP       []byte `json:"ip"`
	Hostname string `json:"host"`
	Expiry   int64  `json:"exp"`
}

// readOldDB reads the old database from the given path.
func readOldDB(path string) (leases []*leaseJSON, err error) {
	// #nosec G304 -- Trust this path, since it's taken from the old file name
	// relative to the working directory and should generally be considered
	// safe.
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// Nothing to migrate.
		return nil, nil
	} else if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}
	defer func() { err = errors.WithDeferred(err, file.Close()) }()

	leases = []*leaseJSON{}
	err = json.NewDecoder(file).Decode(&leases)
	if err != nil {
		return nil, fmt.Errorf("decoding old db: %w", err)
	}

	return leases, nil
}

// migrateDB migrates stored leases if necessary.
func migrateDB(conf *ServerConfig) (err error) {
	defer func() { err = errors.Annotate(err, "migrating db: %w") }()

	oldLeasesPath := filepath.Join(conf.WorkDir, dbFilename)
	dataDirPath := filepath.Join(conf.DataDir, dataFilename)

	oldLeases, err := readOldDB(oldLeasesPath)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	} else if oldLeases == nil {
		// Nothing to migrate.
		return nil
	}

	leases := make([]*dbLease, 0, len(oldLeases))
	for _, l := range oldLeases {
		l.IP = normalizeIP(l.IP)
		ip, ok := netip.AddrFromSlice(l.IP)
		if !ok {
			log.Info("dhcp: invalid IP: %s", l.IP)

			continue
		}

		leases = append(leases, &dbLease{
			Expiry:   time.Unix(l.Expiry, 0).Format(time.RFC3339),
			Hostname: l.Hostname,
			HWAddr:   net.HardwareAddr(l.HWAddr).String(),
			IP:       ip,
			IsStatic: l.Expiry == leaseExpireStatic,
		})
	}

	err = writeDB(dataDirPath, leases)
	if err != nil {
		// Don't wrap the error since an annotation deferred already.
		return err
	}

	return os.Remove(oldLeasesPath)
}

// normalizeIP converts the given IP address to IPv4 if it's IPv4-mapped IPv6,
// or leaves it as is otherwise.
func normalizeIP(ip net.IP) (normalized net.IP) {
	normalized = ip.To4()
	if normalized != nil {
		return normalized
	}

	return ip
}
