//go:build windows

package ossvc

import (
	"context"

	"github.com/AdguardTeam/golibs/errors"
)

// reload is a Windows platform implementation of the Reload method of the
// [ReloadManager] interface for *manager.
func (*manager) reload(context.Context, ServiceName) error {
	return errors.ErrUnsupported
}
