package dhcpd

import (
	"encoding/json"
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

// leaseJSON is the structure of stored lease.
//
// Deprecated:  Use [Lease].
type leaseJSON struct {
	HWAddr   []byte `json:"mac"`
	IP       []byte `json:"ip"`
	Hostname string `json:"host"`
	Expiry   int64  `json:"exp"`
}

func normalizeIP(ip net.IP) net.IP {
	ip4 := ip.To4()
	if ip4 != nil {
		return ip4
	}

	return ip
}

// migrateDB migrates stored leases if necessary.
func migrateDB(conf *ServerConfig) (err error) {
	defer func() { err = errors.Annotate(err, "migrating db: %w") }()

	oldLeasesPath := filepath.Join(conf.WorkDir, dbFilename)
	dataDirPath := filepath.Join(conf.DataDir, dataFilename)

	// #nosec G304 -- Trust this path, since it's taken from the old file name
	// relative to the working directory and should generally be considered
	// safe.
	file, err := os.Open(oldLeasesPath)
	if errors.Is(err, os.ErrNotExist) {
		// Nothing to migrate.
		return nil
	} else if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	ljs := []leaseJSON{}
	err = json.NewDecoder(file).Decode(&ljs)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = file.Close()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	leases := []*Lease{}

	for _, lj := range ljs {
		lj.IP = normalizeIP(lj.IP)

		ip, ok := netip.AddrFromSlice(lj.IP)
		if !ok {
			log.Info("dhcp: invalid IP: %s", lj.IP)

			continue
		}

		lease := &Lease{
			Expiry:   time.Unix(lj.Expiry, 0),
			Hostname: lj.Hostname,
			HWAddr:   lj.HWAddr,
			IP:       ip,
			IsStatic: lj.Expiry == leaseExpireStatic,
		}

		leases = append(leases, lease)
	}

	err = writeDB(dataDirPath, leases)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return os.Remove(oldLeasesPath)
}
