package rulelist

import (
	"bufio"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"slices"

	"github.com/AdguardTeam/golibs/errors"
)

// Parser is a filtering-rule parser that collects data, such as the checksum
// and the title, as well as counts rules and removes comments.
type Parser struct {
	title      string
	rulesCount int
	written    int
	checksum   uint32
	titleFound bool
}

// NewParser returns a new filtering-rule parser.
func NewParser() (p *Parser) {
	return &Parser{}
}

// ParseResult contains information about the results of parsing a
// filtering-rule list by [Parser.Parse].
type ParseResult struct {
	// Title is the title contained within the filtering-rule list, if any.
	Title string

	// RulesCount is the number of rules in the list.  It excludes empty lines
	// and comments.
	RulesCount int

	// BytesWritten is the number of bytes written to dst.
	BytesWritten int

	// Checksum is the CRC-32 checksum of the rules content.  That is, excluding
	// empty lines and comments.
	Checksum uint32
}

// Parse parses data from src into dst using buf during parsing.  r is never
// nil.
func (p *Parser) Parse(dst io.Writer, src io.Reader, buf []byte) (r *ParseResult, err error) {
	s := bufio.NewScanner(src)

	// Don't use [DefaultRuleBufSize] as the maximum size, since some
	// filtering-rule lists compressed by e.g. HostlistsCompiler can have very
	// large lines.  The buffer optimization still works for the more common
	// case of reasonably-sized lines.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/6003.
	s.Buffer(buf, bufio.MaxScanTokenSize)

	// Use a one-based index for lines and columns, since these errors end up in
	// the frontend, and users are more familiar with one-based line and column
	// indexes.
	lineNum := 1
	for s.Scan() {
		var n int
		n, err = p.processLine(dst, s.Bytes(), lineNum)
		p.written += n
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return p.result(), err
		}

		lineNum++
	}

	r = p.result()
	err = s.Err()

	return r, errors.Annotate(err, "scanning filter contents: %w")
}

// result returns the current parsing result.
func (p *Parser) result() (r *ParseResult) {
	return &ParseResult{
		Title:        p.title,
		RulesCount:   p.rulesCount,
		BytesWritten: p.written,
		Checksum:     p.checksum,
	}
}

// processLine processes a single line.  It may write to dst, and if it does, n
// is the number of bytes written.
func (p *Parser) processLine(dst io.Writer, line []byte, lineNum int) (n int, err error) {
	trimmed := bytes.TrimSpace(line)
	if p.written == 0 && isHTMLLine(trimmed) {
		return 0, ErrHTML
	}

	badIdx, isRule := 0, false
	if p.titleFound {
		badIdx, isRule = parseLine(trimmed)
	} else {
		badIdx, isRule = p.parseLineTitle(trimmed)
	}
	if badIdx != -1 {
		return 0, fmt.Errorf(
			"line %d: character %d: likely binary character %q",
			lineNum,
			badIdx+bytes.Index(line, trimmed)+1,
			trimmed[badIdx],
		)
	}

	if !isRule {
		return 0, nil
	}

	p.rulesCount++
	p.checksum = crc32.Update(p.checksum, crc32.IEEETable, trimmed)

	// Assume that there is generally enough space in the buffer to add a
	// newline.
	n, err = dst.Write(append(trimmed, '\n'))

	return n, errors.Annotate(err, "writing rule line: %w")
}

// isHTMLLine returns true if line is likely an HTML line.  line is assumed to
// be trimmed of whitespace characters.
func isHTMLLine(line []byte) (isHTML bool) {
	return hasPrefixFold(line, []byte("<html")) || hasPrefixFold(line, []byte("<!doctype"))
}

// hasPrefixFold is a simple, best-effort prefix matcher.  It may return
// incorrect results for some non-ASCII characters.
func hasPrefixFold(b, prefix []byte) (ok bool) {
	l := len(prefix)

	return len(b) >= l && bytes.EqualFold(b[:l], prefix)
}

// parseLine returns true if the parsed line is a filtering rule.  line is
// assumed to be trimmed of whitespace characters.  badIdx is the index of the
// first character that may indicate that this is a binary file, or -1 if none.
//
// A line is considered a rule if it's not empty, not a comment, and contains
// only printable characters.
func parseLine(line []byte) (badIdx int, isRule bool) {
	if len(line) == 0 || line[0] == '#' || line[0] == '!' {
		return -1, false
	}

	badIdx = slices.IndexFunc(line, likelyBinary)

	return badIdx, badIdx == -1
}

// likelyBinary returns true if b is likely to be a byte from a binary file.
func likelyBinary(b byte) (ok bool) {
	return (b < ' ' || b == 0x7f) && b != '\n' && b != '\r' && b != '\t'
}

// parseLineTitle is like [parseLine] but additionally looks for a title.  line
// is assumed to be trimmed of whitespace characters.
func (p *Parser) parseLineTitle(line []byte) (badIdx int, isRule bool) {
	if len(line) == 0 || line[0] == '#' {
		return -1, false
	}

	if line[0] != '!' {
		badIdx = slices.IndexFunc(line, likelyBinary)

		return badIdx, badIdx == -1
	}

	const titlePattern = "! Title: "
	if !bytes.HasPrefix(line, []byte(titlePattern)) {
		return -1, false
	}

	title := bytes.TrimSpace(line[len(titlePattern):])
	if title != nil {
		// Note that title can be a non-nil empty slice.  Consider that normal
		// and just stop looking for other titles.
		p.title = string(title)
		p.titleFound = true
	}

	return -1, false
}
