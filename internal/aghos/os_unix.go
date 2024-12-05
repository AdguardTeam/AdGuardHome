//go:build unix

package aghos

import (
	"os"
)

func sendShutdownSignal(_ chan<- os.Signal) {
	// On Unix we are already notified by the system.
}
