package aghnet

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
)

type onChangedT func()

// EtcHostsContainer - automatic DNS records
//
// TODO(e.burkov): Move the logic under interface.  Refactor.  Probably remove
// the resolving logic.
type EtcHostsContainer struct {
	// lock protects table and tableReverse.
	lock sync.RWMutex
	// table is the host-to-IPs map.
	table map[string][]net.IP
	// tableReverse is the IP-to-hosts map.  The type of the values in the
	// map is []string.
	tableReverse *IPMap

	hostsFn   string            // path to the main hosts-file
	hostsDirs []string          // paths to OS-specific directories with hosts-files
	watcher   *fsnotify.Watcher // file and directory watcher object

	// onlyWritesChan used to contain only writing events from watcher.
	onlyWritesChan chan fsnotify.Event

	onChanged onChangedT // notification to other modules
}

// SetOnChanged - set callback function that will be called when the data is changed
func (ehc *EtcHostsContainer) SetOnChanged(onChanged onChangedT) {
	ehc.onChanged = onChanged
}

// Notify other modules
func (ehc *EtcHostsContainer) notify() {
	if ehc.onChanged == nil {
		return
	}
	ehc.onChanged()
}

// Init - initialize
// hostsFn: Override default name for the hosts-file (optional)
func (ehc *EtcHostsContainer) Init(hostsFn string) {
	ehc.table = make(map[string][]net.IP)
	ehc.onlyWritesChan = make(chan fsnotify.Event, 2)

	ehc.hostsFn = "/etc/hosts"
	if runtime.GOOS == "windows" {
		ehc.hostsFn = os.ExpandEnv("$SystemRoot\\system32\\drivers\\etc\\hosts")
	}
	if len(hostsFn) != 0 {
		ehc.hostsFn = hostsFn
	}

	if aghos.IsOpenWrt() {
		// OpenWrt: "/tmp/hosts/dhcp.cfg01411c".
		ehc.hostsDirs = append(ehc.hostsDirs, "/tmp/hosts")
	}

	// Load hosts initially
	ehc.updateHosts()

	var err error
	ehc.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Error("etchosts: %s", err)
	}
}

// Start - start module
func (ehc *EtcHostsContainer) Start() {
	if ehc == nil {
		return
	}

	log.Debug("Start etchostscontainer module")

	ehc.updateHosts()

	if ehc.watcher != nil {
		go ehc.watcherLoop()

		err := ehc.watcher.Add(ehc.hostsFn)
		if err != nil {
			log.Error("Error while initializing watcher for a file %s: %s", ehc.hostsFn, err)
		}

		for _, dir := range ehc.hostsDirs {
			err = ehc.watcher.Add(dir)
			if err != nil {
				log.Error("Error while initializing watcher for a directory %s: %s", dir, err)
			}
		}
	}
}

// Close - close module
func (ehc *EtcHostsContainer) Close() {
	if ehc == nil {
		return
	}

	if ehc.watcher != nil {
		_ = ehc.watcher.Close()
	}

	// Don't close onlyWritesChan here and let onlyWrites close it after
	// watcher.Events is closed to prevent close races.
}

// Process returns the list of IP addresses for the hostname or nil if nothing
// found.
func (ehc *EtcHostsContainer) Process(host string, qtype uint16) []net.IP {
	if qtype == dns.TypePTR {
		return nil
	}

	var ipsCopy []net.IP
	ehc.lock.RLock()
	defer ehc.lock.RUnlock()

	if ips, ok := ehc.table[host]; ok {
		ipsCopy = make([]net.IP, len(ips))
		copy(ipsCopy, ips)
	}

	log.Debug("etchosts: answer: %s -> %v", host, ipsCopy)
	return ipsCopy
}

// ProcessReverse processes a PTR request.  It returns nil if nothing is found.
func (ehc *EtcHostsContainer) ProcessReverse(addr string, qtype uint16) (hosts []string) {
	if qtype != dns.TypePTR {
		return nil
	}

	ip := UnreverseAddr(addr)
	if ip == nil {
		return nil
	}

	ehc.lock.RLock()
	defer ehc.lock.RUnlock()

	v, ok := ehc.tableReverse.Get(ip)
	if !ok {
		return nil
	}

	hosts, ok = v.([]string)
	if !ok {
		log.Error("etchosts: bad type %T in tableReverse for %s", v, ip)

		return nil
	} else if len(hosts) == 0 {
		return nil
	}

	log.Debug("etchosts: reverse-lookup: %s -> %s", addr, hosts)

	return hosts
}

