package domainhash

import (
	"bufio"
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// FileName is the name of the built blocklist file inside
	// [DomainHashConfig.DataDir].
	FileName = "blocklist.bin"

	// hashRecordSize is the size of the hash prefix of an intermediate run
	// record.
	hashRecordSize = MaxHashBytes

	// recordLenSize is the size of the length prefix of an intermediate run
	// record.
	recordLenSize = 2

	// runRecordHeader is the size of the header of an intermediate run
	// record: the hash prefix and the length of the domain name.
	runRecordHeader = hashRecordSize + recordLenSize

	// recordOverhead is the number of bytes a single domain adds to the
	// in-memory batch and to a run file, on top of the domain name itself.
	recordOverhead = runRecordHeader

	// Intermediate run records cannot be separated with a byte marker,
	// because the hash prefix is binary and may contain any byte value.
	// Instead, every record is a fixed-size header holding the hash prefix
	// and the little-endian length of the domain name, followed by the
	// domain name itself.

	// runBufferSize is the size of the buffered I/O used for intermediate run
	// files.
	runBufferSize = 64 << 10

	// mergeBufferSize is the size of the buffered write used for the final
	// blocklist file.
	mergeBufferSize = 256 << 10

	// maxCollisionSamples is the maximum number of hash collisions kept for
	// reporting.
	maxCollisionSamples = 16
)

var (
	// ErrBadConfig is returned by [NewBuilder] when the configuration is
	// invalid.
	ErrBadConfig = errors.New("bad domain hash config")

	// ErrBuilderClosed is returned by the methods of a [Builder] after it has
	// been closed.
	ErrBuilderClosed = errors.New("domain hash builder is closed")

	// errBadRunFile is returned when an intermediate run file is malformed.
	errBadRunFile = errors.New("malformed run file")
)

// DomainHashConfig is the configuration of the domain-hash blocklist.
type DomainHashConfig struct {
	// Enabled turns the domain-hash blocklist on.
	Enabled bool

	// HashBytes is the number of bytes of a single hash stored in the
	// blocklist.  Zero means [DefaultHashBytes].
	HashBytes int

	// DataDir is the directory that holds the built blocklist file and the
	// intermediate run files.  It must not be empty.
	DataDir string

	// MemBudgetBytes is the maximum number of bytes the builder may buffer in
	// memory before flushing a batch.  Zero means
	// [DefaultMemBudgetBytes].
	MemBudgetBytes int64
}

// batchBytes returns the number of buffered bytes that makes the builder flush
// the current batch into a run file.  The builder never buffers more than
// [defaultBatchBytes].
func (c DomainHashConfig) batchBytes() (n int64) {
	if c.MemBudgetBytes <= 0 {
		return defaultBatchBytes
	}

	return min(c.MemBudgetBytes, defaultBatchBytes)
}

// Stats contains the statistics of a blocklist built by a [Builder].
type Stats struct {
	// Domains is the number of valid domain names accepted by the builder,
	// duplicates included.
	Domains int

	// Skipped is the number of domain names and rules rejected by the
	// builder.
	Skipped int

	// Hashes is the number of unique hashes written to the blocklist file.
	Hashes int

	// Collisions is the number of distinct domain names that map to a hash
	// that another domain name also maps to.
	Collisions int

	// Runs is the number of intermediate run files written.
	Runs int

	// PeakBatchBytes is the peak number of bytes buffered in memory, which
	// gives the peak memory usage of the builder.
	PeakBatchBytes int64
}

// Collision is a pair of distinct domain names that map to the same truncated
// hash.
type Collision struct {
	// Hash is the truncated hash shared by the domains.
	Hash uint64

	// Domains are the colliding domain names.
	Domains [2]string
}

// batchEntry is a domain name and its hash buffered in memory until the batch
// is flushed into a run file.
type batchEntry struct {
	// domain is the normalized domain name.
	domain string

	// hash is the FNV-1a-64 hash of domain, before truncation.
	hash uint64
}

