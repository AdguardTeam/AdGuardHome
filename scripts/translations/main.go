// translations downloads translations, uploads translations, prints summary
// for translations, prints unused strings.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/c2h5oh/datasize"
)

// TODO(e.burkov):  Remove the default as they should be set by configuration.
const (
	twoskyConfFile  = "./.twosky.json"
	localesDirHome  = "./client/src/__locales"
	defaultBaseFile = "en.json"
	srcDir          = "./client/src"
	twoskyURI       = "https://twosky.int.agrd.dev/api/v1"

	readLimit     = 1 * datasize.MB
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
	ctx := context.Background()
	l := slogutil.New(nil)

	if len(os.Args) == 1 {
		usage("need a command")
	}

	if os.Args[1] == "help" {
		usage("")
	}

	homeConf, servicesConf, err := readTwoskyConfig()
	errors.Check(err)

	var cli *twoskyClient

	switch os.Args[1] {
	case "summary":
		errors.Check(summary(homeConf.Languages))
	case "download":
		cli = errors.Must(newTwoskyClient(homeConf))
		cli.download(ctx, l)

		cli = errors.Must(newTwoskyClient(servicesConf))
		cli.download(ctx, l)
	case "unused":
		errors.Check(unused(ctx, l, homeConf.LocalizableFiles[0]))
	case "upload":
		cli = errors.Must(newTwoskyClient(homeConf))
		errors.Check(cli.upload())
	case "auto-add":
		errors.Check(autoAdd(ctx, l, homeConf.LocalizableFiles[0]))
	default:
		usage("unknown command")
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

		os.Exit(osutil.ExitCodeFailure)
	}

	fmt.Println(usageStr)

	os.Exit(osutil.ExitCodeSuccess)
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
//
// TODO(e.burkov):  Consider making it a method of [twoskyClient] and
// calculating summary for all configurations.
func summary(langs languages) (err error) {
	basePath := filepath.Join(localesDirHome, defaultBaseFile)
	baseLoc, err := readLocales(basePath)
	if err != nil {
		return fmt.Errorf("summary: %w", err)
	}

	size := float64(len(baseLoc))

	for _, lang := range slices.Sorted(maps.Keys(langs)) {
		name := filepath.Join(localesDirHome, string(lang)+".json")
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
//
// TODO(e.burkov):  Consider making it a method of [twoskyClient] and searching
// unused strings for all configurations.
func unused(ctx context.Context, l *slog.Logger, basePath string) (err error) {
	defer func() { err = errors.Annotate(err, "unused: %w") }()

	baseLoc, err := readLocales(basePath)
	if err != nil {
		return err
	}

	locDir := filepath.Clean(localesDirHome)
	js, err := findJS(ctx, l, locDir)
	if err != nil {
		return err
	}

	return findUnused(js, baseLoc)
}

// findJS returns list of JavaScript and JSON files or error.
func findJS(ctx context.Context, l *slog.Logger, locDir string) (fileNames []string, err error) {
	walkFn := func(name string, _ os.FileInfo, err error) error {
		if err != nil {
			l.WarnContext(ctx, "accessing a path", slogutil.KeyError, err)

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

	for _, v := range slices.Sorted(maps.Keys(loc)) {
		fmt.Println(v)
	}

	return nil
}

// autoAdd adds locales with additions to the git and restores locales with
// deletions.
func autoAdd(ctx context.Context, l *slog.Logger, basePath string) (err error) {
	defer func() { err = errors.Annotate(err, "auto add: %w") }()

	cmdCons := executil.SystemCommandConstructor{}

	adds, dels, err := changedLocales(ctx, l, cmdCons)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	if slices.Contains(dels, basePath) {
		return errors.Error("base locale contains deletions")
	}

	err = handleAdds(ctx, l, cmdCons, adds)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil
	}

	err = handleDels(ctx, l, cmdCons, dels)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil
	}

	return nil
}

// gitCmd is the shell command for Git.
const gitCmd = "git"

// handleAdds adds locales with additions to the git.
func handleAdds(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	locales []string,
) (err error) {
	if len(locales) == 0 {
		return nil
	}

	gitArgs := append([]string{"add"}, locales...)
	l.DebugContext(ctx, "executing", "cmd", gitCmd, "args", gitArgs)

	code, out, err := aghos.RunCommand(ctx, cmdCons, gitCmd, gitArgs...)

	if err != nil || code != 0 {
		return fmt.Errorf("git add exited with code %d output %q: %w", code, out, err)
	}

	return nil
}

// handleDels restores locales with deletions.
func handleDels(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
	locales []string,
) (err error) {
	if len(locales) == 0 {
		return nil
	}

	gitArgs := append([]string{"restore"}, locales...)
	l.DebugContext(ctx, "executing", "cmd", gitCmd, "args", gitArgs)

	code, out, err := aghos.RunCommand(ctx, cmdCons, gitCmd, gitArgs...)

	if err != nil || code != 0 {
		return fmt.Errorf("git restore exited with code %d output %q: %w", code, out, err)
	}

	return nil
}

// changedLocales returns cleaned paths of locales with changes or error.  adds
// is the list of locales with only additions.  dels is the list of locales
// with only deletions.
func changedLocales(
	ctx context.Context,
	l *slog.Logger,
	cmdCons executil.CommandConstructor,
) (adds, dels []string, err error) {
	defer func() { err = errors.Annotate(err, "getting changes: %w") }()

	gitArgs := []string{"diff", "--numstat", localesDirHome}
	l.DebugContext(ctx, "executing", "cmd", gitCmd, "args", gitArgs)

	// TODO(s.chzhen):  Consider streaming the output if needed.  Using
	// [io.Pipe] here is unnecessary; it complicates lifecycle management
	// because the output must be read concurrently, and the PipeWriter must be
	// explicitly closed to signal EOF.  Since this command's output is small, a
	// bytes.Buffer via executil.Run is sufficient.
	var out bytes.Buffer
	err = executil.Run(ctx, cmdCons, &executil.CommandConfig{
		Path:   gitCmd,
		Args:   gitArgs,
		Stdout: &out,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("executing cmd: %w", err)
	}

	scanner := bufio.NewScanner(&out)

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

	return adds, dels, nil
}
