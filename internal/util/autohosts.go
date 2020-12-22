package util

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/miekg/dns"

	"github.com/AdguardTeam/golibs/log"
	"github.com/fsnotify/fsnotify"
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

	hostsFn    string            // path to the main hosts-file
	hostsDirs  []string          // paths to OS-specific directories with hosts-files
	watcher    *fsnotify.Watcher // file and directory watcher object
	updateChan chan bool         // signal for 'updateLoop' goroutine

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
	a.updateChan = make(chan bool, 2)

	a.hostsFn = "/etc/hosts"
	if runtime.GOOS == "windows" {
		a.hostsFn = os.ExpandEnv("$SystemRoot\\system32\\drivers\\etc\\hosts")
	}
	if len(hostsFn) != 0 {
		a.hostsFn = hostsFn
	}

	if IsOpenWRT() {
		a.hostsDirs = append(a.hostsDirs, "/tmp/hosts") // OpenWRT: "/tmp/hosts/dhcp.cfg01411c"
	}

	// Load hosts initially
	a.updateHosts()

	var err error
	a.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Error("AutoHosts: %s", err)
	}
}

// Start - start module
func (a *AutoHosts) Start() {
	log.Debug("Start AutoHosts module")

	go a.updateLoop()
	a.updateChan <- true

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
	a.updateChan <- false
	close(a.updateChan)
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
}

// Process returns the list of IP addresses for the hostname or nil if nothing
// found.
func (a *AutoHosts) Process(host string, qtype uint16) []net.IP {
	if qtype == dns.TypePTR {
		return nil
	}

	var ipsCopy []net.IP
	a.lock.RLock()

	if ips, ok := a.table[host]; ok {
		ipsCopy = make([]net.IP, len(ips))
		copy(ipsCopy, ips)
	}

	a.lock.RUnlock()

	log.Debug("AutoHosts: answer: %s -> %v", host, ipsCopy)
	return ipsCopy
}

// ProcessReverse processes a PTR request.  It returns nil if nothing is found.
func (a *AutoHosts) ProcessReverse(addr string, qtype uint16) (hosts []string) {
	if qtype != dns.TypePTR {
		return nil
	}

	ipReal := DNSUnreverseAddr(addr)
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

	log.Debug("AutoHosts: reverse-lookup: %s -> %s", addr, hosts)

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
		log.Debug("AutoHosts: added %s -> %s", ipAddr, host)
	}
}

// updateTableRev updates the reverse address table.
func (a *AutoHosts) updateTableRev(tableRev map[string][]string, newHost string, ipAddr net.IP) {
	ipStr := ipAddr.String()
	hosts, ok := tableRev[ipStr]
	if !ok {
		tableRev[ipStr] = []string{newHost}
		log.Debug("AutoHosts: added reverse-address %s -> %s", ipStr, newHost)

		return
	}

	for _, host := range hosts {
		if host == newHost {
			return
		}
	}

	tableRev[ipStr] = append(tableRev[ipStr], newHost)
	log.Debug("AutoHosts: added reverse-address %s -> %s", ipStr, newHost)
}

// Read IP-hostname pairs from file
// Multiple hostnames per line (per one IP) is supported.
func (a *AutoHosts) load(table map[string][]net.IP, tableRev map[string][]string, fn string) {
	f, err := os.Open(fn)
	if err != nil {
		log.Error("AutoHosts: %s", err)
		return
	}
	defer f.Close()
	r := bufio.NewReader(f)
	log.Debug("AutoHosts: loading hosts from file %s", fn)

	finish := false
	for !finish {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			finish = true
		} else if err != nil {
			log.Error("AutoHosts: %s", err)
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

		ipAddr := net.ParseIP(fields[0])
		if ipAddr == nil {
			continue
		}
		for i := 1; i != len(fields); i++ {
			host := fields[i]
			if len(host) == 0 {
				break
			}
			sharp := strings.IndexByte(host, '#')
			if sharp == 0 {
				break // skip the rest of the line after #
			} else if sharp > 0 {
				host = host[:sharp]
			}

			a.updateTable(table, host, ipAddr)
			a.updateTableRev(tableRev, host, ipAddr)
			if sharp >= 0 {
				break // skip the rest of the line after #
			}
		}
	}
}

// Receive notifications from fsnotify package
func (a *AutoHosts) watcherLoop() {
	for {
		select {
		case event, ok := <-a.watcher.Events:
			if !ok {
				return
			}

			repeat := true
			for repeat {
				select {
				case <-a.watcher.Events:
					// Skip this duplicating event
				default:
					repeat = false
				}
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Debug("AutoHosts: modified: %s", event.Name)
				select {
				case a.updateChan <- true:
					// sent a signal to 'updateLoop' goroutine
				default:
					// queue is full
				}
			}

		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			log.Error("AutoHosts: %s", err)
		}
	}
}

// updateLoop reads static hosts from system files.
func (a *AutoHosts) updateLoop() {
	for ok := range a.updateChan {
		if !ok {
			log.Debug("Finished AutoHosts update loop")
			return
		}

		a.updateHosts()
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
			if !os.IsNotExist(err) {
				log.Error("AutoHosts: Opening directory: %s: %s", dir, err)
			}
			continue
		}

		for _, fi := range fis {
			a.load(table, tableRev, dir+"/"+fi.Name())
		}
	}

	a.lock.Lock()
	a.table = table
	a.tableReverse = tableRev
	a.lock.Unlock()

	a.notify()
}