// Builder builds a blocklist file from a stream of domain names and rules.
type Builder struct {
	// cfg is the normalized configuration.
	cfg DomainHashConfig

	// dir is the directory that holds the intermediate run files.
	dir string

	// runFiles are the paths of the intermediate run files.
	runFiles []string

	// batch is the domain names buffered in memory, in the order they are to
	// be written.
	batch []batchEntry

	// batchBytes is the current size of the batch.
	batchBytes int64

	// collisions are the samples of the hash collisions found by the last
	// merge.
	collisions []Collision

	// stats is the statistics of the builder.
	stats Stats

	// closed reports whether the builder has been closed.
	closed bool
}

// NewBuilder creates a new [Builder] that stores its intermediate files in
// cfg.DataDir and writes the final blocklist to FileName inside it.
func NewBuilder(cfg DomainHashConfig) (b *Builder, err error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("%w: empty data directory", ErrBadConfig)
	}

	cfg.HashBytes = ClampHashBytes(cfg.HashBytes)
	if cfg.MemBudgetBytes <= 0 {
		cfg.MemBudgetBytes = DefaultMemBudgetBytes
	}

	if err = os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dir, err := os.MkdirTemp(cfg.DataDir, "domainhash-runs-")
	if err != nil {
		return nil, fmt.Errorf("creating run directory: %w", err)
	}

	return &Builder{cfg: cfg, dir: dir}, nil
}

// Path returns the path to the blocklist file the builder writes to.
func (b *Builder) Path() (p string) {
	return filepath.Join(b.cfg.DataDir, FileName)
}

// AddDomain adds domain to the blocklist.  Invalid domain names are counted in
// [Stats.Skipped] and otherwise ignored.
func (b *Builder) AddDomain(domain string) (err error) {
	_, err = b.addDomain(domain)

	return err
}

// AddDomains adds each of domains to the blocklist.
func (b *Builder) AddDomains(domains []string) (err error) {
	for _, d := range domains {
		if err = b.AddDomain(d); err != nil {
			return err
		}
	}

	return nil
}

// AddLine parses line with [ParseLine] and adds the domain names found in it.
// It returns the number of domains added.  Lines without a supported blocking
// rule are counted in [Stats.Skipped].
func (b *Builder) AddLine(line string) (n int, err error) {
	domains, ok := ParseLine(line)
	if !ok {
		b.stats.Skipped++

		return 0, nil
	}

	for _, d := range domains {
		var added bool
		if added, err = b.addDomain(d); err != nil {
			return n, err
		}

		if added {
			n++
		}
	}

	return n, nil
}

// ParseLine extracts the domain names from a single line of a blocklist in
// hosts or adblock syntax.  ok is false if the line has no supported blocking
// rule.  The following kinds of lines are supported:
//
//	0.0.0.0 example.com
//	127.0.0.1	example.com www.example.com  # comment
//	||example.com^
//
// Adblock allowlist rules, rules with modifiers, regular expressions, and
// cosmetic rules are ignored, as are hosts rules for addresses other than
// unspecified and loopback ones.
func ParseLine(line string) (domains []string, ok bool) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "\uFEFF")
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false
	}

	switch line[0] {
	case '#', '!', '[':
		return nil, false
	}

	if strings.HasPrefix(line, "@@") {
		return nil, false
	}

	if rest, isAdblock := strings.CutPrefix(line, "||"); isAdblock {
		return parseAdblockRule(rest)
	}

	return parseHostsLine(line)
}

// parseHostsLine extracts the domain names from a hosts-rule line.  ok is false
// if line is not a blocking hosts rule.  Valid domain names are normalized,
// while invalid ones are returned as written, so that the caller can count
// them as skipped.
func parseHostsLine(line string) (domains []string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 || !isBlockingAddr(fields[0]) {
		return nil, false
	}

	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "#") {
			break
		}

		var buf [maxDomainLen]byte
		if norm, isValid := normalizeTo(buf[:], f); isValid {
			f = string(norm)
		}

		domains = append(domains, f)
	}

	if len(domains) == 0 {
		return nil, false
	}

	return domains, true
}

