package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"net"
	"path"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/miekg/dns"
)

// DefaultHostsPaths returns the slice of paths default for the operating system
// to files and directories which are containing the hosts database.  The result
// is intended to use within fs.FS so the initial slash is omitted.
func DefaultHostsPaths() (paths []string) {
	return defaultHostsPaths()
}

// hostsContainerPref is a prefix for logging and wrapping errors in
// HostsContainer's methods.
const hostsContainerPref = "hosts container"

// HostsContainer stores the relevant hosts database provided by the OS and
// processes both A/AAAA and PTR DNS requests for those.
type HostsContainer struct {
	// engLock protects rulesStrg and engine.
	engLock *sync.RWMutex

	// rulesStrg stores the rules obtained from the hosts' file.
	rulesStrg *filterlist.RuleStorage
	// engine serves rulesStrg.
	engine *urlfilter.DNSEngine

	// Updates is the channel for receiving updated hosts.  The receivable map's
	// values has a type of slice of strings.
	updates chan *netutil.IPMap

	// fsys is the working file system to read hosts files from.
	fsys fs.FS

	// w tracks the changes in specified files and directories.
	w aghos.FSWatcher
	// patterns stores specified paths in the fs.Glob-compatible form.
	patterns []string
}

// errNoPaths is returned when there are no paths to watch passed to the
// HostsContainer.
const errNoPaths errors.Error = "hosts paths are empty"

// NewHostsContainer creates a container of hosts, that watches the paths with
// w.  paths shouldn't be empty and each of them should locate either a file or
// a directory in fsys.  fsys and w must be non-nil.
func NewHostsContainer(
	fsys fs.FS,
	w aghos.FSWatcher,
	paths ...string,
) (hc *HostsContainer, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", hostsContainerPref) }()

	if len(paths) == 0 {
		return nil, errNoPaths
	}

	patterns, err := pathsToPatterns(fsys, paths)
	if err != nil {
		return nil, err
	}

	hc = &HostsContainer{
		engLock:  &sync.RWMutex{},
		updates:  make(chan *netutil.IPMap, 1),
		fsys:     fsys,
		w:        w,
		patterns: patterns,
	}

	log.Debug("%s: starting", hostsContainerPref)

	// Load initially.
	if err = hc.refresh(); err != nil {
		return nil, err
	}

	for _, p := range paths {
		err = w.Add(p)
		if err == nil {
			continue
		} else if errors.Is(err, fs.ErrNotExist) {
			log.Debug("%s: file %q expected to exist but doesn't", hostsContainerPref, p)

			continue
		}

		return nil, fmt.Errorf("adding path: %w", err)
	}

	go hc.handleEvents()

	return hc, nil
}

// MatchRequest is the request processing method to resolve hostnames and
// addresses from the operating system's hosts files.  Any request not of A/AAAA
// or PTR type will return with an empty result.  It's safe for concurrent use.
func (hc *HostsContainer) MatchRequest(
	req urlfilter.DNSRequest,
) (res urlfilter.DNSResult, ok bool) {
	switch req.DNSType {
	case dns.TypeA, dns.TypeAAAA, dns.TypePTR:
		log.Debug("%s: handling the request", hostsContainerPref)
	default:
		return urlfilter.DNSResult{}, false
	}

	hc.engLock.RLock()
	defer hc.engLock.RUnlock()

	return hc.engine.MatchRequest(req)
}

// Close implements the io.Closer interface for *HostsContainer.
func (hc *HostsContainer) Close() (err error) {
	log.Debug("%s: closing hosts container", hostsContainerPref)

	return errors.Annotate(hc.w.Close(), "%s: closing: %w", hostsContainerPref)
}

// Upd returns the channel into which the updates are sent.
func (hc *HostsContainer) Upd() (updates <-chan *netutil.IPMap) {
	return hc.updates
}

// pathsToPatterns converts paths into patterns compatible with fs.Glob.
func pathsToPatterns(fsys fs.FS, paths []string) (patterns []string, err error) {
	for i, p := range paths {
		var fi fs.FileInfo
		if fi, err = fs.Stat(fsys, p); err != nil {
			return nil, fmt.Errorf("%q at index %d: %w", p, i, err)
		}

		if fi.IsDir() {
			p = path.Join(p, "*")
		}

		patterns = append(patterns, p)
	}

	return patterns, nil
}

// handleEvents concurrently handles the events.  It closes the update channel
// of HostsContainer when finishes.  Used to be called within a goroutine.
func (hc *HostsContainer) handleEvents() {
	defer log.OnPanic(fmt.Sprintf("%s: handling events", hostsContainerPref))

	defer close(hc.updates)

	for range hc.w.Events() {
		if err := hc.refresh(); err != nil {
			log.Error("%s: %s", hostsContainerPref, err)
		}
	}
}

