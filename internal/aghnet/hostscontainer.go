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
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// DefaultHostsPaths returns the slice of paths default for the operating system
// to files and directories which are containing the hosts database.  The result
// is intended to be used within fs.FS so the initial slash is omitted.
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

	// done is the channel to sign closing the container.
	done chan struct{}

	// updates is the channel for receiving updated hosts.
	updates chan *netutil.IPMap
	// last is the set of hosts that was cached within last detected change.
	last *netutil.IPMap

	// fsys is the working file system to read hosts files from.
	fsys fs.FS

	// w tracks the changes in specified files and directories.
	w aghos.FSWatcher
	// patterns stores specified paths in the fs.Glob-compatible form.
	patterns []string
}

// ErrNoHostsPaths is returned when there are no valid paths to watch passed to
// the HostsContainer.
const ErrNoHostsPaths errors.Error = "no valid paths to hosts files provided"

// NewHostsContainer creates a container of hosts, that watches the paths with
// w.  paths shouldn't be empty and each of paths should locate either a file or
// a directory in fsys.  fsys and w must be non-nil.
func NewHostsContainer(
	fsys fs.FS,
	w aghos.FSWatcher,
	paths ...string,
) (hc *HostsContainer, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", hostsContainerPref) }()

	if len(paths) == 0 {
		return nil, ErrNoHostsPaths
	}

	var patterns []string
	patterns, err = pathsToPatterns(fsys, paths)
	if err != nil {
		return nil, err
	} else if len(patterns) == 0 {
		return nil, ErrNoHostsPaths
	}

	hc = &HostsContainer{
		engLock:  &sync.RWMutex{},
		done:     make(chan struct{}, 1),
		updates:  make(chan *netutil.IPMap, 1),
		last:     &netutil.IPMap{},
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
		if err = w.Add(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("adding path: %w", err)
			}

			log.Debug("%s: file %q expected to exist but doesn't", hostsContainerPref, p)
		}
	}

	go hc.handleEvents()

	return hc, nil
}

// MatchRequest is the request processing method to resolve hostnames and
// addresses from the operating system's hosts files.  res is nil for any
// request having not an A/AAAA or PTR type.  It's safe for concurrent use.
func (hc *HostsContainer) MatchRequest(
	req urlfilter.DNSRequest,
) (res *urlfilter.DNSResult, ok bool) {
	switch req.DNSType {
	case dns.TypeA, dns.TypeAAAA, dns.TypePTR:
		log.Debug("%s: handling the request", hostsContainerPref)
	default:
		return nil, false
	}

	hc.engLock.RLock()
	defer hc.engLock.RUnlock()

	return hc.engine.MatchRequest(req)
}

// Close implements the io.Closer interface for *HostsContainer.  Close must
// only be called once.  The returned err is always nil.
func (hc *HostsContainer) Close() (err error) {
	log.Debug("%s: closing", hostsContainerPref)

	close(hc.done)

	return nil
}

// Upd returns the channel into which the updates are sent.  The receivable
// map's values are guaranteed to be of type of *stringutil.Set.
func (hc *HostsContainer) Upd() (updates <-chan *netutil.IPMap) {
	return hc.updates
}

