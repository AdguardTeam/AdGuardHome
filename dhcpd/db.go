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
	"github.com/krolaw/dhcp4"
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

// Safe version of dhcp4.IPInRange()
func ipInRange(start, stop, ip net.IP) bool {
	if len(start) != len(stop) ||
		len(start) != len(ip) {
		return false
	}
	return dhcp4.IPInRange(start, stop, ip)
}

// Load lease table from DB
func (s *Server) dbLoad() {
	s.leases = nil
	s.IPpool = make(map[[4]byte]net.HardwareAddr)
	dynLeases := []*Lease{}
	staticLeases := []*Lease{}

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

		if obj[i].Expiry != leaseExpireStatic &&
			!ipInRange(s.leaseStart, s.leaseStop, obj[i].IP) {

			log.Tracef("Skipping a lease with IP %v: not within current IP range", obj[i].IP)
			continue
		}

		lease := Lease{
			HWAddr:   obj[i].HWAddr,
			IP:       obj[i].IP,
			Hostname: obj[i].Hostname,
			Expiry:   time.Unix(obj[i].Expiry, 0),
		}

		if obj[i].Expiry == leaseExpireStatic {
			staticLeases = append(staticLeases, &lease)
		} else {
			dynLeases = append(dynLeases, &lease)
		}
	}

	s.leases = normalizeLeases(staticLeases, dynLeases)

	for _, lease := range s.leases {
		s.reserveIP(lease.IP, lease.HWAddr)
	}

	log.Info("DHCP: loaded %d (%d) leases from DB", len(s.leases), numLeases)
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

	for i := range s.leases {
		if s.leases[i].Expiry.Unix() == 0 {
			continue
		}
		lease := leaseJSON{
			HWAddr:   s.leases[i].HWAddr,
			IP:       s.leases[i].IP,
			Hostname: s.leases[i].Hostname,
			Expiry:   s.leases[i].Expiry.Unix(),
		}
		leases = append(leases, lease)
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
