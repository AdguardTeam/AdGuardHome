package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
)

// download and save all translations.
func (c *twoskyClient) download(ctx context.Context, l *slog.Logger) (err error) {
	var numWorker int

	flagSet := flag.NewFlagSet("download", flag.ExitOnError)
	flagSet.Usage = func() {
		usage("download command error")
	}
	flagSet.IntVar(&numWorker, "n", 1, "number of concurrent downloads")

	err = flagSet.Parse(os.Args[2:])
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	if numWorker < 1 {
		usage("count must be positive")
	}

	downloadURI := c.uri.JoinPath("download")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	wg := &sync.WaitGroup{}
	failed := &sync.Map{}
	uriCh := make(chan *url.URL, len(c.langs))

	for range numWorker {
		wg.Add(1)
		go downloadWorker(ctx, l, wg, failed, client, uriCh)
	}

	for _, lang := range c.langs {
		uri := translationURL(downloadURI, defaultBaseFile, c.projectID, lang)

		uriCh <- uri
	}

	close(uriCh)
	wg.Wait()

	printFailedLocales(ctx, l, failed)

	return nil
}

// printFailedLocales prints sorted list of failed downloads, if any.
func printFailedLocales(ctx context.Context, l *slog.Logger, failed *sync.Map) {
	keys := []string{}
	failed.Range(func(k, _ any) bool {
		s, ok := k.(string)
		if !ok {
			panic("unexpected type")
		}

		keys = append(keys, s)

		return true
	})

	if len(keys) == 0 {
		return
	}

	slices.Sort(keys)
	l.InfoContext(ctx, "failed", "locales", keys)
}

// downloadWorker downloads translations by received urls and saves them.
// Where failed is a map for storing failed downloads.
func downloadWorker(
	ctx context.Context,
	l *slog.Logger,
	wg *sync.WaitGroup,
	failed *sync.Map,
	client *http.Client,
	uriCh <-chan *url.URL,
) {
	defer wg.Done()

	for uri := range uriCh {
		q := uri.Query()
		code := q.Get("language")

		err := saveToFile(ctx, l, client, uri, code)
		if err != nil {
			l.ErrorContext(ctx, "download worker", slogutil.KeyError, err)
			failed.Store(code, struct{}{})
		}
	}
}

// saveToFile downloads translation by url and saves it to a file, or returns
// error.
func saveToFile(
	ctx context.Context,
	l *slog.Logger,
	client *http.Client,
	uri *url.URL,
	code string,
) (err error) {
	data, err := getTranslation(ctx, l, client, uri.String())
	if err != nil {
		return fmt.Errorf("getting translation %q: %s", code, err)
	}

	name := filepath.Join(localesDir, code+".json")
	err = os.WriteFile(name, data, 0o664)
	if err != nil {
		return fmt.Errorf("writing file: %s", err)
	}

	fmt.Println(name)

	return nil
}

// getTranslation returns received translation data and error.  If err is not
// nil, data may contain a response from server for inspection.
func getTranslation(
	ctx context.Context,
	l *slog.Logger,
	client *http.Client,
	url string,
) (data []byte, err error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("requesting: %w", err)
	}

	defer slogutil.CloseAndLog(ctx, l, resp.Body, slog.LevelError)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("url: %q; status code: %s", url, http.StatusText(resp.StatusCode))

		// Go on and download the body for inspection.
	}

	limitReader := ioutil.LimitReader(resp.Body, readLimit)

	data, readErr := io.ReadAll(limitReader)

	return data, errors.WithDeferred(err, readErr)
}

// translationURL returns a new url.URL with provided query parameters.
func translationURL(oldURL *url.URL, baseFile, projectID string, lang langCode) (uri *url.URL) {
	uri = &url.URL{}
	*uri = *oldURL

	q := uri.Query()
	q.Set("format", "json")
	q.Set("filename", baseFile)
	q.Set("project", projectID)
	q.Set("language", string(lang))

	uri.RawQuery = q.Encode()

	return uri
}
