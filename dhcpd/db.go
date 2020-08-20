// On-disk database for lease table

package dhcpd

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/file"
	"github.com/AdguardTeam/golibs/log"
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
func (s *Server) dbLoad() {
	dynLeases := []*Lease{}
	staticLeases := []*Lease{}
	v6StaticLeases := []*Lease{}
	v6DynLeases := []*Lease{}

	data, err := ioutil.ReadFile(s.conf.DBFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("DHCP: can't read file %s: %v", s.conf.DBFilePath, err)
		}
		return
	}

	obj := []leaseJSON{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		log.Error("DHCP: invalid DB: %v", err)
		return
	}

	numLeases := len(obj)
	for i := range obj {
		obj[i].IP = normalizeIP(obj[i].IP)

		if !(len(obj[i].IP) == 4 || len(obj[i].IP) == 16) {
			log.Info("DHCP: invalid IP: %s", obj[i].IP)
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
	s.srv4.ResetLeases(leases4)

	leases6 := normalizeLeases(v6StaticLeases, v6DynLeases)
	if s.srv6 != nil {
		s.srv6.ResetLeases(leases6)
	}

	log.Info("DHCP: loaded leases v4:%d  v6:%d  total-read:%d from DB",
		len(leases4), len(leases6), numLeases)
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
func (s *Server) dbStore() {
	var leases []leaseJSON

	leases4 := s.srv4.GetLeasesRef()
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
		leases6 := s.srv6.GetLeasesRef()
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

	data, err := json.Marshal(leases)
	if err != nil {
		log.Error("json.Marshal: %v", err)
		return
	}

	err = file.SafeWrite(s.conf.DBFilePath, data)
	if err != nil {
		log.Error("DHCP: can't store lease table on disk: %v  filename: %s",
			err, s.conf.DBFilePath)
		return
	}
	log.Info("DHCP: stored %d leases in DB", len(leases))
}