// hostsParser is a helper type to parse rules from the operating system's hosts
// file.
type hostsParser struct {
	// rules builds the resulting rules list content.
	rules *strings.Builder

	// table stores only the unique IP-hostname pairs.  It's also sent to the
	// updates channel afterwards.
	table *netutil.IPMap
}

// parseHostsFile is a aghtest.FileWalker for parsing the files with hosts
// syntax.  It never signs to stop the walking.
//
// See man hosts(5).
func (hp hostsParser) parseHostsFile(
	r io.Reader,
) (patterns []string, cont bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		ip, hosts := hp.parseLine(s.Text())
		if ip == nil {
			continue
		}

		for _, host := range hosts {
			hp.addPair(ip, host)
		}
	}

	return nil, true, s.Err()
}

// parseLine parses the line having the hosts syntax ignoring invalid ones.
func (hp hostsParser) parseLine(line string) (ip net.IP, hosts []string) {
	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, nil
	}

	if ip = net.ParseIP(fields[0]); ip == nil {
		return nil, nil
	}

loop:
	for _, f := range fields[1:] {
		switch hashIdx := strings.IndexByte(f, '#'); hashIdx {
		case 0:
			// The rest of the fields are a part of the comment so skip
			// immediately.
			break loop
		case -1:
			hosts = append(hosts, f)
		default:
			// Only a part of the field is a comment.
			hosts = append(hosts, f[:hashIdx])

			break loop
		}
	}

	return ip, hosts
}

// add returns true if the pair of ip and host wasn't added to the hp before.
func (hp hostsParser) add(ip net.IP, host string) (added bool) {
	v, ok := hp.table.Get(ip)
	hosts, _ := v.([]string)
	if ok && stringutil.InSlice(hosts, host) {
		return false
	}

	hp.table.Set(ip, append(hosts, host))

	return true
}

// addPair puts the pair of ip and host to the rules builder if needed.
func (hp hostsParser) addPair(ip net.IP, host string) {
	arpa, err := netutil.IPToReversedAddr(ip)
	if err != nil {
		return
	}

	if !hp.add(ip, host) {
		return
	}

	qtype := "AAAA"
	if ip.To4() != nil {
		// Assume the validation of the IP address is performed already.
		qtype = "A"
	}

	stringutil.WriteToBuilder(
		hp.rules,
		"||",
		host,
		"^$dnsrewrite=NOERROR;",
		qtype,
		";",
		ip.String(),
		"\n",
		"||",
		arpa,
		"^$dnsrewrite=NOERROR;PTR;",
		dns.Fqdn(host),
		"\n",
	)

	log.Debug("%s: added ip-host pair %q/%q", hostsContainerPref, ip, host)
}

// sendUpd tries to send the parsed data to the ch.
func (hp hostsParser) sendUpd(ch chan *netutil.IPMap) {
	log.Debug("%s: sending upd", hostsContainerPref)
	select {
	case ch <- hp.table:
		// Updates are delivered.  Go on.
	default:
		log.Debug("%s: the buffer is full", hostsContainerPref)
	}
}

// newStrg creates a new rules storage from parsed data.
func (hp hostsParser) newStrg() (s *filterlist.RuleStorage, err error) {
	return filterlist.NewRuleStorage([]filterlist.RuleList{&filterlist.StringRuleList{
		ID:             1,
		RulesText:      hp.rules.String(),
		IgnoreCosmetic: true,
	}})
}

// refresh gets the data from specified files and propagates the updates.
func (hc *HostsContainer) refresh() (err error) {
	log.Debug("%s: refreshing", hostsContainerPref)

	hp := hostsParser{
		rules: &strings.Builder{},
		table: netutil.NewIPMap(0),
	}

	_, err = aghos.FileWalker(hp.parseHostsFile).Walk(hc.fsys, hc.patterns...)
	if err != nil {
		return fmt.Errorf("updating: %w", err)
	}

	defer hp.sendUpd(hc.updates)

	var rulesStrg *filterlist.RuleStorage
	if rulesStrg, err = hp.newStrg(); err != nil {
		return fmt.Errorf("initializing rules storage: %w", err)
	}

	hc.resetEng(rulesStrg)

	return nil
}

func (hc *HostsContainer) resetEng(rulesStrg *filterlist.RuleStorage) {
	hc.engLock.Lock()
	defer hc.engLock.Unlock()

	hc.rulesStrg = rulesStrg
	hc.engine = urlfilter.NewDNSEngine(hc.rulesStrg)
}
