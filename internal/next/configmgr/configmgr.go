// Package configmgr defines the AdGuard Home on-disk configuration entities and
// configuration manager.
//
// TODO(a.garipov): Add tests.
package configmgr

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/google/renameio/maybe"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// Configuration Manager

// Manager handles full and partial changes in the configuration, persisting
// them to disk if necessary.
type Manager struct {
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

	// Don't wrap the error, because it's informative enough as is.
	return conf.validate()
}

// New creates a new *Manager that persists changes to the file pointed to by
// fileName.  It reads the configuration file and populates the service fields.
// start is the startup time of AdGuard Home.
func New(
	ctx context.Context,
	fileName string,
	frontend fs.FS,
	start time.Time,
) (m *Manager, err error) {
	conf, err := read(fileName)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return nil, err
	}

	err = conf.validate()
	if err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	m = &Manager{
		updMu:    &sync.RWMutex{},
		current:  conf,
		fileName: fileName,
	}

	err = m.assemble(ctx, conf, frontend, start)
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
	start time.Time,
) (err error) {
	dnsConf := &dnssvc.Config{
		Addresses:           conf.DNS.Addresses,
		BootstrapServers:    conf.DNS.BootstrapDNS,
		UpstreamServers:     conf.DNS.UpstreamDNS,
		DNS64Prefixes:       conf.DNS.DNS64Prefixes,
		UpstreamTimeout:     conf.DNS.UpstreamTimeout.Duration,
		BootstrapPreferIPv6: conf.DNS.BootstrapPreferIPv6,
		UseDNS64:            conf.DNS.UseDNS64,
	}
	err = m.updateDNS(ctx, dnsConf)
	if err != nil {
		return fmt.Errorf("assembling dnssvc: %w", err)
	}

	webSvcConf := &websvc.Config{
		ConfigManager: m,
		Frontend:      frontend,
		// TODO(a.garipov): Fill from config file.
		TLS:             nil,
		Start:           start,
		Addresses:       conf.HTTP.Addresses,
		SecureAddresses: conf.HTTP.SecureAddresses,
		Timeout:         conf.HTTP.Timeout.Duration,
		ForceHTTPS:      conf.HTTP.ForceHTTPS,
	}

	err = m.updateWeb(ctx, webSvcConf)
	if err != nil {
		return fmt.Errorf("assembling websvc: %w", err)
	}

	return nil
}

// write writes the current configuration to disk.
func (m *Manager) write() (err error) {
	b, err := yaml.Marshal(m.current)
	if err != nil {
		return fmt.Errorf("encoding: %w", err)
	}

	err = maybe.WriteFile(m.fileName, b, 0o755)
	if err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	log.Info("configmgr: written to %q", m.fileName)

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

	return m.write()
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
	m.current.DNS.UpstreamTimeout = timeutil.Duration{Duration: c.UpstreamTimeout}
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

	// TODO(a.garipov): Update and write the configuration file.  Return an
	// error if something went wrong.

	err = m.updateWeb(ctx, c)
	if err != nil {
		return fmt.Errorf("reassembling websvc: %w", err)
	}

	m.updateCurrentWeb(c)

	return m.write()
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
	m.current.HTTP.Addresses = slices.Clone(c.Addresses)
	m.current.HTTP.SecureAddresses = slices.Clone(c.SecureAddresses)
	m.current.HTTP.Timeout = timeutil.Duration{Duration: c.Timeout}
	m.current.HTTP.ForceHTTPS = c.ForceHTTPS
}
