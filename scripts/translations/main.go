// translations downloads translations, uploads translations, prints summary
// for translations, prints unused strings.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const (
	twoskyConfFile   = "./.twosky.json"
	localesDir       = "./client/src/__locales"
	defaultBaseFile  = "en.json"
	defaultProjectID = "home"
	srcDir           = "./client/src"
	twoskyURI        = "https://twosky.int.agrd.dev/api/v1"

	readLimit = 1 * 1024 * 1024
)

// langCode is a language code.
type langCode string

// languages is a map, where key is language code and value is display name.
type languages map[langCode]string

// textlabel is a text label of localization.
type textLabel string

// locales is a map, where key is text label and value is translation.
type locales map[textLabel]string

func main() {
	if len(os.Args) == 1 {
		usage("need a command")
	}

	if os.Args[1] == "help" {
		usage("")
	}

	uriStr := os.Getenv("TWOSKY_URI")
	if uriStr == "" {
		uriStr = twoskyURI
	}

	uri, err := url.Parse(uriStr)
	check(err)

	projectID := os.Getenv("TWOSKY_PROJECT_ID")
	if projectID == "" {
		projectID = defaultProjectID
	}

	conf, err := readTwoskyConf()
	check(err)

	switch os.Args[1] {
	case "summary":
		err = summary(conf.Languages)
	case "download":
		err = download(uri, projectID, conf.Languages)
	case "unused":
		err = unused(conf.LocalizableFiles[0])
	case "upload":
		err = upload(uri, projectID, conf.BaseLangcode)
	case "auto-add":
		err = autoAdd(conf.LocalizableFiles[0])
	default:
		usage("unknown command")
	}

	check(err)
}

// check is a simple error-checking helper for scripts.
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// usage prints usage.  If addStr is not empty print addStr and exit with code
// 1, otherwise exit with code 0.
func usage(addStr string) {
	const usageStr = `Usage: go run main.go <command> [<args>]
Commands:
  help
        Print usage.
  summary
        Print summary.
  download [-n <count>]
        Download translations.  count is a number of concurrent downloads.
  unused
        Print unused strings.
  upload
        Upload translations.
  auto-add
		Add locales with additions to the git and restore locales with
		deletions.`

	if addStr != "" {
		fmt.Printf("%s\n%s\n", addStr, usageStr)

		os.Exit(1)
	}

	fmt.Println(usageStr)

	os.Exit(0)
}

// twoskyConf is the configuration structure for localization.
type twoskyConf struct {
	Languages        languages `json:"languages"`
	ProjectID        string    `json:"project_id"`
	BaseLangcode     langCode  `json:"base_locale"`
	LocalizableFiles []string  `json:"localizable_files"`
}

// readTwoskyConf returns configuration.
func readTwoskyConf() (t twoskyConf, err error) {
	defer func() { err = errors.Annotate(err, "parsing twosky conf: %w") }()

	b, err := os.ReadFile(twoskyConfFile)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return twoskyConf{}, err
	}

	var tsc []twoskyConf
	err = json.Unmarshal(b, &tsc)
	if err != nil {
		err = fmt.Errorf("unmarshalling %q: %w", twoskyConfFile, err)

		return twoskyConf{}, err
	}

	if len(tsc) == 0 {
		err = fmt.Errorf("%q is empty", twoskyConfFile)

		return twoskyConf{}, err
	}

	conf := tsc[0]

	for _, lang := range conf.Languages {
		if lang == "" {
			return twoskyConf{}, errors.Error("language is empty")
		}
	}

	if len(conf.LocalizableFiles) == 0 {
		return twoskyConf{}, errors.Error("no localizable files specified")
	}

	return conf, nil
}

// readLocales reads file with name fn and returns a map, where key is text
// label and value is localization.
func readLocales(fn string) (loc locales, err error) {
	b, err := os.ReadFile(fn)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	loc = make(locales)
	err = json.Unmarshal(b, &loc)
	if err != nil {
		err = fmt.Errorf("unmarshalling %q: %w", fn, err)

		return nil, err
	}

	return loc, nil
}

// summary prints summary for translations.
func summary(langs languages) (err error) {
	basePath := filepath.Join(localesDir, defaultBaseFile)
	baseLoc, err := readLocales(basePath)
	if err != nil {
		return fmt.Errorf("summary: %w", err)
	}

	size := float64(len(baseLoc))

	keys := maps.Keys(langs)
	slices.Sort(keys)

	for _, lang := range keys {
		name := filepath.Join(localesDir, string(lang)+".json")
		if name == basePath {
			continue
		}

		var loc locales
		loc, err = readLocales(name)
		if err != nil {
			return fmt.Errorf("summary: reading locales: %w", err)
		}

		f := float64(len(loc)) * 100 / size

		fmt.Printf("%s\t %6.2f %%\n", lang, f)
	}

	return nil
}