// parseAdblockRule extracts the domain name from a plain adblock rule, which
// is a domain name optionally followed by the "^" separator.  ok is false if
// rest is not such a rule.
func parseAdblockRule(rest string) (domains []string, ok bool) {
	if strings.ContainsAny(rest, "$/|*#") {
		return nil, false
	}

	if i := strings.IndexByte(rest, '^'); i >= 0 {
		if i != len(rest)-1 {
			return nil, false
		}

		rest = rest[:i]
	}

	var buf [maxDomainLen]byte

	norm, ok := normalizeTo(buf[:], rest)
	if !ok {
		return nil, false
	}

	return []string{string(norm)}, true
}

// isBlockingAddr reports whether s is an address that hosts files use to block
// domain names, that is, an unspecified or a loopback one.
func isBlockingAddr(s string) (ok bool) {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return false
	}

	return addr.IsUnspecified() || addr.IsLoopback()
}

// addDomain adds domain to the batch, flushing the batch if it has grown to
// the configured memory budget.  added is false if domain is invalid.
func (b *Builder) addDomain(domain string) (added bool, err error) {
	if b.closed {
		return false, ErrBuilderClosed
	}

	var buf [maxDomainLen]byte

	norm, ok := normalizeTo(buf[:], domain)
	if !ok {
		b.stats.Skipped++

		return false, nil
	}

	b.batch = append(b.batch, batchEntry{hash: sum(norm), domain: string(norm)})
	b.batchBytes += int64(len(norm) + recordOverhead)
	if b.batchBytes > b.stats.PeakBatchBytes {
		b.stats.PeakBatchBytes = b.batchBytes
	}

	b.stats.Domains++

	if b.batchBytes >= b.cfg.batchBytes() {
		if err = b.FlushBatch(); err != nil {
			return true, err
		}
	}

	return true, nil
}

// FlushBatch sorts the batch and writes it into a new intermediate run file.
// It is a no-op if the batch is empty.
func (b *Builder) FlushBatch() (err error) {
	if b.closed {
		return ErrBuilderClosed
	}

	if len(b.batch) == 0 {
		return nil
	}

	slices.SortFunc(b.batch, b.compareBatchEntries)

	path := filepath.Join(b.dir, fmt.Sprintf("run-%06d", len(b.runFiles)))
	if err = writeRun(path, b.batch); err != nil {
		return err
	}

	b.runFiles = append(b.runFiles, path)
	b.batch = nil
	b.batchBytes = 0
	b.stats.Runs = len(b.runFiles)

	return nil
}

// compareBatchEntries orders batch entries by their truncated hash, and then
// by their full hash and domain name, which is the order the merge relies on.
func (b *Builder) compareBatchEntries(a, c batchEntry) (res int) {
	mask := truncMask(b.cfg.HashBytes)
	switch at, ct := a.hash&mask, c.hash&mask; {
	case at < ct:
		return -1
	case at > ct:
		return 1
	}

	switch {
	case a.hash < c.hash:
		return -1
	case a.hash > c.hash:
		return 1
	default:
		return strings.Compare(a.domain, c.domain)
	}
}

// writeRun writes the sorted batch into a new run file at path.
func writeRun(path string, batch []batchEntry) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating run file %q: %w", path, err)
	}

	err = writeRunRecords(f, batch)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(path)
	}

	if err != nil {
		return err
	}

	if closeErr != nil {
		return fmt.Errorf("closing run file %q: %w", path, closeErr)
	}

	return nil
}

