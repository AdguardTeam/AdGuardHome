// Package configmgr defines the AdGuard Home on-disk configuration entities and
// configuration manager.
//
// TODO(a.garipov): Add tests.
package configmgr

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/netip"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/renameio/v2/maybe"
	"gopkg.in/yaml.v3"
)

// Manager handles full and partial changes in the configuration, persisting
// them to disk if necessary.
//
// TODO(a.garipov): Support missing configs and default values.
type Manager struct {
	// baseLogger is used to create loggers for other entities.
	baseLogger *slog.Logger

	// logger is used for logging the operation of the configuration manager.
	logger *slog.Logger

	// updMu makes sure that at most one reconfiguration is performed at a time.
	// updMu protects all fields below.
	updMu *sync.RWMutex

	// dns is the DNS service.
	dns *dnssvc.Service

	// Web is the Web API service.
	web *websvc.Service

	// current is the current configuration.
	current *config

	// fileName is the name of the configuration file.
	fileName string
}

// Validate returns an error if the configuration file with the given name does
// not exist or is invalid.
func Validate(fileName string) (err error) {
	conf, err := read(fileName)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	err = conf.Validate()
	if err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	return nil
}

// Config contains the configuration parameters for the configuration manager.
type Config struct {
	// BaseLogger is used to create loggers for other entities.  It must not be
	// nil.
	BaseLogger *slog.Logger

	// Logger is used for logging the operation of the configuration manager.
	// It must not be nil.
	Logger *slog.Logger

	// Frontend is the filesystem with the frontend files.
	Frontend fs.FS

	// WebAddr is the initial or override address for the Web UI.  It is not
	// written to the configuration file.
	WebAddr netip.AddrPort

	// Start is the time of start of AdGuard Home.
	Start time.Time

	// FileName is the path to the configuration file.
	FileName string
}

// New creates a new *Manager that persists changes to the file pointed to by
// c.FileName.  It reads the configuration file and populates the service
// fields.  c must not be nil.
func New(ctx context.Context, c *Config) (m *Manager, err error) {
	conf, err := read(c.FileName)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	err = conf.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	m = &Manager{
		baseLogger: c.BaseLogger,
		logger:     c.Logger,
		updMu:      &sync.RWMutex{},
		current:    conf,
		fileName:   c.FileName,
	}

	err = m.assemble(ctx, conf, c.Frontend, c.WebAddr, c.Start)
	if err != nil {
		return nil, fmt.Errorf("creating config manager: %w", err)
	}

	return m, nil
}

