package aghnet

import (
	"fmt"
	"io"
	"io/fs"
	"net/netip"
	"path"
	"sync/atomic"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// DefaultHostsPaths returns the slice of paths default for the operating system
// to files and directories which are containing the hosts database.  The result
// is intended to be used within fs.FS so the initial slash is omitted.
func DefaultHostsPaths() (paths []string) {
	return defaultHostsPaths()
}

// MatchAddr returns the records for the IP address.
func (hc *HostsContainer) MatchAddr(ip netip.Addr) (recs []*hostsfile.Record) {
	cur := hc.current.Load()
	if cur == nil {
		return nil
	}

	return cur.addrs[ip]
}

// MatchName returns the records for the hostname.
func (hc *HostsContainer) MatchName(name string) (recs []*hostsfile.Record) {
	cur := hc.current.Load()
	if cur != nil {
		recs = cur.names[name]
	}

	return recs
}

// hostsContainerPrefix is a prefix for logging and wrapping errors in
// HostsContainer's methods.
const hostsContainerPrefix = "hosts container"

// Hosts is a map of IP addresses to the records, as it primarily stored in the
// [HostsContainer].  It should not be accessed for writing since it may be read
// concurrently, users should clone it before modifying.
//
// The order of records for each address is preserved from original files, but
// the order of the addresses, being a map key, is not.
//
// TODO(e.burkov):  Probably, this should be a sorted slice of records.
type Hosts map[netip.Addr][]*hostsfile.Record

// HostsContainer stores the relevant hosts database provided by the OS and
// processes both A/AAAA and PTR DNS requests for those.
type HostsContainer struct {
	// done is the channel to sign closing the container.
	done chan struct{}

	// updates is the channel for receiving updated hosts.
	updates chan Hosts

	// current is the last set of hosts parsed.
	current atomic.Pointer[hostsIndex]

	// fsys is the working file system to read hosts files from.
	fsys fs.FS

	// watcher tracks the changes in specified files and directories.
	watcher aghos.FSWatcher

	// patterns stores specified paths in the fs.Glob-compatible form.
	patterns []string
}

// ErrNoHostsPaths is returned when there are no valid paths to watch passed to
// the HostsContainer.
const ErrNoHostsPaths errors.Error = "no valid paths to hosts files provided"

// NewHostsContainer creates a container of hosts, that watches the paths with
// w.  listID is used as an identifier of the underlying rules list.  paths
// shouldn't be empty and each of paths should locate either a file or a
// directory in fsys.  fsys and w must be non-nil.
func NewHostsContainer(
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
		done:     make(chan struct{}, 1),
		updates:  make(chan Hosts, 1),
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
func (hc *HostsContainer) Upd() (updates <-chan Hosts) {
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
				log.Error("%s: warning: refreshing: %s", hostsContainerPrefix, err)
			}
		case _, ok = <-hc.done:
			// Go on.
		}
	}
}

// sendUpd tries to send the parsed data to the ch.
func (hc *HostsContainer) sendUpd(recs Hosts) {
	log.Debug("%s: sending upd", hostsContainerPrefix)

	ch := hc.updates
	select {
	case ch <- recs:
		// Updates are delivered.  Go on.
	case <-ch:
		ch <- recs
		log.Debug("%s: replaced the last update", hostsContainerPrefix)
	case ch <- recs:
		// The previous update was just read and the next one pushed.  Go on.
	default:
		log.Error("%s: the updates channel is broken", hostsContainerPrefix)
	}
}

// hostsIndex is a [hostsfile.Set] to enumerate all the records.
type hostsIndex struct {
	// addrs maps IP addresses to the records.
	addrs Hosts

	// names maps hostnames to the records.
	names map[string][]*hostsfile.Record
}

// walk is a file walking function for hostsIndex.
func (idx *hostsIndex) walk(r io.Reader) (patterns []string, cont bool, err error) {
	return nil, true, hostsfile.Parse(idx, r, nil)
}

// type check
var _ hostsfile.Set = (*hostsIndex)(nil)

// Add implements the [hostsfile.Set] interface for *hostsIndex.
func (idx *hostsIndex) Add(rec *hostsfile.Record) {
	idx.addrs[rec.Addr] = append(idx.addrs[rec.Addr], rec)
	for _, name := range rec.Names {
		idx.names[name] = append(idx.names[name], rec)
	}
}

// type check
var _ hostsfile.HandleSet = (*hostsIndex)(nil)

// HandleInvalid implements the [hostsfile.HandleSet] interface for *hostsIndex.
func (idx *hostsIndex) HandleInvalid(src string, _ []byte, err error) {
	lineErr := &hostsfile.LineError{}
	if !errors.As(err, &lineErr) {
		// Must not happen if idx passed to [hostsfile.Parse].
		return
	} else if errors.Is(lineErr, hostsfile.ErrEmptyLine) {
		// Ignore empty lines.
		return
	}

	log.Info("%s: warning: parsing %q: %s", hostsContainerPrefix, src, lineErr)
}

// equalRecs is an equality function for [*hostsfile.Record].
func equalRecs(a, b *hostsfile.Record) (ok bool) {
	return a.Addr == b.Addr && a.Source == b.Source && slices.Equal(a.Names, b.Names)
}

// equalRecSlices is an equality function for slices of [*hostsfile.Record].
func equalRecSlices(a, b []*hostsfile.Record) (ok bool) { return slices.EqualFunc(a, b, equalRecs) }

// Equal returns true if indexes are equal.
func (idx *hostsIndex) Equal(other *hostsIndex) (ok bool) {
	if idx == nil {
		return other == nil
	} else if other == nil {
		return false
	}

	return maps.EqualFunc(idx.addrs, other.addrs, equalRecSlices)
}

// refresh gets the data from specified files and propagates the updates if
// needed.
//
// TODO(e.burkov):  Accept a parameter to specify the files to refresh.
func (hc *HostsContainer) refresh() (err error) {
	log.Debug("%s: refreshing", hostsContainerPrefix)

	var addrLen, nameLen int
	last := hc.current.Load()
	if last != nil {
		addrLen, nameLen = len(last.addrs), len(last.names)
	}
	idx := &hostsIndex{
		addrs: make(Hosts, addrLen),
		names: make(map[string][]*hostsfile.Record, nameLen),
	}

	_, err = aghos.FileWalker(idx.walk).Walk(hc.fsys, hc.patterns...)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	// TODO(e.burkov):  Serialize updates using time.
	if !last.Equal(idx) {
		hc.current.Store(idx)
		hc.sendUpd(idx.addrs)
	}

	return nil
}