// pathsToPatterns converts paths into patterns compatible with fs.Glob.
func pathsToPatterns(fsys fs.FS, paths []string) (patterns []string, err error) {
	for i, p := range paths {
		var fi fs.FileInfo
		fi, err = fs.Stat(fsys, p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			// Don't put a filename here since it's already added by fs.Stat.
			return nil, fmt.Errorf("path at index %d: %w", i, err)
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

	ok, eventsCh := true, hc.w.Events()
	for ok {
		select {
		case _, ok = <-eventsCh:
			if !ok {
				log.Debug("%s: watcher closed the events channel", hostsContainerPref)

				continue
			}

			if err := hc.refresh(); err != nil {
				log.Error("%s: %s", hostsContainerPref, err)
			}
		case _, ok = <-hc.done:
			// Go on.
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

func (hc *HostsContainer) newHostsParser() (hp *hostsParser) {
	return &hostsParser{
		rules: &strings.Builder{},
		table: netutil.NewIPMap(hc.last.Len()),
	}
}

// parseFile is a aghos.FileWalker for parsing the files with hosts syntax.  It
// never signs to stop walking and never returns any additional patterns.
//
// See man hosts(5).
func (hp *hostsParser) parseFile(
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
func (hp *hostsParser) parseLine(line string) (ip net.IP, hosts []string) {
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
func (hp *hostsParser) add(ip net.IP, host string) (added bool) {
	v, ok := hp.table.Get(ip)
	hosts, _ := v.(*stringutil.Set)
	switch {
	case ok && hosts.Has(host):
		return false
	case hosts == nil:
		hosts = stringutil.NewSet(host)
		hp.table.Set(ip, hosts)
	default:
		hosts.Add(host)
	}

	return true
}

// addPair puts the pair of ip and host to the rules builder if needed.
func (hp *hostsParser) addPair(ip net.IP, host string) {
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

	const (
		nl = "\n"
		sc = ";"

		rewriteSuccess    = "$dnsrewrite=NOERROR" + sc
		rewriteSuccessPTR = rewriteSuccess + "PTR" + sc
	)

	ipStr := ip.String()
	fqdn := dns.Fqdn(host)

	for _, ruleData := range [...][]string{{
		// A/AAAA.
		rules.MaskStartURL,
		host,
		rules.MaskSeparator,
		rewriteSuccess,
		qtype,
		sc,
		ipStr,
		nl,
	}, {
		// PTR.
		rules.MaskStartURL,
		arpa,
		rules.MaskSeparator,
		rewriteSuccessPTR,
		fqdn,
		nl,
	}} {
		stringutil.WriteToBuilder(hp.rules, ruleData...)
	}

	log.Debug("%s: added ip-host pair %q/%q", hostsContainerPref, ip, host)
}

// equalSet returns true if the internal hosts table just parsed equals target.
func (hp *hostsParser) equalSet(target *netutil.IPMap) (ok bool) {
	if hp.table.Len() != target.Len() {
		return false
	}

	hp.table.Range(func(ip net.IP, val interface{}) (cont bool) {
		v, hasIP := target.Get(ip)
		// ok is set to true if the target doesn't contain ip or if the
		// appropriate hosts set isn't equal to the checked one, i.e. the maps
		// have at least one disperancy.
		ok = !hasIP || !v.(*stringutil.Set).Equal(val.(*stringutil.Set))

		// Continue only if maps has no discrepancies.
		return !ok
	})

	// Return true if every value from the IP map has no disperancies with the
	// appropriate one from the target.
	return !ok
}

// sendUpd tries to send the parsed data to the ch.
func (hp *hostsParser) sendUpd(ch chan *netutil.IPMap) {
	log.Debug("%s: sending upd", hostsContainerPref)

	upd := hp.table
	select {
	case ch <- upd:
		// Updates are delivered.  Go on.
	case <-ch:
		ch <- upd
		log.Debug("%s: replaced the last update", hostsContainerPref)
	case ch <- upd:
		// The previous update was just read and the next one pushed.  Go on.
	default:
		log.Debug("%s: the channel is broken", hostsContainerPref)
	}
}

// newStrg creates a new rules storage from parsed data.
func (hp *hostsParser) newStrg() (s *filterlist.RuleStorage, err error) {
	return filterlist.NewRuleStorage([]filterlist.RuleList{&filterlist.StringRuleList{
		ID:             -1,
		RulesText:      hp.rules.String(),
		IgnoreCosmetic: true,
	}})
}

// refresh gets the data from specified files and propagates the updates if
// needed.
//
// TODO(e.burkov):  Accept a parameter to specify the files to refresh.
func (hc *HostsContainer) refresh() (err error) {
	log.Debug("%s: refreshing", hostsContainerPref)

	hp := hc.newHostsParser()
	if _, err = aghos.FileWalker(hp.parseFile).Walk(hc.fsys, hc.patterns...); err != nil {
		return fmt.Errorf("refreshing : %w", err)
	}

	if hp.equalSet(hc.last) {
		log.Debug("%s: no updates detected", hostsContainerPref)

		return nil
	}
	defer hp.sendUpd(hc.updates)

	hc.last = hp.table.ShallowClone()

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
