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

// metadata represents additional information for a single project.
type metadata struct {
	LocalesDir string `json:"locales_dir"`
	SourcesDir string `json:"sources_dir"`
}

// twoskyConfig is the configuration structure for localization of a single
// project.
type twoskyConfig struct {
	Languages        languages `json:"languages"`
	ProjectID        string    `json:"project_id"`
	BaseLangCode     langCode  `json:"base_locale"`
	LocalizableFiles []string  `json:"localizable_files"`
	Metadata         metadata  `json:"metadata"`
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
		validate.NotEmpty("base_locale", t.BaseLangCode),
		validate.NotEmptySlice("localizable_files", t.LocalizableFiles),
		validate.NotEmpty("metadata", t.Metadata),
		validate.NotEmpty("metadata.locales_dir", t.Metadata.LocalesDir),
		validate.NotEmpty("metadata.sources_dir", t.Metadata.SourcesDir),
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

// newTwoskyConfig returns new twosky configuration.
func newTwoskyConfig() (conf *twoskyConfig, err error) {
	defer func() { err = errors.Annotate(err, "parsing twosky config: %w") }()

	b, err := os.ReadFile(twoskyConfFile)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	var confs []*twoskyConfig
	err = json.Unmarshal(b, &confs)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %q: %w", twoskyConfFile, err)
	}

	err = validate.NotEmptySlice("projects", confs)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	err = errors.Join(validate.AppendSlice(nil, "projects", confs)...)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	projectID := cmp.Or(os.Getenv("TWOSKY_PROJECT_ID"), defaultProjectID)
	for _, c := range confs {
		if c.ProjectID == projectID {
			return c, nil
		}
	}

	return nil, fmt.Errorf("project %q not found in %s", projectID, twoskyConfFile)
}

// twoskyClient is the client for the twosky translation service.
type twoskyClient struct {
	// uri is the base URL.
	uri *url.URL

	// baseLang is the base language code.
	baseLang langCode

	// localesDir is the path to the directory with locale files.
	localesDir string

	// projectID is the name of the project.
	projectID string

	// sourcesDir is the path to directory with source files in there the
	// localizations are used.
	sourcesDir string

	// langs is the list of codes of languages.
	langs []langCode

	// localizableFiles are the files to localize.
	localizableFiles []string
}

// newTwoskyClient reads values from environment variables or defaults,
// validates them, and returns the twosky client.  conf must be valid.
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

	baseLang := conf.BaseLangCode
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
		baseLang:         baseLang,
		localesDir:       conf.Metadata.LocalesDir,
		projectID:        projectID,
		sourcesDir:       conf.Metadata.SourcesDir,
		langs:            langs,
		localizableFiles: conf.LocalizableFiles,
	}, nil
}
