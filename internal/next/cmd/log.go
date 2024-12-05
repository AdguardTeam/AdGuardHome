package cmd

import (
	"io"
	"log/slog"
	"os"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// newBaseLogger constructs a base logger based on the command-line options.
// opts must not be nil.
func newBaseLogger(opts *options) (baseLogger *slog.Logger) {
	var output io.Writer
	switch opts.confFile {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "syslog":
		// TODO(a.garipov):  Add a syslog handler to golibs.
	default:
		// TODO(a.garipov):  Use the path.
	}

	lvl := slog.LevelInfo
	if opts.verbose {
		lvl = slog.LevelDebug
	}

	return slogutil.New(&slogutil.Config{
		Output: output,
		// TODO(a.garipov):  Get from config?
		Format: slogutil.FormatText,
		Level:  lvl,
		// TODO(a.garipov):  Get from config.
		AddTimestamp: true,
	})
}
