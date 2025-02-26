package client

import (
	"log/slog"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/stringutil"
)

// CommonUpstreamConfig contains common settings for custom client upstream
// configurations.
type CommonUpstreamConfig struct {
	Bootstrap               upstream.Resolver
	UpstreamTimeout         time.Duration
	BootstrapPreferIPv6     bool
	EDNSClientSubnetEnabled bool
	UseHTTP3Upstreams       bool
}

// customUpstreamConfig contains custom client upstream configuration and the
// timestamp of the latest configuration update.
type customUpstreamConfig struct {
	prxConf               *proxy.CustomUpstreamConfig
	confUpdate            time.Time
	upstreams             []string
	upstreamsCacheSize    uint32
	upstreamsCacheEnabled bool
}

// upstreamManager stores and updates custom client upstream configurations.
type upstreamManager struct {
	// logger is used for logging the operation of the upstream manager.  It
	// must not be nil.
	//
	// TODO(s.chzhen):  Consider using a logger with its own prefix.
	logger *slog.Logger

	// uidToCustomConf maps persistent client UID to the custom client upstream
	// configuration.
	uidToCustomConf map[UID]*customUpstreamConfig

	// commonConf is the common upstream configuration.
	commonConf *CommonUpstreamConfig

	// confUpdate is the timestamp of the latest common upstream configuration
	// update.
	confUpdate time.Time
}

// newUpstreamManager returns the new properly initialized upstream manager.
func newUpstreamManager(logger *slog.Logger) (m *upstreamManager) {
	return &upstreamManager{
		logger:          logger,
		uidToCustomConf: make(map[UID]*customUpstreamConfig),
	}
}

// updateCommonUpstreamConfig updates the common upstream configuration and the
// timestamp of the latest configuration update.
func (m *upstreamManager) updateCommonUpstreamConfig(conf *CommonUpstreamConfig) {
	m.commonConf = conf
	m.confUpdate = time.Now()
}

// customUpstreamConfig returns the custom client upstream configuration.
func (m *upstreamManager) customUpstreamConfig(
	c *Persistent,
) (prxConf *proxy.CustomUpstreamConfig) {
	cliConf, ok := m.uidToCustomConf[c.UID]
	if ok && !m.isConfigChanged(c, cliConf) {
		return cliConf.prxConf
	}

	if ok && cliConf.prxConf != nil {
		err := cliConf.prxConf.Close()
		if err != nil {
			// TODO(s.chzhen):  Pass context.
			m.logger.Debug("closing custom upstream config", slogutil.KeyError, err)
		}
	}

	prxConf = newCustomUpstreamConfig(c, m.commonConf)
	m.uidToCustomConf[c.UID] = &customUpstreamConfig{
		prxConf:               prxConf,
		confUpdate:            m.confUpdate,
		upstreams:             slices.Clone(c.Upstreams),
		upstreamsCacheEnabled: c.UpstreamsCacheEnabled,
		upstreamsCacheSize:    c.UpstreamsCacheSize,
	}

	return prxConf
}

// isConfigChanged returns true if the update is necessary for the custom client
// upstream configuration.
func (m *upstreamManager) isConfigChanged(c *Persistent, cliConf *customUpstreamConfig) (ok bool) {
	if !slices.Equal(c.Upstreams, cliConf.upstreams) {
		return true
	}

	if c.UpstreamsCacheEnabled != cliConf.upstreamsCacheEnabled {
		return true
	}

	if c.UpstreamsCacheSize != cliConf.upstreamsCacheSize {
		return true
	}

	return !m.confUpdate.Equal(cliConf.confUpdate)
}

// clearUpstreamCache clears the upstream cache for each stored custom client
// upstream configuration.
func (m *upstreamManager) clearUpstreamCache() {
	for _, c := range m.uidToCustomConf {
		c.prxConf.ClearCache()
	}
}

// remove deletes the custom client upstream configuration.
func (m *upstreamManager) remove(c *Persistent) (err error) {
	cliConf, ok := m.uidToCustomConf[c.UID]
	if ok {
		return cliConf.prxConf.Close()
	}

	delete(m.uidToCustomConf, c.UID)

	return nil
}

// close shuts down each stored custom client upstream configuration.
func (m *upstreamManager) close() (err error) {
	var errs []error
	for _, c := range m.uidToCustomConf {
		if c.prxConf == nil {
			continue
		}

		errs = append(errs, c.prxConf.Close())
	}

	return errors.Join(errs...)
}

// newCustomUpstreamConfig returns the new properly initialized custom proxy
// upstream configuration for the client.
func newCustomUpstreamConfig(
	c *Persistent,
	conf *CommonUpstreamConfig,
) (prxConf *proxy.CustomUpstreamConfig) {
	upstreams := stringutil.FilterOut(c.Upstreams, aghnet.IsCommentOrEmpty)
	if len(upstreams) == 0 {
		return nil
	}

	upsConf, err := proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{
			Bootstrap:    conf.Bootstrap,
			Timeout:      time.Duration(conf.UpstreamTimeout),
			HTTPVersions: aghnet.UpstreamHTTPVersions(conf.UseHTTP3Upstreams),
			PreferIPv6:   conf.BootstrapPreferIPv6,
		},
	)
	if err != nil {
		// Should not happen because upstreams are already validated.  See
		// [Persistent.validate].
		panic(err)
	}

	return proxy.NewCustomUpstreamConfig(
		upsConf,
		c.UpstreamsCacheEnabled,
		int(c.UpstreamsCacheSize),
		conf.EDNSClientSubnetEnabled,
	)
}
