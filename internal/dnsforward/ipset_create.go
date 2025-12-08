//go:build linux

package dnsforward

import (
	"context"
	"fmt"
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/digineo/go-ipset/v2"
	"github.com/ti-mo/netfilter"
)

// createIpsets creates ipsets defined in the configuration if they don't exist.
// It skips ipsets that already exist and logs the action.
func (s *Server) createIpsets(ctx context.Context, config *IpsetCreateConfig) error {
	if config == nil || !config.Enabled || len(config.Sets) == 0 {
		return nil
	}

	s.logger.InfoContext(ctx, "creating ipsets if missing", "count", len(config.Sets))

	for _, setConfig := range config.Sets {
		err := s.createSingleIpset(ctx, setConfig)
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"failed to create ipset",
				"name", setConfig.Name,
				slogutil.KeyError, err,
			)
			// Continue with next ipset instead of failing completely
			continue
		}
	}

	return nil
}

// createSingleIpset creates a single ipset if it doesn't exist.
func (s *Server) createSingleIpset(ctx context.Context, config IpsetSetConfig) error {
	// Determine protocol family
	var family netfilter.ProtoFamily
	switch config.Family {
	case "inet", "ipv4":
		family = netfilter.ProtoIPv4
	case "inet6", "ipv6":
		family = netfilter.ProtoIPv6
	default:
		return fmt.Errorf("unknown family %q, expected inet or inet6", config.Family)
	}

	// Connect to netfilter
	conn, err := ipset.Dial(family, nil)
	if err != nil {
		return fmt.Errorf("dialing netfilter: %w", err)
	}
	defer func() {
		closeErr := conn.Close()
		if closeErr != nil {
			s.logger.WarnContext(
				ctx,
				"closing ipset connection",
				slogutil.KeyError, closeErr,
			)
		}
	}()

	// Check if ipset already exists
	_, err = conn.Header(config.Name)
	if err == nil {
		// Ipset exists, skip creation
		s.logger.InfoContext(
			ctx,
			"ipset already exists, skipping creation",
			"name", config.Name,
		)
		return nil
	}

	// Create the ipset
	s.logger.InfoContext(
		ctx,
		"creating ipset",
		"name", config.Name,
		"type", config.Type,
		"family", config.Family,
		"timeout", config.Timeout,
	)

	// Determine ipset type revision (typically 0 for basic types)
	var revision uint8 = 0

	// Prepare create options
	var opts []ipset.CreateDataOption

	// Add timeout if specified
	if config.Timeout > 0 {
		opts = append(opts, ipset.CreateDataTimeout(time.Duration(config.Timeout)*time.Second))
	}

	err = conn.Create(config.Name, config.Type, revision, family, opts...)
	if err != nil {
		return fmt.Errorf("creating ipset %q: %w", config.Name, err)
	}

	s.logger.InfoContext(
		ctx,
		"successfully created ipset",
		"name", config.Name,
		"type", config.Type,
	)

	return nil
}
