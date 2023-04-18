// On-disk database for lease table

package dhcpd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/renameio/maybe"
	"golang.org/x/exp/slices"
)

const (
	// dataFilename contains saved leases.
	dataFilename = "leases.json"

	// dataVersion is the current version of the stored DHCP leases structure.
	dataVersion = 1
)

// dataLeases is the structure of the stored DHCP leases.
type dataLeases struct {
	// Version is the current version of the structure.
	Version int `json:"version"`

	// Leases is the list containing stored DHCP leases.
	Leases []*Lease `json:"leases"`
}

// dbLoad loads stored leases.
func (s *server) dbLoad() (err error) {
	data, err := os.ReadFile(s.conf.dbFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading db: %w", err)
		}

		return nil
	}

	dl := &dataLeases{}
	err = json.Unmarshal(data, dl)
	if err != nil {
		return fmt.Errorf("decoding db: %w", err)
	}

	leases := dl.Leases

	leases4 := []*Lease{}
	leases6 := []*Lease{}

	for _, l := range leases {
		if l.IP.Is4() {
			leases4 = append(leases4, l)
		} else {
			leases6 = append(leases6, l)
		}
	}

	err = s.srv4.ResetLeases(leases4)
	if err != nil {
		return fmt.Errorf("resetting dhcpv4 leases: %w", err)
	}

	if s.srv6 != nil {
		err = s.srv6.ResetLeases(leases6)
		if err != nil {
			return fmt.Errorf("resetting dhcpv6 leases: %w", err)
		}
	}

	log.Info("dhcp: loaded leases v4:%d  v6:%d  total-read:%d from DB",
		len(leases4), len(leases6), len(leases))

	return nil
}

// dbStore stores DHCP leases.
func (s *server) dbStore() (err error) {
	// Use an empty slice here as opposed to nil so that it doesn't write
	// "null" into the database file if leases are empty.
	leases := []*Lease{}

	leases4 := s.srv4.getLeasesRef()
	leases = append(leases, leases4...)

	if s.srv6 != nil {
		leases6 := s.srv6.getLeasesRef()
		leases = append(leases, leases6...)
	}

	return writeDB(s.conf.dbFilePath, leases)
}

// writeDB writes leases to file at path.
func writeDB(path string, leases []*Lease) (err error) {
	defer func() { err = errors.Annotate(err, "writing db: %w") }()

	slices.SortFunc(leases, func(a, b *Lease) bool {
		return a.Hostname < b.Hostname
	})

	dl := &dataLeases{
		Version: dataVersion,
		Leases:  leases,
	}

	buf, err := json.Marshal(dl)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = maybe.WriteFile(path, buf, 0o644)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	log.Info("dhcp: stored %d leases in %q", len(leases), path)

	return nil
}
