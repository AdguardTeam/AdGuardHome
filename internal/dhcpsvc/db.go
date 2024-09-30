package dhcpsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/google/renameio/v2/maybe"
)

// dataVersion is the current version of the stored DHCP leases structure.
const dataVersion = 1

// databasePerm is the permissions for the database file.
const databasePerm fs.FileMode = 0o640

// dataLeases is the structure of the stored DHCP leases.
type dataLeases struct {
	// Leases is the list containing stored DHCP leases.
	Leases []*dbLease `json:"leases"`

	// Version is the current version of the structure.
	Version int `json:"version"`
}

// dbLease is the structure of stored lease.
type dbLease struct {
	Expiry   string     `json:"expires"`
	IP       netip.Addr `json:"ip"`
	Hostname string     `json:"hostname"`
	HWAddr   string     `json:"mac"`
	IsStatic bool       `json:"static"`
}

// compareNames returns the result of comparing the hostnames of dl and other
// lexicographically.
func (dl *dbLease) compareNames(other *dbLease) (res int) {
	return strings.Compare(dl.Hostname, other.Hostname)
}

// toDBLease converts *Lease to *dbLease.
func toDBLease(l *Lease) (dl *dbLease) {
	var expiryStr string
	if !l.IsStatic {
		// The front-end is waiting for RFC 3999 format of the time value.  It
		// also shouldn't got an Expiry field for static leases.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2692.
		expiryStr = l.Expiry.Format(time.RFC3339)
	}

	return &dbLease{
		Expiry:   expiryStr,
		Hostname: l.Hostname,
		HWAddr:   l.HWAddr.String(),
		IP:       l.IP,
		IsStatic: l.IsStatic,
	}
}

// toInternal converts dl to *Lease.
func (dl *dbLease) toInternal() (l *Lease, err error) {
	mac, err := net.ParseMAC(dl.HWAddr)
	if err != nil {
		return nil, fmt.Errorf("parsing hardware address: %w", err)
	}

	expiry := time.Time{}
	if !dl.IsStatic {
		expiry, err = time.Parse(time.RFC3339, dl.Expiry)
		if err != nil {
			return nil, fmt.Errorf("parsing expiry time: %w", err)
		}
	}

	return &Lease{
		Expiry:   expiry,
		IP:       dl.IP,
		Hostname: dl.Hostname,
		HWAddr:   mac,
		IsStatic: dl.IsStatic,
	}, nil
}

// dbLoad loads stored leases.  It must only be called before the service has
// been started.
func (srv *DHCPServer) dbLoad(ctx context.Context) (err error) {
	defer func() { err = errors.Annotate(err, "loading db: %w") }()

	file, err := os.Open(srv.dbFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading db: %w", err)
		}

		srv.logger.DebugContext(ctx, "no db file found")

		return nil
	}
	defer func() {
		err = errors.WithDeferred(err, file.Close())
	}()

	dl := &dataLeases{}
	err = json.NewDecoder(file).Decode(dl)
	if err != nil {
		return fmt.Errorf("decoding db: %w", err)
	}

	srv.resetLeases()
	srv.addDBLeases(ctx, dl.Leases)

	return nil
}

// addDBLeases adds leases to the server.
func (srv *DHCPServer) addDBLeases(ctx context.Context, leases []*dbLease) {
	var v4, v6 uint
	for i, l := range leases {
		lease, err := l.toInternal()
		if err != nil {
			srv.logger.WarnContext(ctx, "converting lease", "idx", i, slogutil.KeyError, err)

			continue
		}

		iface, err := srv.ifaceForAddr(l.IP)
		if err != nil {
			srv.logger.WarnContext(ctx, "searching lease iface", "idx", i, slogutil.KeyError, err)

			continue
		}

		err = srv.leases.add(lease, iface)
		if err != nil {
			srv.logger.WarnContext(ctx, "adding lease", "idx", i, slogutil.KeyError, err)

			continue
		}

		if l.IP.Is4() {
			v4++
		} else {
			v6++
		}
	}

	// TODO(e.burkov):  Group by interface.
	srv.logger.InfoContext(ctx, "loaded leases", "v4", v4, "v6", v6, "total", len(leases))
}

// writeDB writes leases to the database file.  It expects the
// [DHCPServer.leasesMu] to be locked.
func (srv *DHCPServer) dbStore(ctx context.Context) (err error) {
	defer func() { err = errors.Annotate(err, "writing db: %w") }()

	dl := &dataLeases{
		// Avoid writing null into the database file if there are no leases.
		Leases:  make([]*dbLease, 0, srv.leases.len()),
		Version: dataVersion,
	}

	srv.leases.rangeLeases(func(l *Lease) (cont bool) {
		lease := toDBLease(l)
		i, _ := slices.BinarySearchFunc(dl.Leases, lease, (*dbLease).compareNames)
		dl.Leases = slices.Insert(dl.Leases, i, lease)

		return true
	})

	buf, err := json.Marshal(dl)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = maybe.WriteFile(srv.dbFilePath, buf, databasePerm)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	srv.logger.InfoContext(ctx, "stored leases", "num", len(dl.Leases), "file", srv.dbFilePath)

	return nil
}
