//go:build freebsd

package aghnet

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/AdguardTeam/golibs/osutil/executil"
)

// ObserveIPv6Addrs returns IPv6 interface address state for ifaceName.
func ObserveIPv6Addrs(
	ctx context.Context,
	_ *slog.Logger,
	cmdCons executil.CommandConstructor,
	ifaceName string,
) (states []IPv6AddrState, err error) {
	stdout := &bytes.Buffer{}
	err = executil.Run(ctx, cmdCons, &executil.CommandConfig{
		Path:   "ifconfig",
		Args:   []string{"-L", ifaceName, "inet6"},
		Stdout: stdout,
	})
	if err != nil {
		return nil, fmt.Errorf("running ifconfig -L %s inet6: %w", ifaceName, err)
	}

	return parseIfconfigIPv6Addrs(stdout.Bytes())
}
