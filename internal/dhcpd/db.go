// On-disk database for lease table

package dhcpd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/renameio/maybe"
)

const dbFilename = "leases.db"

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

// Load lease table from DB
func (s *Server) dbLoad() (err error) {
	dynLeases := []*Lease{}
	staticLeases := []*Lease{}
	v6StaticLeases := []*Lease{}
	v6DynLeases := []*Lease{}

	data, err := os.ReadFile(s.conf.DBFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading db: %w", err)
		}

		return nil
	}

	obj := []leaseJSON{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return fmt.Errorf("decoding db: %w", err)
	}

	numLeases := len(obj)
	for i := range obj {
		obj[i].IP = normalizeIP(obj[i].IP)

		if !(len(obj[i].IP) == 4 || len(obj[i].IP) == 16) {
			log.Info("dhcp: invalid IP: %s", obj[i].IP)
			continue
		}

		lease := Lease{
			HWAddr:   obj[i].HWAddr,
			IP:       obj[i].IP,
			Hostname: obj[i].Hostname,
			Expiry:   time.Unix(obj[i].Expiry, 0),
		}

		if len(obj[i].IP) == 16 {
			if obj[i].Expiry == leaseExpireStatic {
				v6StaticLeases = append(v6StaticLeases, &lease)
			} else {
				v6DynLeases = append(v6DynLeases, &lease)
			}
		} else {
			if obj[i].Expiry == leaseExpireStatic {
				staticLeases = append(staticLeases, &lease)
			} else {
				dynLeases = append(dynLeases, &lease)
			}
		}
	}

	leases4 := normalizeLeases(staticLeases, dynLeases)
	err = s.srv4.ResetLeases(leases4)
	if err != nil {
		return fmt.Errorf("resetting dhcpv4 leases: %w", err)
	}

	leases6 := normalizeLeases(v6StaticLeases, v6DynLeases)
	if s.srv6 != nil {
		err = s.srv6.ResetLeases(leases6)
		if err != nil {
			return fmt.Errorf("resetting dhcpv6 leases: %w", err)
		}
	}

	log.Info("dhcp: loaded leases v4:%d  v6:%d  total-read:%d from DB",
		len(leases4), len(leases6), numLeases)

	return nil
}

// Skip duplicate leases
// Static leases have a priority over dynamic leases
func normalizeLeases(staticLeases, dynLeases []*Lease) []*Lease {
	leases := []*Lease{}
	index := map[string]int{}

	for i, lease := range staticLeases {
		_, ok := index[lease.HWAddr.String()]
		if ok {
			continue // skip the lease with the same HW address
		}
		index[lease.HWAddr.String()] = i
		leases = append(leases, lease)
	}

	for i, lease := range dynLeases {
		_, ok := index[lease.HWAddr.String()]
		if ok {
			continue // skip the lease with the same HW address
		}
		index[lease.HWAddr.String()] = i
		leases = append(leases, lease)
	}

	return leases
}

// Store lease table in DB
func (s *Server) dbStore() (err error) {
	// Use an empty slice here as opposed to nil so that it doesn't write
	// "null" into the database file if leases are empty.
	leases := []leaseJSON{}

	leases4 := s.srv4.getLeasesRef()
	for _, l := range leases4 {
		if l.Expiry.Unix() == 0 {
			continue
		}

		lease := leaseJSON{
			HWAddr:   l.HWAddr,
			IP:       l.IP,
			Hostname: l.Hostname,
			Expiry:   l.Expiry.Unix(),
		}

		leases = append(leases, lease)
	}

	if s.srv6 != nil {
		leases6 := s.srv6.getLeasesRef()
		for _, l := range leases6 {
			if l.Expiry.Unix() == 0 {
				continue
			}

			lease := leaseJSON{
				HWAddr:   l.HWAddr,
				IP:       l.IP,
				Hostname: l.Hostname,
				Expiry:   l.Expiry.Unix(),
			}

			leases = append(leases, lease)
		}
	}

	var data []byte
	data, err = json.Marshal(leases)
	if err != nil {
		return fmt.Errorf("encoding db: %w", err)
	}

	err = maybe.WriteFile(s.conf.DBFilePath, data, 0o644)
	if err != nil {
		return fmt.Errorf("writing db: %w", err)
	}

	log.Info("dhcp: stored %d leases in db", len(leases))

	return nil
}
