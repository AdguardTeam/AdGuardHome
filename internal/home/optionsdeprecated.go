package home

import (
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"
)

// filterNoOpOpts is a best-effort helper that returns a clone of opts with
// legacy no-op command-line options filtered out.  It assumes that the options
// used to be valid pre-v1.0.0.  It also prints appropriate warnings to stdout
// and stderr.
//
// The following legacy options are processed:
//  1. --no-mem-optimization has been a no-op for some time now.
//  2. --port/-p has been migrating their values into config, so safe to make it
//     a no-op.
//  3. --host is the same as --port, but -h is now help.  filterNoOpOpts
//     performs a best-effort check of the next option to distinguish between
//     the two.
//  4. --no-etc-hosts, if set, is returned to migrate into config.
//
// stdout and stderr must not be nil.
//
// TODO(a.garipov):  Once package ossvc learns to reason about service files,
// remove these options from them instead of ignoring them.  Alternatively, just
// remove this handling in a future version.
func filterNoOpOpts(opts []string, stdout, stderr io.Writer) (res []string, noEtcHosts bool) {
	res = make([]string, 0, len(opts))

	next, stop := iter.Pull(slices.Values(opts))
	defer stop()

	for opt, ok := next(); ok; {
		// Normalize long options to one dash.
		normOpt := opt
		if strings.HasPrefix(opt, "--") {
			normOpt = opt[1:]
		}

		switch normOpt {
		case "-host", "-port", "-p":
			// Print the warning and skip the argument.
			printNoOpWarning(opt, stdout, stderr)
			_, _ = next()
		case "-no-etc-hosts":
			printNoOpWarning(opt, stdout, stderr)
			noEtcHosts = true
		case "-no-mem-optimization":
			printNoOpWarning(opt, stdout, stderr)
		case "-h":
			res, opt, ok = handleHelpOrHost(res, next, stdout, stderr)

			continue
		default:
			res = append(res, opt)
		}

		opt, ok = next()
	}

	return res, noEtcHosts
}

// handleHelpOrHost handles the "-h" flag, which used to be treated either as
// "--help" or as "--host" in pre-v1.0.0 AdGuard Home.  handleHelpOrHost pulls
// the next option using next (if any), appends "-h" to orig (if necessary), and
// returns it along with the next option to be processed.  next, stdout, and
// stderr must not be nil.
func handleHelpOrHost(
	orig []string,
	next func() (v string, ok bool),
	stdout io.Writer,
	stderr io.Writer,
) (res []string, nextOpt string, ok bool) {
	res = orig
	nextOpt, ok = next()
	if !ok || strings.HasPrefix(nextOpt, "-") {
		// Either the final option or followed by another option.  Append the
		// "-h" and handle this next option, if any, on the next iteration.
		res = append(res, "-h")

		return res, nextOpt, ok
	}

	// Likely, a legacy use of -h as a hostname.  Skip this option and the next.
	printNoOpWarning("--host/-h USED AS HOSTNAME", stdout, stderr)
	nextOpt, ok = next()

	return res, nextOpt, ok
}

// msgFmtNoOp is the formatting message used to warn the user that an
// command-line option they're using is a no-op and will be removed.
const msgFmtNoOp = "WARNING: ADGUARD HOME OPTION %s IS A NO-OP AND TO BE REMOVED; " +
	"UPDATE YOUR SCRIPTS AND/OR REINSTALL ADGUARD HOME\n"

// printNoOpWarning warns the user that an command-line option they're using is
// a no-op and will be removed.  stdout and stderr must not be nil.
func printNoOpWarning(op string, stdout, stderr io.Writer) {
	// Log to both stderr and stdout to increase the chance that users see it.
	_, _ = fmt.Fprintf(stdout, msgFmtNoOp, op)
	_, _ = fmt.Fprintf(stderr, msgFmtNoOp, op)
}
