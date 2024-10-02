package configmigrate

import (
	"bytes"
	"fmt"

	"github.com/AdguardTeam/golibs/log"
	yaml "gopkg.in/yaml.v3"
)

// Config is a the configuration for initializing a [Migrator].
type Config struct {
	// WorkingDir is the absolute path to the working directory of AdGuardHome.
	WorkingDir string

	// DataDir is the absolute path to the data directory of AdGuardHome.
	DataDir string
}

// Migrator performs the YAML configuration file migrations.
type Migrator struct {
	workingDir string
	dataDir    string
}

// New creates a new Migrator.
func New(c *Config) (m *Migrator) {
	return &Migrator{
		workingDir: c.WorkingDir,
		dataDir:    c.DataDir,
	}
}

// Migrate preforms necessary upgrade operations to upgrade file to target
// schema version, if needed.  It returns the body of the upgraded config file,
// whether the file was upgraded, and an error, if any.  If upgraded is false,
// the body is the same as the input.
func (m *Migrator) Migrate(body []byte, target uint) (newBody []byte, upgraded bool, err error) {
	diskConf := yobj{}
	err = yaml.Unmarshal(body, &diskConf)
	if err != nil {
		return body, false, fmt.Errorf("parsing config file for upgrade: %w", err)
	}

	currentInt, _, err := fieldVal[int](diskConf, "schema_version")
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return body, false, err
	}

	current := uint(currentInt)
	log.Debug("got schema version %v", current)

	if err = validateVersion(current, target); err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return body, false, err
	} else if current == target {
		return body, false, nil
	}

	if err = m.upgradeConfigSchema(current, target, diskConf); err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return body, false, err
	}

	buf := bytes.NewBuffer(newBody)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	if err = enc.Encode(diskConf); err != nil {
		return body, false, fmt.Errorf("generating new config: %w", err)
	}

	return buf.Bytes(), true, nil
}

// validateVersion validates the current and desired schema versions.
func validateVersion(current, target uint) (err error) {
	switch {
	case current > target:
		return fmt.Errorf("unknown current schema version %d", current)
	case target > LastSchemaVersion:
		return fmt.Errorf("unknown target schema version %d", target)
	case target < current:
		return fmt.Errorf("target schema version %d lower than current %d", target, current)
	default:
		return nil
	}
}

// migrateFunc is a function that upgrades a config and returns an error.
type migrateFunc = func(diskConf yobj) (err error)

// upgradeConfigSchema upgrades the configuration schema in diskConf from
// current to target version.  current must be less than target, and both must
// be non-negative and less or equal to [LastSchemaVersion].
func (m *Migrator) upgradeConfigSchema(current, target uint, diskConf yobj) (err error) {
	upgrades := [LastSchemaVersion]migrateFunc{
		0:  m.migrateTo1,
		1:  m.migrateTo2,
		2:  migrateTo3,
		3:  migrateTo4,
		4:  migrateTo5,
		5:  migrateTo6,
		6:  migrateTo7,
		7:  migrateTo8,
		8:  migrateTo9,
		9:  migrateTo10,
		10: migrateTo11,
		11: migrateTo12,
		12: migrateTo13,
		13: migrateTo14,
		14: migrateTo15,
		15: migrateTo16,
		16: migrateTo17,
		17: migrateTo18,
		18: migrateTo19,
		19: migrateTo20,
		20: migrateTo21,
		21: migrateTo22,
		22: migrateTo23,
		23: migrateTo24,
		24: migrateTo25,
		25: migrateTo26,
		26: migrateTo27,
		27: migrateTo28,
		28: m.migrateTo29,
	}

	for i, migrate := range upgrades[current:target] {
		cur := current + uint(i)
		next := current + uint(i) + 1

		log.Printf("Upgrade yaml: %d to %d", cur, next)

		if err = migrate(diskConf); err != nil {
			return fmt.Errorf("migrating schema %d to %d: %w", cur, next, err)
		}
	}

	return nil
}
