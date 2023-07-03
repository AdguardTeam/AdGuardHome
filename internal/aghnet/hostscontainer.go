package aghnet

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"net/netip"
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
	"golang.org/x/exp/maps"
)

// DefaultHostsPaths returns the slice of paths default for the operating system
// to files and directories which are containing the hosts database.  The result
// is intended to be used within fs.FS so the initial slash is omitted.
func DefaultHostsPaths() (paths []string) {
	return defaultHostsPaths()
}

// requestMatcher combines the logic for matching requests and translating the
// appropriate rules.
type requestMatcher struct {
	// stateLock protects all the fields of requestMatcher.
	stateLock *sync.RWMutex

	// rulesStrg stores the rules obtained from the hosts' file.
	rulesStrg *filterlist.RuleStorage
	// engine serves rulesStrg.
	engine *urlfilter.DNSEngine

	// translator maps generated $dnsrewrite rules into hosts-syntax rules.
	//
	// TODO(e.burkov):  Store the filename from which the rule was parsed.
	translator map[string]string
}

// MatchRequest processes the request rewriting hostnames and addresses read
// from the operating system's hosts files.  res is nil for any request having
// not an A/AAAA or PTR type, see man 5 hosts.
//
// It's safe for concurrent use.
func (rm *requestMatcher) MatchRequest(
	req *urlfilter.DNSRequest,
) (res *urlfilter.DNSResult, ok bool) {
	switch req.DNSType {
	case dns.TypeA, dns.TypeAAAA, dns.TypePTR:
		log.Debug(
			"%s: handling %s request for %s",
			hostsContainerPrefix,
			dns.Type(req.DNSType),
			req.Hostname,
		)

		rm.stateLock.RLock()
		defer rm.stateLock.RUnlock()

		return rm.engine.MatchRequest(req)
	default:
		return nil, false
	}
}

// Translate returns the source hosts-syntax rule for the generated dnsrewrite
// rule or an empty string if the last doesn't exist.  The returned rules are in
// a processed format like:
//
//	ip host1 host2 ...
func (rm *requestMatcher) Translate(rule string) (hostRule string) {
	rm.stateLock.RLock()
	defer rm.stateLock.RUnlock()

	return rm.translator[rule]
}

// resetEng updates container's engine and the translation map.
func (rm *requestMatcher) resetEng(rulesStrg *filterlist.RuleStorage, tr map[string]string) {
	rm.stateLock.Lock()
	defer rm.stateLock.Unlock()

	rm.rulesStrg = rulesStrg
	rm.engine = urlfilter.NewDNSEngine(rm.rulesStrg)

	rm.translator = tr
}

// hostsContainerPrefix is a prefix for logging and wrapping errors in
// HostsContainer's methods.
const hostsContainerPrefix = "hosts container"

// HostsContainer stores the relevant hosts database provided by the OS and
// processes both A/AAAA and PTR DNS requests for those.
//
// TODO(e.burkov):  Improve API and move to golibs.
type HostsContainer struct {
	// requestMatcher matches the requests and translates the rules.  It's
	// embedded to implement MatchRequest and Translate for *HostsContainer.
	//
	// TODO(a.garipov, e.burkov): Consider fully merging into HostsContainer.
	requestMatcher

	// done is the channel to sign closing the container.
	done chan struct{}

	// updates is the channel for receiving updated hosts.
	updates chan HostsRecords

	// last is the set of hosts that was cached within last detected change.
	last HostsRecords

	// fsys is the working file system to read hosts files from.
	fsys fs.FS

	// watcher tracks the changes in specified files and directories.
	watcher aghos.FSWatcher

	// patterns stores specified paths in the fs.Glob-compatible form.
	patterns []string

	// listID is the identifier for the list of generated rules.
	listID int
}

// HostsRecords is a mapping of an IP address to its hosts data.
type HostsRecords map[netip.Addr]*HostsRecord

// HostsRecord represents a single hosts file record.
type HostsRecord struct {
	Aliases   *stringutil.Set
	Canonical string
}

// equal returns true if all fields of rec are equal to field in other or they
// both are nil.
func (rec *HostsRecord) equal(other *HostsRecord) (ok bool) {
	if rec == nil {
		return other == nil
	} else if other == nil {
		return false
	}

	return rec.Canonical == other.Canonical && rec.Aliases.Equal(other.Aliases)
}

// ErrNoHostsPaths is returned when there are no valid paths to watch passed to
// the HostsContainer.
const ErrNoHostsPaths errors.Error = "no valid paths to hosts files provided"