// read reads and decodes configuration from the provided filename.
func read(fileName string) (conf *config, err error) {
	defer func() { err = errors.Annotate(err, "reading config: %w") }()

	conf = &config{}
	f, err := os.Open(fileName)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	err = yaml.NewDecoder(f).Decode(conf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	return conf, nil
}

// assemble creates all services and puts them into the corresponding fields.
// The fields of conf must not be modified after calling assemble.
func (m *Manager) assemble(
	ctx context.Context,
	conf *config,
	frontend fs.FS,
	webAddr netip.AddrPort,
	start time.Time,
) (err error) {
	dnsConf := &dnssvc.Config{
		Logger:              m.baseLogger.With(slogutil.KeyPrefix, "dnssvc"),
		Addresses:           conf.DNS.Addresses,
		BootstrapServers:    conf.DNS.BootstrapDNS,
		UpstreamServers:     conf.DNS.UpstreamDNS,
		DNS64Prefixes:       conf.DNS.DNS64Prefixes,
		UpstreamTimeout:     time.Duration(conf.DNS.UpstreamTimeout),
		BootstrapPreferIPv6: conf.DNS.BootstrapPreferIPv6,
		UseDNS64:            conf.DNS.UseDNS64,
	}
	err = m.updateDNS(ctx, dnsConf)
	if err != nil {
		return fmt.Errorf("assembling dnssvc: %w", err)
	}

	webSvcConf := &websvc.Config{
		Logger: m.baseLogger.With(slogutil.KeyPrefix, "websvc"),
		Pprof: &websvc.PprofConfig{
			Port:    conf.HTTP.Pprof.Port,
			Enabled: conf.HTTP.Pprof.Enabled,
		},
		ConfigManager: m,
		Frontend:      frontend,
		// TODO(a.garipov): Fill from config file.
		TLS:             nil,
		Start:           start,
		Addresses:       conf.HTTP.Addresses,
		SecureAddresses: conf.HTTP.SecureAddresses,
		OverrideAddress: webAddr,
		Timeout:         time.Duration(conf.HTTP.Timeout),
		ForceHTTPS:      conf.HTTP.ForceHTTPS,
	}

	err = m.updateWeb(ctx, webSvcConf)
	if err != nil {
		return fmt.Errorf("assembling websvc: %w", err)
	}

	return nil
}

// write writes the current configuration to disk.
func (m *Manager) write(ctx context.Context) (err error) {
	b, err := yaml.Marshal(m.current)
	if err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	err = maybe.WriteFile(m.fileName, b, aghos.DefaultPermFile)
	if err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	m.logger.InfoContext(ctx, "config file written", "path", m.fileName)

	return nil
}

// DNS returns the current DNS service.  It is safe for concurrent use.
func (m *Manager) DNS() (dns agh.ServiceWithConfig[*dnssvc.Config]) {
	m.updMu.RLock()
	defer m.updMu.RUnlock()

	return m.dns
}

// UpdateDNS implements the [websvc.ConfigManager] interface for *Manager.  The
// fields of c must not be modified after calling UpdateDNS.
func (m *Manager) UpdateDNS(ctx context.Context, c *dnssvc.Config) (err error) {
	m.updMu.Lock()
	defer m.updMu.Unlock()

	// TODO(a.garipov): Update and write the configuration file.  Return an
	// error if something went wrong.

	err = m.updateDNS(ctx, c)
	if err != nil {
		return fmt.Errorf("reassembling dnssvc: %w", err)
	}

	m.updateCurrentDNS(c)

	return m.write(ctx)
}

// updateDNS recreates the DNS service.  m.updMu is expected to be locked.
func (m *Manager) updateDNS(ctx context.Context, c *dnssvc.Config) (err error) {
	if prev := m.dns; prev != nil {
		err = prev.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutting down dns svc: %w", err)
		}
	}

	svc, err := dnssvc.New(c)
	if err != nil {
		return fmt.Errorf("creating dns svc: %w", err)
	}

	m.dns = svc

	return nil
}

// updateCurrentDNS updates the DNS configuration in the current config.
func (m *Manager) updateCurrentDNS(c *dnssvc.Config) {
	m.current.DNS.Addresses = slices.Clone(c.Addresses)
	m.current.DNS.BootstrapDNS = slices.Clone(c.BootstrapServers)
	m.current.DNS.UpstreamDNS = slices.Clone(c.UpstreamServers)
	m.current.DNS.DNS64Prefixes = slices.Clone(c.DNS64Prefixes)
	m.current.DNS.UpstreamTimeout = timeutil.Duration(c.UpstreamTimeout)
	m.current.DNS.BootstrapPreferIPv6 = c.BootstrapPreferIPv6
	m.current.DNS.UseDNS64 = c.UseDNS64
}

// Web returns the current web service.  It is safe for concurrent use.
func (m *Manager) Web() (web agh.ServiceWithConfig[*websvc.Config]) {
	m.updMu.RLock()
	defer m.updMu.RUnlock()

	return m.web
}

// UpdateWeb implements the [websvc.ConfigManager] interface for *Manager.  The
// fields of c must not be modified after calling UpdateWeb.
func (m *Manager) UpdateWeb(ctx context.Context, c *websvc.Config) (err error) {
	m.updMu.Lock()
	defer m.updMu.Unlock()

	err = m.updateWeb(ctx, c)
	if err != nil {
		return fmt.Errorf("reassembling websvc: %w", err)
	}

	m.updateCurrentWeb(c)

	return m.write(ctx)
}

// updateWeb recreates the web service.  m.upd is expected to be locked.
func (m *Manager) updateWeb(ctx context.Context, c *websvc.Config) (err error) {
	if prev := m.web; prev != nil {
		err = prev.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("shutting down web svc: %w", err)
		}
	}

	m.web, err = websvc.New(c)
	if err != nil {
		return fmt.Errorf("creating web svc: %w", err)
	}

	return nil
}

// updateCurrentWeb updates the web configuration in the current config.
func (m *Manager) updateCurrentWeb(c *websvc.Config) {
	// TODO(a.garipov): Update pprof from API?

	m.current.HTTP.Addresses = slices.Clone(c.Addresses)
	m.current.HTTP.SecureAddresses = slices.Clone(c.SecureAddresses)
	m.current.HTTP.Timeout = timeutil.Duration(c.Timeout)
	m.current.HTTP.ForceHTTPS = c.ForceHTTPS
}
