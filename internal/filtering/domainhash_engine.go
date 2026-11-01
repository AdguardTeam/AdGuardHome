package filtering

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering/domainhash"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// domainHashDirName is the subdirectory of DataDir that holds the built
// domain-hash blocklist and the complex-rule leftovers that still go through
// urlfilter.
const domainHashDirName = "domainhash"

// domainHashComplexPrefix is the filename prefix of the complex-rule leftovers.
const domainHashComplexPrefix = "complex-"

// buildDomainHashIndex streams blockFilters, diverts simple hosts / ||domain^
// rules into a disk-backed hash index, and returns the leftover complex rules
// as a new filter slice for urlfilter.  When domain-hash indexing is disabled,
// it returns the original filters and a nil index.
func (d *DNSFilter) buildDomainHashIndex(
	ctx context.Context,
	blockFilters []Filter,
) (complexFilters []Filter, idx *domainhash.Index, err error) {
	if d == nil || d.conf == nil || !d.conf.DomainHashEnabled {
		return blockFilters, nil, nil
	}

	dataDir := filepath.Join(d.conf.DataDir, domainHashDirName)
	if err = os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("creating domainhash directory: %w", err)
	}

	margin := d.conf.DomainHashMemMarginBytes
	if margin <= 0 {
		margin = domainhash.DefaultMemBudgetBytes
	}

	budget := domainhash.DetectMemBudget(margin)
	hashBytes := d.conf.DomainHashBytes

	b, err := domainhash.NewBuilder(domainhash.DomainHashConfig{
		Enabled:        true,
		HashBytes:      hashBytes,
		DataDir:        dataDir,
		MemBudgetBytes: budget,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating domainhash builder: %w", err)
	}
	defer func() { _ = b.Close() }()

	complexFilters = make([]Filter, 0, len(blockFilters))
	var diverted int

	for _, f := range blockFilters {
		cf, n, splitErr := d.splitFilterForDomainHash(ctx, b, dataDir, f)
		if splitErr != nil {
			return nil, nil, splitErr
		}

		diverted += n
		if cf != nil {
			complexFilters = append(complexFilters, *cf)
		}
	}

	stats, err := b.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("building domainhash index: %w", err)
	}

	idx, err = domainhash.Open(b.Path(), hashBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("opening domainhash index: %w", err)
	}

	d.logger.InfoContext(
		ctx,
		"domainhash index ready",
		"hashes", stats.Hashes,
		"domains", stats.Domains,
		"diverted_lines", diverted,
		"collisions", stats.Collisions,
		"runs", stats.Runs,
		"peak_batch_bytes", stats.PeakBatchBytes,
		"mem_budget_bytes", budget,
		"complex_filters", len(complexFilters),
	)

	if stats.Collisions > 0 {
		d.logger.WarnContext(
			ctx,
			"domainhash collisions (over-block risk)",
			"collisions", stats.Collisions,
			"samples", b.CollisionSamples(),
		)
	}

	return complexFilters, idx, nil
}

// splitFilterForDomainHash writes simple blocking rules from f into b and the
// remaining lines into a complex leftover file.  n is the number of lines
// diverted into the hash builder.  cf is nil when nothing remains for
// urlfilter.
func (d *DNSFilter) splitFilterForDomainHash(
	ctx context.Context,
	b *domainhash.Builder,
	dataDir string,
	f Filter,
) (cf *Filter, n int, err error) {
	r, closer, err := openFilterReader(f)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = closer.Close() }()

	complexPath := filepath.Join(
		dataDir,
		fmt.Sprintf("%s%d.txt", domainHashComplexPrefix, f.ID),
	)
	out, err := os.Create(complexPath)
	if err != nil {
		return nil, 0, fmt.Errorf("creating complex filter %q: %w", complexPath, err)
	}
	defer func() { _ = out.Close() }()

	bw := bufio.NewWriterSize(out, 64<<10)
	br := bufio.NewReaderSize(r, 64<<10)

	var complexLines int
	for {
		line, readErr := br.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			domains, ok := domainhash.ParseLine(trimmed)
			if ok {
				for _, domain := range domains {
					if addErr := b.AddDomain(domain); addErr != nil {
						return nil, n, fmt.Errorf(
							"adding domain from filter %d: %w",
							f.ID,
							addErr,
						)
					}
				}
				n++
			} else if keepComplexLine(trimmed) {
				if _, writeErr := bw.WriteString(line); writeErr != nil {
					return nil, n, fmt.Errorf("writing complex filter: %w", writeErr)
				}
				complexLines++
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, n, fmt.Errorf("reading filter %d: %w", f.ID, readErr)
		}
	}

	if err = bw.Flush(); err != nil {
		return nil, n, fmt.Errorf("flushing complex filter: %w", err)
	}

	if complexLines == 0 {
		_ = os.Remove(complexPath)
		d.logger.DebugContext(ctx, "filter fully diverted to domainhash", "filter_id", f.ID, "lines", n)

		return nil, n, nil
	}

	return &Filter{ID: f.ID, FilePath: complexPath}, n, nil
}

// openFilterReader returns a reader over the filter contents.  closer must be
// closed by the caller.
func openFilterReader(f Filter) (r io.Reader, closer io.Closer, err error) {
	if len(f.Data) != 0 {
		br := bytes.NewReader(f.Data)

		return br, io.NopCloser(br), nil
	}

	if f.FilePath == "" {
		br := bytes.NewReader(nil)

		return br, io.NopCloser(br), nil
	}

	file, err := os.Open(f.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			br := bytes.NewReader(nil)

			return br, io.NopCloser(br), nil
		}

		return nil, nil, fmt.Errorf("opening filter %q: %w", f.FilePath, err)
	}

	return file, file, nil
}

// keepComplexLine reports whether line should stay in the urlfilter engine.
// Empty lines and comments are dropped; everything ParseLine rejected that is
// still a rule (including @@ allow exceptions and $modifier rules) is kept.
func keepComplexLine(line string) (ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	switch line[0] {
	case '#', '!', '[':
		return false
	}

	return true
}

// closeDomainHashIndex closes the current domain-hash index, if any.
func (d *DNSFilter) closeDomainHashIndex(ctx context.Context) {
	if d.domainHashIndex == nil {
		return
	}

	err := d.domainHashIndex.Close()
	if err != nil {
		d.logger.ErrorContext(ctx, "closing domainhash index", slogutil.KeyError, err)
	}

	d.domainHashIndex = nil
}

// logDomainHashDisabled emits a debug line when the feature is off.
func logDomainHashDisabled(ctx context.Context, l *slog.Logger) {
	if l == nil {
		return
	}

	l.DebugContext(ctx, "domainhash index disabled")
}