// List returns an IP-to-hostnames table.  The type of the values in the map is
// []string.  It is safe for concurrent use.
func (ehc *EtcHostsContainer) List() (ipToHosts *IPMap) {
	ehc.lock.RLock()
	defer ehc.lock.RUnlock()

	return ehc.tableReverse.ShallowClone()
}

// update table
func (ehc *EtcHostsContainer) updateTable(table map[string][]net.IP, host string, ipAddr net.IP) {
	ips, ok := table[host]
	if ok {
		for _, ip := range ips {
			if ip.Equal(ipAddr) {
				// IP already exists: don't add duplicates
				ok = false
				break
			}
		}
		if !ok {
			ips = append(ips, ipAddr)
			table[host] = ips
		}
	} else {
		table[host] = []net.IP{ipAddr}
		ok = true
	}
	if ok {
		log.Debug("etchosts: added %s -> %s", ipAddr, host)
	}
}

// updateTableRev updates the reverse address table.
func (ehc *EtcHostsContainer) updateTableRev(tableRev *IPMap, newHost string, ip net.IP) {
	v, ok := tableRev.Get(ip)
	if !ok {
		tableRev.Set(ip, []string{newHost})
		log.Debug("etchosts: added reverse-address %s -> %s", ip, newHost)

		return
	}

	hosts, _ := v.([]string)
	for _, host := range hosts {
		if host == newHost {
			return
		}
	}

	hosts = append(hosts, newHost)
	tableRev.Set(ip, hosts)

	log.Debug("etchosts: added reverse-address %s -> %s", ip, newHost)
}

// parseHostsLine parses hosts from the fields.
func parseHostsLine(fields []string) (hosts []string) {
	for _, f := range fields {
		hashIdx := strings.IndexByte(f, '#')
		if hashIdx == 0 {
			// The rest of the fields are a part of the comment.
			// Skip immediately.
			return
		} else if hashIdx > 0 {
			// Only a part of the field is a comment.
			hosts = append(hosts, f[:hashIdx])

			return hosts
		}

		hosts = append(hosts, f)
	}

	return hosts
}

// load reads IP-hostname pairs from the hosts file.  Multiple hostnames per
// line for one IP are supported.
func (ehc *EtcHostsContainer) load(
	table map[string][]net.IP,
	tableRev *IPMap,
	fn string,
) {
	f, err := os.Open(fn)
	if err != nil {
		log.Error("etchosts: %s", err)

		return
	}

	defer func() {
		derr := f.Close()
		if derr != nil {
			log.Error("etchosts: closing file: %s", err)
		}
	}()

	log.Debug("etchosts: loading hosts from file %s", fn)

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}

		hosts := parseHostsLine(fields[1:])
		for _, host := range hosts {
			ehc.updateTable(table, host, ip)
			ehc.updateTableRev(tableRev, host, ip)
		}
	}

	err = s.Err()
	if err != nil {
		log.Error("etchosts: %s", err)
	}
}

// onlyWrites is a filter for (*fsnotify.Watcher).Events.
func (ehc *EtcHostsContainer) onlyWrites() {
	for event := range ehc.watcher.Events {
		if event.Op&fsnotify.Write == fsnotify.Write {
			ehc.onlyWritesChan <- event
		}
	}

	close(ehc.onlyWritesChan)
}

// Receive notifications from fsnotify package
func (ehc *EtcHostsContainer) watcherLoop() {
	go ehc.onlyWrites()
	for {
		select {
		case event, ok := <-ehc.onlyWritesChan:
			if !ok {
				return
			}

			// Assume that we sometimes have the same event occurred
			// several times.
			repeat := true
			for repeat {
				select {
				case _, ok = <-ehc.onlyWritesChan:
					repeat = ok
				default:
					repeat = false
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Debug("etchosts: modified: %s", event.Name)
				ehc.updateHosts()
			}

		case err, ok := <-ehc.watcher.Errors:
			if !ok {
				return
			}
			log.Error("etchosts: %s", err)
		}
	}
}

// updateHosts - loads system hosts
func (ehc *EtcHostsContainer) updateHosts() {
	table := make(map[string][]net.IP)
	tableRev := NewIPMap(0)

	ehc.load(table, tableRev, ehc.hostsFn)

	for _, dir := range ehc.hostsDirs {
		des, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Error("etchosts: Opening directory: %q: %s", dir, err)
			}

			continue
		}

		for _, de := range des {
			ehc.load(table, tableRev, filepath.Join(dir, de.Name()))
		}
	}

	func() {
		ehc.lock.Lock()
		defer ehc.lock.Unlock()

		ehc.table = table
		ehc.tableReverse = tableRev
	}()

	ehc.notify()
}
