package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/syncutil"
	"github.com/AdguardTeam/golibs/validate"
)

// download and save all translations.
func (c *twoskyClient) download(ctx context.Context, l *slog.Logger) {
	numWorker, err := parseDownloadArgs()
	if err != nil {
		usage(err.Error())
	}

	downloadURI := c.uri.JoinPath("download")

	wg := &sync.WaitGroup{}
	reqCh := make(chan downloadRequest, numWorker)

	dw := &downloadWorker{
		ctx:    ctx,
		l:      l,
		failed: syncutil.NewMap[string, struct{}](),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		reqCh: reqCh,
	}

	for range numWorker {
		wg.Go(dw.run)
	}

	for _, mask := range c.localizableFiles {
		_, file := filepath.Split(mask)

		for _, lang := range c.langs {
			uri := translationURL(downloadURI, file, c.projectID, lang)

			reqCh <- downloadRequest{
				uri:  uri,
				dirs: c.localesDirs,
			}
		}
	}

	close(reqCh)
	wg.Wait()

	printFailedLocales(ctx, l, dw.failed)
}

// parseDownloadArgs parses command-line arguments for the download command.
func parseDownloadArgs() (numWorker int, err error) {
	flagSet := flag.NewFlagSet("download", flag.ExitOnError)
	flagSet.IntVar(&numWorker, "n", 1, "number of concurrent downloads")

	err = flagSet.Parse(os.Args[2:])
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return 0, err
	}

	return numWorker, validate.Positive("count", numWorker)
}

// printFailedLocales prints sorted list of failed downloads, if any.  l and
// failed must not be nil.
func printFailedLocales(
	ctx context.Context,
	l *slog.Logger,
	failed *syncutil.Map[string, struct{}],
) {
	var keys []string
	for k := range failed.Range {
		keys = append(keys, k)
	}

	if len(keys) == 0 {
		return
	}

	slices.Sort(keys)

	l.InfoContext(ctx, "failed", "locales", keys)
}

// downloadWorker is a worker for downloading translations.  It uses URLs
// received from the channel to download translations and save them to files.
// Failures are stored in the failed map.  All fields must not be nil.
type downloadWorker struct {
	ctx    context.Context
	l      *slog.Logger
	failed *syncutil.Map[string, struct{}]
	client *http.Client
	reqCh  <-chan downloadRequest
}

// downloadRequest is a request to download a translation and save it to every
// directory in dirs.  All fields must not be empty.
type downloadRequest struct {
	uri  *url.URL
	dirs []string
}

// run handles the channel of URLs, one by one.  It returns when the channel is
// closed.  It's used to be run in a separate goroutine.
func (w *downloadWorker) run() {
	for req := range w.reqCh {
		q := req.uri.Query()
		code := q.Get("language")

		err := saveToFile(w.ctx, w.l, w.client, req.uri, code, req.dirs)
		if err != nil {
			w.l.ErrorContext(w.ctx, "download worker", slogutil.KeyError, err)
			w.failed.Store(code, struct{}{})
		}
	}
}

// saveToFile downloads translation by url and saves it to a file in every
// directory in localesDirs, or returns error.  The translation is fetched and
// encoded once, so that the copies stay identical.
func saveToFile(
	ctx context.Context,
	l *slog.Logger,
	client *http.Client,
	uri *url.URL,
	code string,
	localesDirs []string,
) (err error) {
	data, err := getTranslation(ctx, l, client, uri.String())
	if err != nil {
		return fmt.Errorf("getting translation %q: %s", code, err)
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	err = enc.Encode(data)
	if err != nil {
		return fmt.Errorf("encoding translation %q: %w", code, err)
	}

	var errs []error
	for _, localesDir := range localesDirs {
		name := filepath.Join(localesDir, code+".json")
		wErr := os.WriteFile(name, buf.Bytes(), 0o664)
		if wErr != nil {
			errs = append(errs, fmt.Errorf("writing %q: %w", name, wErr))

			continue
		}

		fmt.Println(name)
	}

	return errors.Join(errs...)
}

// getTranslation returns received translation data and error.  If err is not
// nil, data may contain a response from server for inspection.  Otherwise, the
// data is guaranteed to be non-empty.
func getTranslation(
	ctx context.Context,
	l *slog.Logger,
	client *http.Client,
	url string,
) (data map[string]any, err error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("requesting: %w", err)
	}

	defer slogutil.CloseAndLog(ctx, l, resp.Body, slog.LevelError)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("url: %q; status code: %s", url, http.StatusText(resp.StatusCode))

		// Go on and download the body for inspection.
	}

	limitReader := ioutil.LimitReader(resp.Body, readLimit.Bytes())

	dec := json.NewDecoder(limitReader)

	decodeErr := dec.Decode(&data)
	if decodeErr != nil {
		return nil, errors.WithDeferred(err, decodeErr)
	}

	return data, validate.NotEmpty("response", len(data))
}

// translationURL returns a new url.URL with provided query parameters.
func translationURL(baseURL *url.URL, baseFile, projectID string, lang langCode) (uri *url.URL) {
	uri = netutil.CloneURL(baseURL)

	q := uri.Query()
	q.Set("format", "json")
	q.Set("filename", baseFile)
	q.Set("project", projectID)
	q.Set("language", string(lang))

	uri.RawQuery = q.Encode()

	return uri
}
