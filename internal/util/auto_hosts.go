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
	lock         sync.Mutex          // serialize access to table
	table        map[string][]net.IP // 'hostname -> IP' table
	tableReverse map[string]string   // "IP -> hostname" table for reverse lookup

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

	if IsOpenWrt() {
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

// Process - get the list of IP addresses for the hostname
// Return nil if not found
func (a *AutoHosts) Process(host string, qtype uint16) []net.IP {
	if qtype == dns.TypePTR {
		return nil
	}

	var ipsCopy []net.IP
	a.lock.Lock()
	ips, _ := a.table[host]
	if len(ips) != 0 {
		ipsCopy = make([]net.IP, len(ips))
		copy(ipsCopy, ips)
	}
	a.lock.Unlock()

	log.Debug("AutoHosts: answer: %s -> %v", host, ipsCopy)
	return ipsCopy
}

// ProcessReverse - process PTR request
// Return "" if not found or an error occurred
func (a *AutoHosts) ProcessReverse(addr string, qtype uint16) string {
	if qtype != dns.TypePTR {
		return ""
	}

	ipReal := DNSUnreverseAddr(addr)
	if ipReal == nil {
		return "" // invalid IP in question
	}
	ipStr := ipReal.String()

	a.lock.Lock()
	host := a.tableReverse[ipStr]
	a.lock.Unlock()

	if len(host) == 0 {
		return "" // not found
	}

	log.Debug("AutoHosts: reverse-lookup: %s -> %s", addr, host)
	return host
}

// List - get "IP -> hostname" table.  Thread-safe.
func (a *AutoHosts) List() map[string]string {
	table := make(map[string]string)
	a.lock.Lock()
	for k, v := range a.tableReverse {
		table[k] = v
	}
	a.lock.Unlock()
	return table
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

// update "reverse" table
func (a *AutoHosts) updateTableRev(tableRev map[string]string, host string, ipAddr net.IP) {
	ipStr := ipAddr.String()
	_, ok := tableRev[ipStr]
	if !ok {
		tableRev[ipStr] = host
		log.Debug("AutoHosts: added reverse-address %s -> %s", ipStr, host)
	}
}

// Read IP-hostname pairs from file
// Multiple hostnames per line (per one IP) is supported.
func (a *AutoHosts) load(table map[string][]net.IP, tableRev map[string]string, fn string) {
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

			// skip duplicate events
			repeat := true
			for repeat {
				select {
				case _ = <-a.watcher.Events:
					// skip this event
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

// updateLoop - read static hosts from system files
func (a *AutoHosts) updateLoop() {
	for {
		select {
		case ok := <-a.updateChan:
			if !ok {
				log.Debug("Finished AutoHosts update loop")
				return
			}

			a.updateHosts()
		}
	}
}

// updateHosts - loads system hosts
func (a *AutoHosts) updateHosts() {
	table := make(map[string][]net.IP)
	tableRev := make(map[string]string)

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