// download and save all translations.  uri is the base URL.  projectID is the
// name of the project.
func download(uri *url.URL, projectID string, langs languages) (err error) {
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

	downloadURI := uri.JoinPath("download")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	wg := &sync.WaitGroup{}
	uriCh := make(chan *url.URL, len(langs))

	for i := 0; i < numWorker; i++ {
		wg.Add(1)
		go downloadWorker(wg, client, uriCh)
	}

	for lang := range langs {
		uri = translationURL(downloadURI, defaultBaseFile, projectID, lang)

		uriCh <- uri
	}

	close(uriCh)
	wg.Wait()

	return nil
}

// downloadWorker downloads translations by received urls and saves them.
func downloadWorker(wg *sync.WaitGroup, client *http.Client, uriCh <-chan *url.URL) {
	defer wg.Done()

	for uri := range uriCh {
		data, err := getTranslation(client, uri.String())
		if err != nil {
			log.Error("download worker: getting translation: %s", err)
			log.Info("download worker: error response:\n%s", data)

			continue
		}

		q := uri.Query()
		code := q.Get("language")

		// Fix some TwoSky weirdnesses.
		//
		// TODO(a.garipov): Remove when those are fixed.
		code = strings.ToLower(code)

		name := filepath.Join(localesDir, code+".json")
		err = os.WriteFile(name, data, 0o664)
		if err != nil {
			log.Error("download worker: writing file: %s", err)

			continue
		}

		fmt.Println(name)
	}
}

// getTranslation returns received translation data and error.  If err is not
// nil, data may contain a response from server for inspection.
func getTranslation(client *http.Client, url string) (data []byte, err error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("requesting: %w", err)
	}

	defer log.OnCloserError(resp.Body, log.ERROR)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("url: %q; status code: %s", url, http.StatusText(resp.StatusCode))

		// Go on and download the body for inspection.
	}

	limitReader, lrErr := aghio.LimitReader(resp.Body, readLimit)
	if lrErr != nil {
		// Generally shouldn't happen, since the only error returned by
		// [aghio.LimitReader] is an argument error.
		panic(fmt.Errorf("limit reading: %w", lrErr))
	}

	data, readErr := io.ReadAll(limitReader)

	return data, errors.WithDeferred(err, readErr)
}

// translationURL returns a new url.URL with provided query parameters.
func translationURL(oldURL *url.URL, baseFile, projectID string, lang langCode) (uri *url.URL) {
	uri = &url.URL{}
	*uri = *oldURL

	// Fix some TwoSky weirdnesses.
	//
	// TODO(a.garipov): Remove when those are fixed.
	switch lang {
	case "si-lk":
		lang = "si-LK"
	case "zh-hk":
		lang = "zh-HK"
	default:
		// Go on.
	}

	q := uri.Query()
	q.Set("format", "json")
	q.Set("filename", baseFile)
	q.Set("project", projectID)
	q.Set("language", string(lang))

	uri.RawQuery = q.Encode()

	return uri
}