// writeRunRecords writes the batch to f as intermediate run records.
func writeRunRecords(f *os.File, batch []batchEntry) (err error) {
	w := bufio.NewWriterSize(f, runBufferSize)

	var h [hashRecordSize]byte
	for i := range batch {
		e := &batch[i]
		binary.LittleEndian.PutUint64(h[:], e.hash)

		if _, err = w.Write(h[:]); err != nil {
			return fmt.Errorf("writing run record: %w", err)
		}

		binary.LittleEndian.PutUint16(h[:recordLenSize], uint16(len(e.domain)))
		if _, err = w.Write(h[:recordLenSize]); err != nil {
			return fmt.Errorf("writing run record: %w", err)
		}

		if _, err = w.WriteString(e.domain); err != nil {
			return fmt.Errorf("writing run record: %w", err)
		}
	}

	if err = w.Flush(); err != nil {
		return fmt.Errorf("flushing run file: %w", err)
	}

	return nil
}

// MergeRuns merges the intermediate run files into the blocklist file in a
// single pass and renames it over the previous one.  It flushes the remaining
// batch first.
func (b *Builder) MergeRuns() (err error) {
	if b.closed {
		return ErrBuilderClosed
	}

	if err = b.FlushBatch(); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(b.cfg.DataDir, FileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary blocklist: %w", err)
	}

	tmpPath := tmp.Name()
	finalPath := b.Path()
	renamed := false
	defer func() {
		if renamed {
			return
		}

		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	hashes, collisions, err := b.mergeInto(tmp)
	if err != nil {
		return err
	}

	if err = tmp.Chmod(0o644); err != nil {
		return fmt.Errorf("setting permissions of %q: %w", tmpPath, err)
	}

	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("syncing %q: %w", tmpPath, err)
	}

	if err = tmp.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", tmpPath, err)
	}

	if err = os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("renaming %q to %q: %w", tmpPath, finalPath, err)
	}

	renamed = true

	b.stats.Hashes = hashes
	b.stats.Collisions = collisions

	return nil
}

// mergeInto merges the run files into dst in the order of their truncated
// hashes, deduplicating hashes and recording collisions.
func (b *Builder) mergeInto(dst *os.File) (hashes, collisions int, err error) {
	all := make(runHeap, 0, len(b.runFiles))
	defer func() {
		for _, c := range all {
			_ = c.close()
		}
	}()

	for _, p := range b.runFiles {
		var c *runCursor
		if c, err = openRunCursor(p, b.cfg.HashBytes); err != nil {
			return 0, 0, err
		}

		var ok bool
		if ok, err = c.next(); err != nil {
			_ = c.close()

			return 0, 0, err
		}

		if ok {
			all = append(all, c)
		} else if err = c.close(); err != nil {
			return 0, 0, err
		}
	}

	heap.Init(&all)

	w := bufio.NewWriterSize(dst, mergeBufferSize)

	b.collisions = nil

	var (
		havePrev   bool
		prevTrunc  uint64
		prevHash   uint64
		prevDomain string
		buf        [MaxHashBytes]byte
		hb         = b.cfg.HashBytes
	)

	for all.Len() > 0 {
		c := all[0]
		switch {
		case !havePrev || c.trunc != prevTrunc:
			PutHash(buf[:], c.trunc, hb)
			if _, err = w.Write(buf[:hb]); err != nil {
				return 0, 0, fmt.Errorf("writing blocklist: %w", err)
			}

			hashes++
			havePrev = true
			prevTrunc, prevHash, prevDomain = c.trunc, c.hash, c.domain
		case c.hash != prevHash || c.domain != prevDomain:
			collisions++
			b.collisions = appendCollision(b.collisions, Collision{
				Hash:    c.trunc,
				Domains: [2]string{prevDomain, c.domain},
			})
		}

		var ok bool
		if ok, err = c.next(); err != nil {
			return 0, 0, err
		}

		if ok {
			heap.Fix(&all, 0)
		} else if err = c.close(); err != nil {
			return 0, 0, err
		} else {
			heap.Pop(&all)
		}
	}

	if err = w.Flush(); err != nil {
		return 0, 0, fmt.Errorf("flushing blocklist: %w", err)
	}

	return hashes, collisions, nil
}

// Build flushes the remaining batch, merges the run files into the blocklist
// file, and removes the intermediate files.
func (b *Builder) Build() (stats Stats, err error) {
	if err = b.MergeRuns(); err != nil {
		return Stats{}, err
	}

	if err = b.Close(); err != nil {
		return Stats{}, err
	}

	return b.stats, nil
}

