// translations downloads translations, uploads translations, prints summary
// for translations, prints unused strings.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
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

	readLimit     = 1 * 1024 * 1024
	uploadTimeout = 20 * time.Second
)

// blockerLangCodes is the codes of languages which need to be fully translated.
var blockerLangCodes = []langCode{
	"de",
	"en",
	"es",
	"fr",
	"it",
	"ja",
	"ko",
	"pt-br",
	"pt-pt",
	"ru",
	"zh-cn",
	"zh-tw",
}

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

	conf, err := readTwoskyConfig()
	check(err)

	var cli *twoskyClient

	switch os.Args[1] {
	case "summary":
		err = summary(conf.Languages)
	case "download":
		cli, err = conf.toClient()
		check(err)

		err = cli.download()
	case "unused":
		err = unused(conf.LocalizableFiles[0])
	case "upload":
		cli, err = conf.toClient()
		check(err)

		err = cli.upload()
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

// twoskyConfig is the configuration structure for localization.
type twoskyConfig struct {
	Languages        languages `json:"languages"`
	ProjectID        string    `json:"project_id"`
	BaseLangcode     langCode  `json:"base_locale"`
	LocalizableFiles []string  `json:"localizable_files"`
}

// readTwoskyConfig returns twosky configuration.
func readTwoskyConfig() (t *twoskyConfig, err error) {
	defer func() { err = errors.Annotate(err, "parsing twosky config: %w") }()

	b, err := os.ReadFile(twoskyConfFile)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	var tsc []twoskyConfig
	err = json.Unmarshal(b, &tsc)
	if err != nil {
		err = fmt.Errorf("unmarshalling %q: %w", twoskyConfFile, err)

		return nil, err
	}

	if len(tsc) == 0 {
		err = fmt.Errorf("%q is empty", twoskyConfFile)

		return nil, err
	}

	conf := tsc[0]

	for _, lang := range conf.Languages {
		if lang == "" {
			return nil, errors.Error("language is empty")
		}
	}

	if len(conf.LocalizableFiles) == 0 {
		return nil, errors.Error("no localizable files specified")
	}

	return &conf, nil
}

// twoskyClient is the twosky client with methods for download and upload
// translations.
type twoskyClient struct {
	// uri is the base URL.
	uri *url.URL

	// projectID is the name of the project.
	projectID string

	// baseLang is the base language code.
	baseLang langCode

	// langs is the list of codes of languages to download.
	langs []langCode
}

// toClient reads values from environment variables or defaults, validates
// them, and returns the twosky client.
func (t *twoskyConfig) toClient() (cli *twoskyClient, err error) {
	defer func() { err = errors.Annotate(err, "filling config: %w") }()

	uriStr := os.Getenv("TWOSKY_URI")
	if uriStr == "" {
		uriStr = twoskyURI
	}
	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, err
	}

	projectID := os.Getenv("TWOSKY_PROJECT_ID")
	if projectID == "" {
		projectID = defaultProjectID
	}

	baseLang := t.BaseLangcode
	uLangStr := os.Getenv("UPLOAD_LANGUAGE")
	if uLangStr != "" {
		baseLang = langCode(uLangStr)
	}

	langs := maps.Keys(t.Languages)
	dlLangStr := os.Getenv("DOWNLOAD_LANGUAGES")
	if dlLangStr == "blocker" {
		langs = blockerLangCodes
	} else if dlLangStr != "" {
		var dlLangs []langCode
		dlLangs, err = validateLanguageStr(dlLangStr, t.Languages)
		if err != nil {
			return nil, err
		}

		langs = dlLangs
	}

	return &twoskyClient{
		uri:       uri,
		projectID: projectID,
		baseLang:  baseLang,
		langs:     langs,
	}, nil
}

// validateLanguageStr validates languages codes that contain in the str and
// returns them or error.
func validateLanguageStr(str string, all languages) (langs []langCode, err error) {
	codes := strings.Fields(str)
	langs = make([]langCode, 0, len(codes))

	for _, k := range codes {
		lc := langCode(k)
		_, ok := all[lc]
		if !ok {
			return nil, fmt.Errorf("validating languages: unexpected language code %q", k)
		}

		langs = append(langs, lc)
	}

	return langs, nil
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

		blocker := ""

		// N is small enough to not raise performance questions.
		ok := slices.Contains(blockerLangCodes, lang)
		if ok {
			blocker = " (blocker)"
		}

		fmt.Printf("%s\t %6.2f %%%s\n", lang, f, blocker)
	}

	return nil
}

// unused prints unused text labels.
func unused(basePath string) (err error) {
	defer func() { err = errors.Annotate(err, "unused: %w") }()

	baseLoc, err := readLocales(basePath)
	if err != nil {
		return err
	}

	locDir := filepath.Clean(localesDir)
	js, err := findJS(locDir)
	if err != nil {
		return err
	}

	return findUnused(js, baseLoc)
}

// findJS returns list of JavaScript and JSON files or error.
func findJS(locDir string) (fileNames []string, err error) {
	walkFn := func(name string, _ os.FileInfo, err error) error {
		if err != nil {
			log.Info("warning: accessing a path %q: %s", name, err)

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
	}

	err = filepath.Walk(srcDir, walkFn)
	if err != nil {
		return nil, fmt.Errorf("filepath walking %q: %w", srcDir, err)
	}

	return fileNames, nil
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

	err = handleAdds(adds)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil
	}

	err = handleDels(dels)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil
	}

	return nil
}

// handleAdds adds locales with additions to the git.
func handleAdds(locales []string) (err error) {
	if len(locales) == 0 {
		return nil
	}

	args := append([]string{"add"}, locales...)
	code, out, err := aghos.RunCommand("git", args...)

	if err != nil || code != 0 {
		return fmt.Errorf("git add exited with code %d output %q: %w", code, out, err)
	}

	return nil
}

// handleDels restores locales with deletions.
func handleDels(locales []string) (err error) {
	if len(locales) == 0 {
		return nil
	}

	args := append([]string{"restore"}, locales...)
	code, out, err := aghos.RunCommand("git", args...)

	if err != nil || code != 0 {
		return fmt.Errorf("git restore exited with code %d output %q: %w", code, out, err)
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
