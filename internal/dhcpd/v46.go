package dhcpd

import (
	"time"
)

// Currently used defaults for ifaceDNSAddrs.
const (
	defaultMaxAttempts int = 10

	defaultBackoff time.Duration = 500 * time.Millisecond
)