// Close removes the intermediate files.  It is safe to call it more than once.
func (b *Builder) Close() (err error) {
	if b.closed {
		return nil
	}

	b.closed = true
	b.batch = nil
	b.batchBytes = 0

	dir := b.dir
	b.dir = ""

	if dir == "" {
		return nil
	}

	if err = os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing run directory: %w", err)
	}

	return nil
}

// CollisionSamples returns up to maxCollisionSamples of the collisions found by
// the last merge.
func (b *Builder) CollisionSamples() (cs []Collision) {
	return slices.Clone(b.collisions)
}

// appendCollision appends c to cs unless it is already full.
func appendCollision(cs []Collision, c Collision) (res []Collision) {
	if len(cs) >= maxCollisionSamples {
		return cs
	}

	return append(cs, c)
}

// runCursor reads the records of a single intermediate run file.
type runCursor struct {
	// f is the open run file.
	f *os.File

	// r reads the records of f.
	r *bufio.Reader

	// mask is the mask of the truncated hash.
	mask uint64

	// domain is the domain name of the current record.
	domain string

	// hash is the full hash of the current record.
	hash uint64

	// trunc is the truncated hash of the current record.
	trunc uint64
}

// openRunCursor opens the run file at path for reading.
func openRunCursor(path string, hashBytes int) (c *runCursor, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening run file %q: %w", path, err)
	}

	return &runCursor{
		f:    f,
		r:    bufio.NewReaderSize(f, runBufferSize),
		mask: truncMask(ClampHashBytes(hashBytes)),
	}, nil
}

// next reads the next record of c.  ok is false if the run file is exhausted.
func (c *runCursor) next() (ok bool, err error) {
	var hdr [runRecordHeader]byte
	if _, err = io.ReadFull(c.r, hdr[:]); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}

		return false, fmt.Errorf("%w: reading the header: %w", errBadRunFile, err)
	}

	size := int(binary.LittleEndian.Uint16(hdr[hashRecordSize:]))
	if size == 0 || size > maxDomainLen {
		return false, fmt.Errorf("%w: bad domain length %d", errBadRunFile, size)
	}

	var d [maxDomainLen]byte
	if _, err = io.ReadFull(c.r, d[:size]); err != nil {
		return false, fmt.Errorf("%w: reading the domain: %w", errBadRunFile, err)
	}

	c.hash = binary.LittleEndian.Uint64(hdr[:hashRecordSize])
	c.domain = string(d[:size])
	c.trunc = c.hash & c.mask

	return true, nil
}

// close closes the run file.  It is safe to call it more than once.
func (c *runCursor) close() (err error) {
	if c.f == nil {
		return nil
	}

	f := c.f
	c.f = nil

	return f.Close()
}

// runHeap is a min-heap of run cursors ordered by truncated hash, full hash,
// and domain name.
type runHeap []*runCursor

// type check
var _ heap.Interface = (*runHeap)(nil)

// Len implements the [heap.Interface] interface for runHeap.
func (h runHeap) Len() (n int) {
	return len(h)
}

// Less implements the [heap.Interface] interface for runHeap.
func (h runHeap) Less(i, j int) (less bool) {
	a, b := h[i], h[j]
	switch {
	case a.trunc != b.trunc:
		return a.trunc < b.trunc
	case a.hash != b.hash:
		return a.hash < b.hash
	default:
		return a.domain < b.domain
	}
}

// Swap implements the [heap.Interface] interface for runHeap.
func (h runHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push implements the [heap.Interface] interface for runHeap.
func (h *runHeap) Push(x any) {
	*h = append(*h, x.(*runCursor))
}

// Pop implements the [heap.Interface] interface for runHeap.
func (h *runHeap) Pop() (x any) {
	old := *h
	n := len(old)
	x = old[n-1]
	old[n-1] = nil
	*h = old[:n-1]

	return x
}
