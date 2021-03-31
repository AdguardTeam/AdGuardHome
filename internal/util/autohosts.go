package util

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/log"
	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
)

type onChangedT func()

// AutoHosts - automatic DNS records
type AutoHosts struct {
	// lock protects table and tableReverse.
	lock sync.RWMutex
	// table is the host-to-IPs map.
	table map[string][]net.IP
	// tableReverse is the IP-to-hosts map.
	//
	// TODO(a.garipov): Make better use of newtypes.  Perhaps a custom map.
	tableReverse map[string][]string

	hostsFn   string            // path to the main hosts-file
	hostsDirs []string          // paths to OS-specific directories with hosts-files
	watcher   *fsnotify.Watcher // file and directory watcher object

	// onlyWritesChan used to contain only writing events from watcher.
	onlyWritesChan chan fsnotify.Event

	onChanged onChangedT // notification to other modules
}

// SetOnChanged - set callback function that will be called when the data is changed
func (a *AutoHosts) SetOnChanged(onChanged onChangedT) {
	a.onChanged = onChanged
}

// Notify other modules
func (a *AutoHosts) notify() {
	if a.onChanged == nil {
		return
	}
	a.onChanged()
}

// Init - initialize
// hostsFn: Override default name for the hosts-file (optional)
func (a *AutoHosts) Init(hostsFn string) {
	a.table = make(map[string][]net.IP)
	a.onlyWritesChan = make(chan fsnotify.Event, 2)

	a.hostsFn = "/etc/hosts"
	if runtime.GOOS == "windows" {
		a.hostsFn = os.ExpandEnv("$SystemRoot\\system32\\drivers\\etc\\hosts")
	}
	if len(hostsFn) != 0 {
		a.hostsFn = hostsFn
	}

	if IsOpenWrt() {
		// OpenWrt: "/tmp/hosts/dhcp.cfg01411c".
		a.hostsDirs = append(a.hostsDirs, "/tmp/hosts")
	}

	// Load hosts initially
	a.updateHosts()

	var err error
	a.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Error("autohosts: %s", err)
	}
}

// Start - start module
func (a *AutoHosts) Start() {
	log.Debug("Start AutoHosts module")

	a.updateHosts()

	if a.watcher != nil {
		go a.watcherLoop()

		err := a.watcher.Add(a.hostsFn)
		if err != nil {
			log.Error("Error while initializing watcher for a file %s: %s", a.hostsFn, err)
		}

		for _, dir := range a.hostsDirs {
			err = a.watcher.Add(dir)
			if err != nil {
				log.Error("Error while initializing watcher for a directory %s: %s", dir, err)
			}
		}
	}
}

// Close - close module
func (a *AutoHosts) Close() {
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
	close(a.onlyWritesChan)
}

// Process returns the list of IP addresses for the hostname or nil if nothing
// found.
func (a *AutoHosts) Process(host string, qtype uint16) []net.IP {
	if qtype == dns.TypePTR {
		return nil
	}

	var ipsCopy []net.IP
	a.lock.RLock()
	defer a.lock.RUnlock()

	if ips, ok := a.table[host]; ok {
		ipsCopy = make([]net.IP, len(ips))
		copy(ipsCopy, ips)
	}

	log.Debug("autohosts: answer: %s -> %v", host, ipsCopy)
	return ipsCopy
}

// ProcessReverse processes a PTR request.  It returns nil if nothing is found.
func (a *AutoHosts) ProcessReverse(addr string, qtype uint16) (hosts []string) {
	if qtype != dns.TypePTR {
		return nil
	}

	ipReal := aghnet.UnreverseAddr(addr)
	if ipReal == nil {
		return nil
	}

	ipStr := ipReal.String()

	a.lock.RLock()
	defer a.lock.RUnlock()

	hosts = a.tableReverse[ipStr]

	if len(hosts) == 0 {
		return nil // not found
	}

	log.Debug("autohosts: reverse-lookup: %s -> %s", addr, hosts)

	return hosts
}

