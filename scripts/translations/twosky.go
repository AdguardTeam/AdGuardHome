package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"
	"slices"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// Constants for mapping the twosky configurations.
//
// Keep in sync with the .twosky.json file.
const (
	// twoskyProjectIdxHome is the index of the Home project in the localization
	// configuration.
	twoskyProjectIdxHome = iota

	// twoskyProjectIdxServices is the index of the Services project in the
	// localization configuration.
	twoskyProjectIdxServices

	// twoskyProjectCount is the number of projects in the localization
	// configuration.
	twoskyProjectCount
)

// twoskyConfig is the configuration structure for localization of a single
// project.
type twoskyConfig struct {
	Languages        languages `json:"languages"`
	ProjectID        string    `json:"project_id"`
	BaseLangcode     langCode  `json:"base_locale"`
	LocalizableFiles []string  `json:"localizable_files"`
}

// type check
var _ validate.Interface = (*twoskyConfig)(nil)

// Validate implements the [validate.Interface] interface for *twoskyConfig.
func (t *twoskyConfig) Validate() (err error) {
	if t == nil {
		return errors.ErrNoValue
	}

	errs := []error{
		validate.NotEmpty("project_id", t.ProjectID),
		validate.NotEmpty("base_locale", t.BaseLangcode),
		validate.NotEmptySlice("localizable_files", t.LocalizableFiles),
	}

	if len(t.Languages) == 0 {
		errs = append(errs, fmt.Errorf("languages: %w", errors.ErrEmptyValue))
	}

	for code, lang := range t.Languages {
		err = validate.NotEmpty("languages: "+string(code), lang)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// readTwoskyConfig returns twosky configuration.
func readTwoskyConfig() (home, services *twoskyConfig, err error) {
	defer func() { err = errors.Annotate(err, "parsing twosky config: %w") }()

	b, err := os.ReadFile(twoskyConfFile)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, nil, err
	}

	var tsc []*twoskyConfig
	err = json.Unmarshal(b, &tsc)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshalling %q: %w", twoskyConfFile, err)
	}

	err = validate.Equal("projects count", len(tsc), twoskyProjectCount)
	if err != nil {
		return nil, nil, err
	}

	err = errors.Join(validate.AppendSlice(nil, "projects", tsc)...)
	if err != nil {
		return nil, nil, err
	}

	return tsc[twoskyProjectIdxHome], tsc[twoskyProjectIdxServices], nil
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

	// localizableFiles are the files to localize.
	localizableFiles []string
}

// newTwoskyClient reads values from environment variables or defaults,
// validates them, and returns the twosky client.
func newTwoskyClient(conf *twoskyConfig) (cli *twoskyClient, err error) {
	defer func() { err = errors.Annotate(err, "filling config: %w") }()

	uriStr := cmp.Or(os.Getenv("TWOSKY_URI"), twoskyURI)
	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, err
	}

	// TODO(e.burkov):  Don't use env.
	projectID := conf.ProjectID
	envProjectID := os.Getenv("PROJECT_ID")
	if envProjectID != "" {
		projectID = envProjectID
	}

	baseLang := conf.BaseLangcode
	uLangStr := os.Getenv("UPLOAD_LANGUAGE")
	if uLangStr != "" {
		baseLang = langCode(uLangStr)
	}

	langs := slices.Sorted(maps.Keys(conf.Languages))

	dlLangStr := os.Getenv("DOWNLOAD_LANGUAGES")
	if dlLangStr == "blocker" {
		langs = blockerLangCodes
	} else if dlLangStr != "" {
		var dlLangs []langCode
		dlLangs, err = validateLanguageStr(dlLangStr, conf.Languages)
		if err != nil {
			return nil, err
		}

		langs = dlLangs
	}

	return &twoskyClient{
		uri:              uri,
		projectID:        projectID,
		baseLang:         baseLang,
		langs:            langs,
		localizableFiles: conf.LocalizableFiles,
	}, nil
}
