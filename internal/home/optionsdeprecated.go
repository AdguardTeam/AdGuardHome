package home

import (
	"fmt"
	"io"
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
func filterNoOpOpts(opts []string, stdout, stderr io.Writer) (res []string, noEtcHosts bool) {
	res = make([]string, 0, len(opts))
	for i := 0; i < len(opts); i++ {
		opt := opts[i]

		// Normalize long options to one dash.
		normOpt := opt
		if strings.HasPrefix(opt, "--") {
			normOpt = opt[1:]
		}

		switch normOpt {
		case "-host", "-port", "-p":
			printNoOpWarning(opt, stdout, stderr)
			i++
		case "-no-etc-hosts":
			printNoOpWarning(opt, stdout, stderr)
			noEtcHosts = true
		case "-no-mem-optimization":
			printNoOpWarning(opt, stdout, stderr)
		case "-h":
			i++
			res = handleHelpOrHost(res, opts[i:], stdout, stderr)
		default:
			res = append(res, opt)
		}
	}

	return res, noEtcHosts
}

// handleHelpOrHost handles the "-h" flag, which used to be treated either as
// "--help" or as "--host" in pre-v1.0.0 AdGuard Home.  handleHelpOrHost
// inspects the next option in the nextOpts subslice, if any, and appends the
// necessary options to orig and returns it.  stdout and stderr must not be nil.
func handleHelpOrHost(orig, nextOpts []string, stdout, stderr io.Writer) (res []string) {
	if len(nextOpts) == 0 {
		return append(orig, "-h")
	}

	nextOpt := nextOpts[0]
	if !strings.HasPrefix(nextOpt, "-") {
		// Likely, a legacy use of -h as a hostname.
		printNoOpWarning("--host/-h USED AS HOSTNAME", stdout, stderr)

		return orig
	}

	// Another option, not a hostname.  Append the new option and continue.
	return append(orig, "-h", nextOpt)
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
