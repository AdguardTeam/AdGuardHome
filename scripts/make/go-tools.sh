#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 3

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]
then
	set -x
	v_flags='-v=1'
	x_flags='-x=1'
elif [ "$verbose" -gt '0' ]
then
	set -x
	v_flags='-v=1'
	x_flags='-x=0'
else
	set +x
	v_flags='-v=0'
	x_flags='-x=0'
fi
readonly v_flags x_flags

set -e -f -u

go="${GO:-go}"
readonly go

# TODO(a.garipov): Add goconst?

# Remove only the actual binaries in the bin/ directory, as developers may add
# their own scripts there.  Most commonly, a script named “go” for tools that
# call the go binary and need a particular version.
rm -f\
	bin/errcheck\
	bin/fieldalignment\
	bin/gocognit\
	bin/gocyclo\
	bin/gofumpt\
	bin/gosec\
	bin/govulncheck\
	bin/ineffassign\
	bin/looppointer\
	bin/misspell\
	bin/nilness\
	bin/shadow\
	bin/staticcheck\
	bin/unparam\
	;

# Reset GOARCH and GOOS to make sure we install the tools for the native
# architecture even when we're cross-compiling the main binary, and also to
# prevent the "cannot install cross-compiled binaries when GOBIN is set" error.
env\
	GOARCH=""\
	GOBIN="${PWD}/bin"\
	GOOS=""\
	GOWORK='off'\
	"$go" install\
	--modfile=./internal/tools/go.mod\
	"$v_flags"\
	"$x_flags"\
	github.com/fzipp/gocyclo/cmd/gocyclo\
	github.com/golangci/misspell/cmd/misspell\
	github.com/gordonklaus/ineffassign\
	github.com/kisielk/errcheck\
	github.com/kyoh86/looppointer/cmd/looppointer\
	github.com/securego/gosec/v2/cmd/gosec\
	github.com/uudashr/gocognit/cmd/gocognit\
	golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment\
	golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness\
	golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow\
	golang.org/x/vuln/cmd/govulncheck\
	honnef.co/go/tools/cmd/staticcheck\
	mvdan.cc/gofumpt\
	mvdan.cc/unparam\
	;
