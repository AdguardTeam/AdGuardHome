package rulelist

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghrenameio"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/c2h5oh/datasize"
)

// Filter contains information about a single rule-list filter.
//
// TODO(a.garipov): Use.
type Filter struct {
	// url is the URL of this rule list.  Supported schemes are:
	//   - http
	//   - https
	//   - file
	url *url.URL

	// ruleList is the last successfully compiled [filterlist.RuleList].
	ruleList filterlist.RuleList

	// updated is the time of the last successful update.
	updated time.Time

	// name is the human-readable name of this rule-list filter.
	name string

	// uid is the unique ID of this rule-list filter.
	uid UID

	// urlFilterID is used for working with package urlfilter.
	urlFilterID URLFilterID

	// rulesCount contains the number of rules in this rule-list filter.
	rulesCount int

	// checksum is a CRC32 hash used to quickly check if the rules within a list
	// file have changed.
	checksum uint32

	// enabled, if true, means that this rule-list filter is used for filtering.
	enabled bool
}

// FilterConfig contains the configuration for a [Filter].
type FilterConfig struct {
	// URL is the URL of this rule-list filter.  Supported schemes are:
	//   - http
	//   - https
	//   - file
	URL *url.URL

	// Name is the human-readable name of this rule-list filter.  If not set, it
	// is either taken from the rule-list data or generated synthetically from
	// the UID.
	Name string

	// UID is the unique ID of this rule-list filter.
	UID UID

	// URLFilterID is used for working with package urlfilter.
	URLFilterID URLFilterID

	// Enabled, if true, means that this rule-list filter is used for filtering.
	Enabled bool
}

// NewFilter creates a new rule-list filter.  The filter is not refreshed, so a
// refresh should be performed before use.
func NewFilter(c *FilterConfig) (f *Filter, err error) {
	if c.URL == nil {
		return nil, errors.Error("no url")
	}

	switch s := c.URL.Scheme; s {
	case "http", "https", "file":
		// Go on.
	default:
		return nil, fmt.Errorf("bad url scheme: %q", s)
	}

	return &Filter{
		url:         c.URL,
		name:        c.Name,
		uid:         c.UID,
		urlFilterID: c.URLFilterID,
		enabled:     c.Enabled,
	}, nil
}

// Refresh updates the data in the rule-list filter.  parseBuf is the initial
// buffer used to parse information from the data.  cli and maxSize are only
// used when f is a URL-based list.
//
// TODO(a.garipov): Unexport and test in an internal test or through engine
// tests.
//
// TODO(a.garipov): Consider not returning parseRes.
func (f *Filter) Refresh(
	ctx context.Context,
	parseBuf []byte,
	cli *http.Client,
	cacheDir string,
	maxSize datasize.ByteSize,
) (parseRes *ParseResult, err error) {
	cachePath := filepath.Join(cacheDir, f.uid.String()+".txt")

	switch s := f.url.Scheme; s {
	case "http", "https":
		parseRes, err = f.setFromHTTP(ctx, parseBuf, cli, cachePath, maxSize.Bytes())
	case "file":
		parseRes, err = f.setFromFile(parseBuf, f.url.Path, cachePath)
	default:
		// Since the URL has been prevalidated in New, consider this a
		// programmer error.
		panic(fmt.Errorf("bad url scheme: %q", s))
	}
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	if f.checksum != parseRes.Checksum {
		f.checksum = parseRes.Checksum
		f.rulesCount = parseRes.RulesCount
		f.setName(parseRes.Title)
		f.updated = time.Now()
	}

	return parseRes, nil
}

// setFromHTTP sets the rule-list filter's data from its URL.  It also caches
// the data into a file.
func (f *Filter) setFromHTTP(
	ctx context.Context,
	parseBuf []byte,
	cli *http.Client,
	cachePath string,
	maxSize uint64,
) (parseRes *ParseResult, err error) {
	defer func() { err = errors.Annotate(err, "setting from http: %w") }()

	text, parseRes, err := f.readFromHTTP(ctx, parseBuf, cli, cachePath, maxSize)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	// TODO(a.garipov): Add filterlist.BytesRuleList.
	f.ruleList = &filterlist.StringRuleList{
		ID:             f.urlFilterID,
		RulesText:      text,
		IgnoreCosmetic: true,
	}

	return parseRes, nil
}

// readFromHTTP reads the data from the rule-list filter's URL into the cache
// file as well as returns it as a string.  The data is filtered through a
// parser and so is free from comments, unnecessary whitespace, etc.
func (f *Filter) readFromHTTP(
	ctx context.Context,
	parseBuf []byte,
	cli *http.Client,
	cachePath string,
	maxSize uint64,
) (text string, parseRes *ParseResult, err error) {
	urlStr := f.url.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", nil, fmt.Errorf("making request for http url %q: %w", urlStr, err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("requesting from http url: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, resp.Body.Close()) }()

	// TODO(a.garipov): Use [agdhttp.CheckStatus] when it's moved to golibs.
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("got status code %d, want %d", resp.StatusCode, http.StatusOK)
	}

	fltFile, err := aghrenameio.NewPendingFile(cachePath, aghos.DefaultPermFile)
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { err = aghrenameio.WithDeferredCleanup(err, fltFile) }()

	buf := &bytes.Buffer{}
	mw := io.MultiWriter(buf, fltFile)

	parser := NewParser()
	httpBody := ioutil.LimitReader(resp.Body, maxSize)
	parseRes, err = parser.Parse(mw, httpBody, parseBuf)
	if err != nil {
		return "", nil, fmt.Errorf("parsing response from http url %q: %w", urlStr, err)
	}

	return buf.String(), parseRes, nil
}

// setName sets the title using either the already-present name, the given title
// from the rule-list data, or a synthetic name.
func (f *Filter) setName(title string) {
	if f.name != "" {
		return
	}

	if title != "" {
		f.name = title

		return
	}

	f.name = fmt.Sprintf("List %s", f.uid)
}

// setFromFile sets the rule-list filter's data from a file path.  It also
// caches the data into a file.
//
// TODO(a.garipov): Retest on Windows once rule-list updater is committed.  See
// if calling Close is necessary here.
func (f *Filter) setFromFile(
	parseBuf []byte,
	filePath string,
	cachePath string,
) (parseRes *ParseResult, err error) {
	defer func() { err = errors.Annotate(err, "setting from file: %w") }()

	parseRes, err = parseIntoCache(parseBuf, filePath, cachePath)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("closing old rule list: %w", err)
	}

	rl, err := filterlist.NewFileRuleList(f.urlFilterID, cachePath, true)
	if err != nil {
		return nil, fmt.Errorf("opening new rule list: %w", err)
	}

	f.ruleList = rl

	return parseRes, nil
}

// parseIntoCache copies the relevant the data from filePath into cachePath
// while also parsing it.
func parseIntoCache(
	parseBuf []byte,
	filePath string,
	cachePath string,
) (parseRes *ParseResult, err error) {
	tmpFile, err := aghrenameio.NewPendingFile(cachePath, aghos.DefaultPermFile)
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { err = aghrenameio.WithDeferredCleanup(err, tmpFile) }()

	// #nosec G304 -- Assume that cachePath is always cacheDir joined with a
	// uid using [filepath.Join].
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening src file: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	parser := NewParser()
	parseRes, err = parser.Parse(tmpFile, f, parseBuf)
	if err != nil {
		return nil, fmt.Errorf("copying src file: %w", err)
	}

	return parseRes, nil
}

// Close closes the underlying rule list.
func (f *Filter) Close() (err error) {
	if f.ruleList == nil {
		return nil
	}

	return f.ruleList.Close()
}