// List returns an IP-to-hostnames table.  It is safe for concurrent use.
func (a *AutoHosts) List() (ipToHosts map[string][]string) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	ipToHosts = make(map[string][]string, len(a.tableReverse))
	for k, v := range a.tableReverse {
		ipToHosts[k] = v
	}

	return ipToHosts
}

// update table
func (a *AutoHosts) updateTable(table map[string][]net.IP, host string, ipAddr net.IP) {
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
		log.Debug("autohosts: added %s -> %s", ipAddr, host)
	}
}

// updateTableRev updates the reverse address table.
func (a *AutoHosts) updateTableRev(tableRev map[string][]string, newHost string, ipAddr net.IP) {
	ipStr := ipAddr.String()
	hosts, ok := tableRev[ipStr]
	if !ok {
		tableRev[ipStr] = []string{newHost}
		log.Debug("autohosts: added reverse-address %s -> %s", ipStr, newHost)

		return
	}

	for _, host := range hosts {
		if host == newHost {
			return
		}
	}

	tableRev[ipStr] = append(tableRev[ipStr], newHost)
	log.Debug("autohosts: added reverse-address %s -> %s", ipStr, newHost)
}

// Read IP-hostname pairs from file
// Multiple hostnames per line (per one IP) is supported.
func (a *AutoHosts) load(table map[string][]net.IP, tableRev map[string][]string, fn string) {
	f, err := os.Open(fn)
	if err != nil {
		log.Error("autohosts: %s", err)
		return
	}
	defer f.Close()
	r := bufio.NewReader(f)
	log.Debug("autohosts: loading hosts from file %s", fn)

	for done := false; !done; {
		var line string
		line, err = r.ReadString('\n')
		if err == io.EOF {
			done = true
		} else if err != nil {
			log.Error("autohosts: %s", err)

			return
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}

		for i := 1; i != len(fields); i++ {
			host := fields[i]
			if len(host) == 0 {
				break
			}

			sharp := strings.IndexByte(host, '#')
			if sharp == 0 {
				// Skip the comments.
				break
			} else if sharp > 0 {
				host = host[:sharp]
			}

			a.updateTable(table, host, ip)
			a.updateTableRev(tableRev, host, ip)
			if sharp >= 0 {
				// Skip the comments again.
				break
			}
		}
	}
}

// onlyWrites is a filter for (*fsnotify.Watcher).Events.
func (a *AutoHosts) onlyWrites() {
	for event := range a.watcher.Events {
		if event.Op&fsnotify.Write == fsnotify.Write {
			a.onlyWritesChan <- event
		}
	}
}

// Receive notifications from fsnotify package
func (a *AutoHosts) watcherLoop() {
	go a.onlyWrites()
	for {
		select {
		case event, ok := <-a.onlyWritesChan:
			if !ok {
				return
			}

			// Assume that we sometimes have the same event occurred
			// several times.
			repeat := true
			for repeat {
				select {
				case _, ok = <-a.onlyWritesChan:
					repeat = ok
				default:
					repeat = false
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Debug("autohosts: modified: %s", event.Name)
				a.updateHosts()
			}

		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			log.Error("autohosts: %s", err)
		}
	}
}

// updateHosts - loads system hosts
func (a *AutoHosts) updateHosts() {
	table := make(map[string][]net.IP)
	tableRev := make(map[string][]string)

	a.load(table, tableRev, a.hostsFn)

	for _, dir := range a.hostsDirs {
		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Error("autohosts: Opening directory: %q: %s", dir, err)
			}

			continue
		}

		for _, fi := range fis {
			a.load(table, tableRev, filepath.Join(dir, fi.Name()))
		}
	}

	func() {
		a.lock.Lock()
		defer a.lock.Unlock()

		a.table = table
		a.tableReverse = tableRev
	}()

	a.notify()
}
