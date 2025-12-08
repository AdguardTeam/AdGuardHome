//go:build !linux

package dnsforward

import (
	"context"
)

// createIpsets is a stub for non-Linux systems.
func (s *Server) createIpsets(ctx context.Context, config *IpsetCreateConfig) error {
	// IPSet is only supported on Linux
	return nil
}
