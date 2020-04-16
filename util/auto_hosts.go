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

	"github.com/AdguardTeam/golibs/log"
	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
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

// Close - close module
func (a *AutoHosts) Close() {
	a.updateChan <- false
	close(a.updateChan)
	_ = a.watcher.Close()
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

		ip := SplitNext(&line, ' ')
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			continue
		}
		for {
			host := SplitNext(&line, ' ')
			if len(host) == 0 {
				break
			}
			a.updateTable(table, host, ipAddr)
			a.updateTableRev(tableRev, host, ipAddr)
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

// convert character to hex number
func charToHex(n byte) int8 {
	if n >= '0' && n <= '9' {
		return int8(n) - '0'
	} else if (n|0x20) >= 'a' && (n|0x20) <= 'f' {
		return (int8(n) | 0x20) - 'a' + 10
	}
	return -1
}

// parse IPv6 reverse address
func ipParseArpa6(s string) net.IP {
	if len(s) != 63 {
		return nil
	}
	ip6 := make(net.IP, 16)

	for i := 0; i != 64; i += 4 {

		// parse "0.1."
		n := charToHex(s[i])
		n2 := charToHex(s[i+2])
		if s[i+1] != '.' || (i != 60 && s[i+3] != '.') ||
			n < 0 || n2 < 0 {
			return nil
		}

		ip6[16-i/4-1] = byte(n2<<4) | byte(n&0x0f)
	}
	return ip6
}

// ipReverse - reverse IP address: 1.0.0.127 -> 127.0.0.1
func ipReverse(ip net.IP) net.IP {
	n := len(ip)
	r := make(net.IP, n)
	for i := 0; i != n; i++ {
		r[i] = ip[n-i-1]
	}
	return r
}

// Convert reversed ARPA address to a normal IP address
func dnsUnreverseAddr(s string) net.IP {
	const arpaV4 = ".in-addr.arpa"
	const arpaV6 = ".ip6.arpa"

	if strings.HasSuffix(s, arpaV4) {
		ip := strings.TrimSuffix(s, arpaV4)
		ip4 := net.ParseIP(ip).To4()
		if ip4 == nil {
			return nil
		}

		return ipReverse(ip4)

	} else if strings.HasSuffix(s, arpaV6) {
		ip := strings.TrimSuffix(s, arpaV6)
		return ipParseArpa6(ip)
	}

	return nil // unknown suffix
}

// ProcessReverse - process PTR request
// Return "" if not found or an error occurred
func (a *AutoHosts) ProcessReverse(addr string, qtype uint16) string {
	if qtype != dns.TypePTR {
		return ""
	}

	ipReal := dnsUnreverseAddr(addr)
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

// List - get the hosts table.  Thread-safe.
func (a *AutoHosts) List() map[string][]net.IP {
	table := make(map[string][]net.IP)
	a.lock.Lock()
	for k, v := range a.table {
		table[k] = v
	}
	a.lock.Unlock()
	return table
}
