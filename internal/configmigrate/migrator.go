package configmigrate

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	yaml "go.yaml.in/yaml/v4"
)

// Config is a the configuration for initializing a [Migrator].
type Config struct {
	// Logger is used to log the operation of configuration migrator.  It must
	// not be nil.
	Logger *slog.Logger

	// WorkingDir is the absolute path to the working directory of AdGuardHome.
	WorkingDir string

	// DataDir is the absolute path to the data directory of AdGuardHome.
	DataDir string
}

// Migrator performs the YAML configuration file migrations.
type Migrator struct {
	logger     *slog.Logger
	workingDir string
	dataDir    string
}

// New creates a new Migrator.
func New(c *Config) (m *Migrator) {
	return &Migrator{
		logger:     c.Logger,
		workingDir: c.WorkingDir,
		dataDir:    c.DataDir,
	}
}

// Migrate preforms necessary upgrade operations to upgrade file to target
// schema version, if needed.  It returns the body of the upgraded config file,
// whether the file was upgraded, and an error, if any.  If upgraded is false,
// the body is the same as the input.
func (m *Migrator) Migrate(
	ctx context.Context,
	body []byte,
	target uint,
) (newBody []byte, upgraded bool, err error) {
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
	m.logger.DebugContext(ctx, "got", "schema_version", current)

	if err = validateVersion(current, target); err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return body, false, err
	} else if current == target {
		return body, false, nil
	}

	if err = m.upgradeConfigSchema(ctx, current, target, diskConf); err != nil {
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
type migrateFunc = func(ctx context.Context, diskConf yobj) (err error)

// upgradeConfigSchema upgrades the configuration schema in diskConf from
// current to target version.  current must be less than target, and both must
// be non-negative and less or equal to [LastSchemaVersion].
func (m *Migrator) upgradeConfigSchema(
	ctx context.Context,
	current, target uint,
	diskConf yobj,
) (err error) {
	upgrades := [LastSchemaVersion]migrateFunc{
		0:  m.migrateTo1,
		1:  m.migrateTo2,
		2:  m.migrateTo3,
		3:  m.migrateTo4,
		4:  m.migrateTo5,
		5:  m.migrateTo6,
		6:  m.migrateTo7,
		7:  m.migrateTo8,
		8:  m.migrateTo9,
		9:  m.migrateTo10,
		10: m.migrateTo11,
		11: m.migrateTo12,
		12: m.migrateTo13,
		13: m.migrateTo14,
		14: m.migrateTo15,
		15: m.migrateTo16,
		16: m.migrateTo17,
		17: m.migrateTo18,
		18: m.migrateTo19,
		19: m.migrateTo20,
		20: m.migrateTo21,
		21: m.migrateTo22,
		22: m.migrateTo23,
		23: m.migrateTo24,
		24: m.migrateTo25,
		25: m.migrateTo26,
		26: m.migrateTo27,
		27: m.migrateTo28,
		28: m.migrateTo29,
		29: m.migrateTo30,
		30: m.migrateTo31,
		31: m.migrateTo32,
		32: m.migrateTo33,
	}

	for i, migrate := range upgrades[current:target] {
		cur := current + uint(i)
		next := current + uint(i) + 1

		m.logger.InfoContext(ctx, "upgrade yaml", "from", cur, "to", next)

		if err = migrate(ctx, diskConf); err != nil {
			return fmt.Errorf("migrating schema %d to %d: %w", cur, next, err)
		}
	}

	return nil
}