// NewHostsContainer creates a container of hosts, that watches the paths with
// w.  listID is used as an identifier of the underlying rules list.  paths
// shouldn't be empty and each of paths should locate either a file or a
// directory in fsys.  fsys and w must be non-nil.
func NewHostsContainer(
	listID int,
	fsys fs.FS,
	w aghos.FSWatcher,
	paths ...string,
) (hc *HostsContainer, err error) {
	defer func() { err = errors.Annotate(err, "%s: %w", hostsContainerPrefix) }()

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
		requestMatcher: requestMatcher{
			stateLock: &sync.RWMutex{},
		},
		listID:   listID,
		done:     make(chan struct{}, 1),
		updates:  make(chan HostsRecords, 1),
		fsys:     fsys,
		watcher:  w,
		patterns: patterns,
	}

	log.Debug("%s: starting", hostsContainerPrefix)

	// Load initially.
	if err = hc.refresh(); err != nil {
		return nil, err
	}

	for _, p := range paths {
		if err = w.Add(p); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("adding path: %w", err)
			}

			log.Debug("%s: %s is expected to exist but doesn't", hostsContainerPrefix, p)
		}
	}

	go hc.handleEvents()

	return hc, nil
}

// Close implements the [io.Closer] interface for *HostsContainer.  It closes
// both itself and its [aghos.FSWatcher].  Close must only be called once.
func (hc *HostsContainer) Close() (err error) {
	log.Debug("%s: closing", hostsContainerPrefix)

	err = hc.watcher.Close()
	if err != nil {
		err = fmt.Errorf("closing fs watcher: %w", err)

		// Go on and close the container either way.
	}

	close(hc.done)

	return err
}

// Upd returns the channel into which the updates are sent.
func (hc *HostsContainer) Upd() (updates <-chan HostsRecords) {
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

// handleEvents concurrently handles the file system events.  It closes the
// update channel of HostsContainer when finishes.  It's used to be called
// within a separate goroutine.
func (hc *HostsContainer) handleEvents() {
	defer log.OnPanic(fmt.Sprintf("%s: handling events", hostsContainerPrefix))

	defer close(hc.updates)

	ok, eventsCh := true, hc.watcher.Events()
	for ok {
		select {
		case _, ok = <-eventsCh:
			if !ok {
				log.Debug("%s: watcher closed the events channel", hostsContainerPrefix)

				continue
			}

			if err := hc.refresh(); err != nil {
				log.Error("%s: %s", hostsContainerPrefix, err)
			}
		case _, ok = <-hc.done:
			// Go on.
		}
	}
}

// hostsParser is a helper type to parse rules from the operating system's hosts
// file.  It exists for only a single refreshing session.
type hostsParser struct {
	// rulesBuilder builds the resulting rules list content.
	rulesBuilder *strings.Builder

	// translations maps generated rules into actual hosts file lines.
	translations map[string]string

	// table stores only the unique IP-hostname pairs.  It's also sent to the
	// updates channel afterwards.
	table HostsRecords
}

// newHostsParser creates a new *hostsParser with buffers of size taken from the
// previous parse.
func (hc *HostsContainer) newHostsParser() (hp *hostsParser) {
	return &hostsParser{
		rulesBuilder: &strings.Builder{},
		translations: map[string]string{},
		table:        make(HostsRecords, len(hc.last)),
	}
}

// parseFile is a aghos.FileWalker for parsing the files with hosts syntax.  It
// never signs to stop walking and never returns any additional patterns.
//
// See man hosts(5).
func (hp *hostsParser) parseFile(r io.Reader) (patterns []string, cont bool, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		ip, hosts := hp.parseLine(s.Text())
		if ip == (netip.Addr{}) || len(hosts) == 0 {
			continue
		}

		hp.addRecord(ip, hosts)
	}

	return nil, true, s.Err()
}

// parseLine parses the line having the hosts syntax ignoring invalid ones.
func (hp *hostsParser) parseLine(line string) (ip netip.Addr, hosts []string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return netip.Addr{}, nil
	}

	ip, err := netip.ParseAddr(fields[0])
	if err != nil {
		return netip.Addr{}, nil
	}

	for _, f := range fields[1:] {
		hashIdx := strings.IndexByte(f, '#')
		if hashIdx == 0 {
			// The rest of the fields are a part of the comment so return.
			break
		} else if hashIdx > 0 {
			// Only a part of the field is a comment.
			f = f[:hashIdx]
		}

		// Make sure that invalid hosts aren't turned into rules.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/3946.
		//
		// TODO(e.burkov):  Investigate if hosts may contain DNS-SD domains.
		err = netutil.ValidateHostname(f)
		if err != nil {
			log.Error("%s: host %q is invalid, ignoring", hostsContainerPrefix, f)

			continue
		}

		hosts = append(hosts, f)
	}

	return ip, hosts
}

