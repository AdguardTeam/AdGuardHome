// Package confmigrate provides a way to upgrade the YAML configuration file.
package confmigrate

import (
	"bytes"
	"fmt"

	"github.com/AdguardTeam/golibs/log"
	yaml "gopkg.in/yaml.v3"
)

// CurrentSchemaVersion is the current schema version.
const CurrentSchemaVersion = 26

// These aliases are provided for convenience.
type (
	yarr = []any
	yobj = map[string]any
)

// Config is a the configuration for initializing a [Migrator].
type Config struct {
	// WorkingDir is an absolute path to the working directory of AdGuardHome.
	WorkingDir string
}

// Migrator performs the YAML configuration file migrations.
type Migrator struct {
	// workingDir is an absolute path to the working directory of AdGuardHome.
	workingDir string
}

// New creates a new Migrator.
func New(cfg *Config) (m *Migrator) {
	return &Migrator{
		workingDir: cfg.WorkingDir,
	}
}

// Migrate does necessary upgrade operations if needed.  It returns the new
// configuration file body, and a boolean indicating whether the configuration
// file was actually upgraded.
func (m *Migrator) Migrate(body []byte) (newBody []byte, upgraded bool, err error) {
	// read a config file into an interface map, so we can manipulate values without losing any
	diskConf := yobj{}
	err = yaml.Unmarshal(body, &diskConf)
	if err != nil {
		log.Printf("parsing config file for upgrade: %s", err)

		return nil, false, err
	}

	schemaVersionVal, ok := diskConf["schema_version"]
	log.Tracef("got schema version %v", schemaVersionVal)
	if !ok {
		// no schema version, set it to 0
		schemaVersionVal = 0
	}

	schemaVersion, ok := schemaVersionVal.(int)
	if !ok {
		err = fmt.Errorf("configuration file contains non-integer schema_version, abort")
		log.Println(err)

		return nil, false, err
	}

	if schemaVersion == CurrentSchemaVersion {
		// do nothing
		return body, false, nil
	}

	err = m.upgradeConfigSchema(schemaVersion, diskConf)
	if err != nil {
		log.Printf("upgrading configuration file: %s", err)

		return nil, false, err
	}

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	err = enc.Encode(diskConf)
	if err != nil {
		return nil, false, fmt.Errorf("generating new config: %w", err)
	}

	return buf.Bytes(), true, nil
}

// upgradeFunc is a function that upgrades a config and returns an error.
type upgradeFunc = func(diskConf yobj) (err error)

// Upgrade from oldVersion to newVersion
func (m *Migrator) upgradeConfigSchema(oldVersion int, diskConf yobj) (err error) {
	upgrades := []upgradeFunc{
		m.upgradeSchema0to1,
		m.upgradeSchema1to2,
		upgradeSchema2to3,
		upgradeSchema3to4,
		upgradeSchema4to5,
		upgradeSchema5to6,
		upgradeSchema6to7,
		upgradeSchema7to8,
		upgradeSchema8to9,
		upgradeSchema9to10,
		upgradeSchema10to11,
		upgradeSchema11to12,
		upgradeSchema12to13,
		upgradeSchema13to14,
		upgradeSchema14to15,
		upgradeSchema15to16,
		upgradeSchema16to17,
		upgradeSchema17to18,
		upgradeSchema18to19,
		upgradeSchema19to20,
		upgradeSchema20to21,
		upgradeSchema21to22,
		upgradeSchema22to23,
		upgradeSchema23to24,
		upgradeSchema24to25,
		upgradeSchema25to26,
	}

	n := 0
	for i, u := range upgrades {
		if i >= oldVersion {
			err = u(diskConf)
			if err != nil {
				return err
			}

			n++
		}
	}

	if n == 0 {
		return fmt.Errorf("unknown configuration schema version %d", oldVersion)
	}

	return nil
}
