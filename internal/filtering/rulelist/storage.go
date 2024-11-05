package rulelist

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/c2h5oh/datasize"
)

// Storage contains the main filtering engines, including the allowlist, the
// blocklist, and the user's custom filtering rules.
type Storage struct {
	// refreshMu makes sure that only one update takes place at a time.
	refreshMu *sync.Mutex

	allow    *Engine
	block    *Engine
	custom   *TextEngine
	httpCli  *http.Client
	cacheDir string
	parseBuf []byte
	maxSize  datasize.ByteSize
}

// StorageConfig is the configuration for the filtering-engine storage.
type StorageConfig struct {
	// Logger is used to log the operation of the storage.  It must not be nil.
	Logger *slog.Logger

	// HTTPClient is the HTTP client used to perform updates of rule lists.
	// It must not be nil.
	HTTPClient *http.Client

	// CacheDir is the path to the directory used to cache rule-list files.
	// It must be set.
	CacheDir string

	// AllowFilters are the filtering-rule lists used to exclude domain names
	// from the filtering.  Each item must not be nil.
	AllowFilters []*Filter

	// BlockFilters are the filtering-rule lists used to block domain names.
	// Each item must not be nil.
	BlockFilters []*Filter

	// CustomRules contains custom rules of the user.  They have priority over
	// both allow- and blacklist rules.
	CustomRules []string

	// MaxRuleListTextSize is the maximum size of a rule-list file.  It must be
	// greater than zero.
	MaxRuleListTextSize datasize.ByteSize
}

// NewStorage creates a new filtering-engine storage.  The engines are not
// refreshed, so a refresh should be performed before use.
func NewStorage(c *StorageConfig) (s *Storage, err error) {
	custom, err := NewTextEngine(&TextEngineConfig{
		Name:  EngineNameCustom,
		Rules: c.CustomRules,
		ID:    URLFilterIDCustom,
	})
	if err != nil {
		return nil, fmt.Errorf("creating custom engine: %w", err)
	}

	return &Storage{
		refreshMu: &sync.Mutex{},
		allow: NewEngine(&EngineConfig{
			Logger:  c.Logger.With("engine", EngineNameAllow),
			Name:    EngineNameAllow,
			Filters: c.AllowFilters,
		}),
		block: NewEngine(&EngineConfig{
			Logger:  c.Logger.With("engine", EngineNameBlock),
			Name:    EngineNameBlock,
			Filters: c.BlockFilters,
		}),
		custom:   custom,
		httpCli:  c.HTTPClient,
		cacheDir: c.CacheDir,
		parseBuf: make([]byte, DefaultRuleBufSize),
		maxSize:  c.MaxRuleListTextSize,
	}, nil
}

// Close closes the underlying rule-list engines.
func (s *Storage) Close() (err error) {
	// Don't wrap the errors since they are informative enough as is.
	return errors.Join(
		s.allow.Close(),
		s.block.Close(),
	)
}

// Refresh updates all engines in s.
//
// TODO(a.garipov): Refresh allow and block separately?
func (s *Storage) Refresh(ctx context.Context) (err error) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	// Don't wrap the errors since they are informative enough as is.
	return errors.Join(
		s.allow.Refresh(ctx, s.parseBuf, s.httpCli, s.cacheDir, s.maxSize),
		s.block.Refresh(ctx, s.parseBuf, s.httpCli, s.cacheDir, s.maxSize),
	)
}