// unused prints unused text labels.
func unused(basePath string) (err error) {
	baseLoc, err := readLocales(basePath)
	if err != nil {
		return fmt.Errorf("unused: %w", err)
	}

	locDir := filepath.Clean(localesDir)

	fileNames := []string{}
	err = filepath.Walk(srcDir, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			log.Info("warning: accessing a path %q: %s", name, err)

			return nil
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(name, locDir) {
			return nil
		}

		ext := filepath.Ext(name)
		if ext == ".js" || ext == ".json" {
			fileNames = append(fileNames, name)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("filepath walking %q: %w", srcDir, err)
	}

	return findUnused(fileNames, baseLoc)
}

// findUnused prints unused text labels from fileNames.
func findUnused(fileNames []string, loc locales) (err error) {
	knownUsed := []textLabel{
		"blocking_mode_refused",
		"blocking_mode_nxdomain",
		"blocking_mode_custom_ip",
	}

	for _, v := range knownUsed {
		delete(loc, v)
	}

	for _, fn := range fileNames {
		var buf []byte
		buf, err = os.ReadFile(fn)
		if err != nil {
			return fmt.Errorf("finding unused: %w", err)
		}

		for k := range loc {
			if bytes.Contains(buf, []byte(k)) {
				delete(loc, k)
			}
		}
	}

	keys := maps.Keys(loc)
	slices.Sort(keys)

	for _, v := range keys {
		fmt.Println(v)
	}

	return nil
}

// upload base translation.  uri is the base URL.  projectID is the name of the
// project.  baseLang is the base language code.
func upload(uri *url.URL, projectID string, baseLang langCode) (err error) {
	defer func() { err = errors.Annotate(err, "upload: %w") }()

	uploadURI := uri.JoinPath("upload")

	lang := baseLang

	langStr := os.Getenv("UPLOAD_LANGUAGE")
	if langStr != "" {
		lang = langCode(langStr)
	}

	basePath := filepath.Join(localesDir, defaultBaseFile)

	formData := map[string]string{
		"format":   "json",
		"language": string(lang),
		"filename": defaultBaseFile,
		"project":  projectID,
	}

	buf, cType, err := prepareMultipartMsg(formData, basePath)
	if err != nil {
		return fmt.Errorf("preparing multipart msg: %w", err)
	}

	err = send(uploadURI.String(), cType, buf)
	if err != nil {
		return fmt.Errorf("sending multipart msg: %w", err)
	}

	return nil
}

// prepareMultipartMsg prepares translation data for upload.
func prepareMultipartMsg(
	formData map[string]string,
	basePath string,
) (buf *bytes.Buffer, cType string, err error) {
	buf = &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	var fw io.Writer

	for k, v := range formData {
		err = w.WriteField(k, v)
		if err != nil {
			return nil, "", fmt.Errorf("writing field: %w", err)
		}
	}

	file, err := os.Open(basePath)
	if err != nil {
		return nil, "", fmt.Errorf("opening file: %w", err)
	}

	defer func() {
		err = errors.WithDeferred(err, file.Close())
	}()

	h := make(textproto.MIMEHeader)
	h.Set(httphdr.ContentType, aghhttp.HdrValApplicationJSON)

	d := fmt.Sprintf("form-data; name=%q; filename=%q", "file", defaultBaseFile)
	h.Set(httphdr.ContentDisposition, d)

	fw, err = w.CreatePart(h)
	if err != nil {
		return nil, "", fmt.Errorf("creating part: %w", err)
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		return nil, "", fmt.Errorf("copying: %w", err)
	}

	err = w.Close()
	if err != nil {
		return nil, "", fmt.Errorf("closing writer: %w", err)
	}

	return buf, w.FormDataContentType(), nil
}

// send POST request to uriStr.
func send(uriStr, cType string, buf *bytes.Buffer) (err error) {
	var client http.Client

	req, err := http.NewRequest(http.MethodPost, uriStr, buf)
	if err != nil {
		return fmt.Errorf("bad request: %w", err)
	}

	req.Header.Set(httphdr.ContentType, cType)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("client post form: %w", err)
	}

	defer func() {
		err = errors.WithDeferred(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code is not ok: %q", http.StatusText(resp.StatusCode))
	}

	return nil
}

// autoAdd adds locales with additions to the git and restores locales with
// deletions.
func autoAdd(basePath string) (err error) {
	defer func() { err = errors.Annotate(err, "auto add: %w") }()

	adds, dels, err := changedLocales()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	if slices.Contains(dels, basePath) {
		return errors.Error("base locale contains deletions")
	}

	var (
		args []string
		code int
		out  []byte
	)

	if len(adds) > 0 {
		args = append([]string{"add"}, adds...)
		code, out, err = aghos.RunCommand("git", args...)

		if err != nil || code != 0 {
			return fmt.Errorf("git add exited with code %d output %q: %w", code, out, err)
		}
	}

	if len(dels) > 0 {
		args = append([]string{"restore"}, dels...)
		code, out, err = aghos.RunCommand("git", args...)

		if err != nil || code != 0 {
			return fmt.Errorf("git restore exited with code %d output %q: %w", code, out, err)
		}
	}

	return nil
}

// changedLocales returns cleaned paths of locales with changes or error.  adds
// is the list of locales with only additions.  dels is the list of locales
// with only deletions.
func changedLocales() (adds, dels []string, err error) {
	defer func() { err = errors.Annotate(err, "getting changes: %w") }()

	cmd := exec.Command("git", "diff", "--numstat", localesDir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("piping: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("starting: %w", err)
	}

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, nil, fmt.Errorf("invalid input: %q", line)
		}

		path := fields[2]

		if fields[0] == "0" {
			dels = append(dels, path)
		} else if fields[1] == "0" {
			adds = append(adds, path)
		}
	}

	err = scanner.Err()
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("waiting: %w", err)
	}

	return adds, dels, nil
}