// addRecord puts the record for the IP address to the rules builder if needed.
// The first host is considered to be the canonical name for the IP address.
// hosts must have at least one name.
func (hp *hostsParser) addRecord(ip netip.Addr, hosts []string) {
	line := strings.Join(append([]string{ip.String()}, hosts...), " ")

	rec, ok := hp.table[ip]
	if !ok {
		rec = &HostsRecord{
			Aliases: stringutil.NewSet(),
		}

		rec.Canonical, hosts = hosts[0], hosts[1:]
		hp.addRules(ip, rec.Canonical, line)
		hp.table[ip] = rec
	}

	for _, host := range hosts {
		if rec.Canonical == host || rec.Aliases.Has(host) {
			continue
		}

		rec.Aliases.Add(host)

		hp.addRules(ip, host, line)
	}
}

// addRules adds rules and rule translations for the line.
func (hp *hostsParser) addRules(ip netip.Addr, host, line string) {
	rule, rulePtr := hp.writeRules(host, ip)
	hp.translations[rule], hp.translations[rulePtr] = line, line

	log.Debug("%s: added ip-host pair %q-%q", hostsContainerPrefix, ip, host)
}

// writeRules writes the actual rule for the qtype and the PTR for the host-ip
// pair into internal builders.
func (hp *hostsParser) writeRules(host string, ip netip.Addr) (rule, rulePtr string) {
	// TODO(a.garipov):  Add a netip.Addr version to netutil.
	arpa, err := netutil.IPToReversedAddr(ip.AsSlice())
	if err != nil {
		return "", ""
	}

	const (
		nl = "\n"

		rwSuccess    = "^$dnsrewrite=NOERROR;"
		rwSuccessPTR = "^$dnsrewrite=NOERROR;PTR;"

		modLen    = len(rules.MaskPipe) + len(rwSuccess) + len(";")
		modLenPTR = len(rules.MaskPipe) + len(rwSuccessPTR)
	)

	var qtype string
	// The validation of the IP address has been performed earlier so it is
	// guaranteed to be either an IPv4 or an IPv6.
	if ip.Is4() {
		qtype = "A"
	} else {
		qtype = "AAAA"
	}

	ipStr := ip.String()
	fqdn := dns.Fqdn(host)

	ruleBuilder := &strings.Builder{}
	ruleBuilder.Grow(modLen + len(host) + len(qtype) + len(ipStr))
	stringutil.WriteToBuilder(ruleBuilder, rules.MaskPipe, host, rwSuccess, qtype, ";", ipStr)
	rule = ruleBuilder.String()

	ruleBuilder.Reset()

	ruleBuilder.Grow(modLenPTR + len(arpa) + len(fqdn))
	stringutil.WriteToBuilder(ruleBuilder, rules.MaskPipe, arpa, rwSuccessPTR, fqdn)

	rulePtr = ruleBuilder.String()

	hp.rulesBuilder.Grow(len(rule) + len(rulePtr) + 2*len(nl))
	stringutil.WriteToBuilder(hp.rulesBuilder, rule, nl, rulePtr, nl)

	return rule, rulePtr
}

// sendUpd tries to send the parsed data to the ch.
func (hp *hostsParser) sendUpd(ch chan HostsRecords) {
	log.Debug("%s: sending upd", hostsContainerPrefix)

	upd := hp.table
	select {
	case ch <- upd:
		// Updates are delivered.  Go on.
	case <-ch:
		ch <- upd
		log.Debug("%s: replaced the last update", hostsContainerPrefix)
	case ch <- upd:
		// The previous update was just read and the next one pushed.  Go on.
	default:
		log.Error("%s: the updates channel is broken", hostsContainerPrefix)
	}
}

// newStrg creates a new rules storage from parsed data.
func (hp *hostsParser) newStrg(id int) (s *filterlist.RuleStorage, err error) {
	return filterlist.NewRuleStorage([]filterlist.RuleList{&filterlist.StringRuleList{
		ID:             id,
		RulesText:      hp.rulesBuilder.String(),
		IgnoreCosmetic: true,
	}})
}

// refresh gets the data from specified files and propagates the updates if
// needed.
//
// TODO(e.burkov):  Accept a parameter to specify the files to refresh.
func (hc *HostsContainer) refresh() (err error) {
	log.Debug("%s: refreshing", hostsContainerPrefix)

	hp := hc.newHostsParser()
	if _, err = aghos.FileWalker(hp.parseFile).Walk(hc.fsys, hc.patterns...); err != nil {
		return fmt.Errorf("refreshing : %w", err)
	}

	// hc.last is nil on the first refresh, so let that one through.
	if hc.last != nil && maps.EqualFunc(hp.table, hc.last, (*HostsRecord).equal) {
		log.Debug("%s: no changes detected", hostsContainerPrefix)

		return nil
	}
	defer hp.sendUpd(hc.updates)

	hc.last = maps.Clone(hp.table)

	var rulesStrg *filterlist.RuleStorage
	if rulesStrg, err = hp.newStrg(hc.listID); err != nil {
		return fmt.Errorf("initializing rules storage: %w", err)
	}

	hc.resetEng(rulesStrg, hp.translations)

	return nil
}
