// Package aghslog contains logging constants and helpers.
package aghslog

import (
	"log/slog"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// PrefixDNSProxy is the prefix for DNS proxy logs.
const PrefixDNSProxy = "dnsproxy"

const (
	// KeyClientName is the log attribute for the client name.
	KeyClientName = "client_name"

	// KeyUpstreamType is the log attribute for the upstream types.  See the
	// UpstreamType* constants below.
	KeyUpstreamType = "upstream_type"
)

const (
	// UpstreamTypeBootstrap is the log attribute value for bootstrap upstreams.
	UpstreamTypeBootstrap = "bootstrap"

	// UpstreamTypeCustom is the log attribute value for custom upstreams.
	UpstreamTypeCustom = "custom"

	// UpstreamTypeFallback is the log attribute value for fallback upstreams.
	UpstreamTypeFallback = "fallback"

	// UpstreamTypeMain is the log attribute value for main upstreams.
	UpstreamTypeMain = "main"

	// UpstreamTypeLocal is the log attribute value for upstreams used for
	// resolving PTR records for local addresses.
	UpstreamTypeLocal = "local"

	// UpstreamTypeService is the log attribute value for upstreams used for
	// safe browsing and parental services.
	UpstreamTypeService = "service"

	// UpstreamTypeTest is the log attribute value for upstreams used for
	// testing and validation.
	UpstreamTypeTest = "test"
)

// NewForUpstream returns a new logger with a prefix for logs related to a
// specific upstream type.
func NewForUpstream(baseLogger *slog.Logger, typ string) (l *slog.Logger) {
	return baseLogger.With(slogutil.KeyPrefix, PrefixDNSProxy, KeyUpstreamType, typ)
}
