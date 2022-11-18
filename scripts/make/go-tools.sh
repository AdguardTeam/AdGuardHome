#!/bin/sh

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '1' ]
then
	set -x
	v_flags='-v'
	x_flags='-x'
elif [ "$verbose" -gt '0' ]
then
	set -x
	v_flags='-v'
	x_flags=''
else
	set +x
	v_flags=''
	x_flags=''
fi
readonly v_flags x_flags

set -e -f -u

go="${GO:-go}"
readonly go

# TODO(a.garipov): Add goconst?

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
	$v_flags\
	$x_flags\
	github.com/fzipp/gocyclo/cmd/gocyclo\
	github.com/golangci/misspell/cmd/misspell\
	github.com/gordonklaus/ineffassign\
	github.com/kisielk/errcheck\
	github.com/kyoh86/looppointer/cmd/looppointer\
	github.com/securego/gosec/v2/cmd/gosec\
	golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness\
	golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow\
	golang.org/x/vuln/cmd/govulncheck\
	honnef.co/go/tools/cmd/staticcheck\
	mvdan.cc/gofumpt\
	mvdan.cc/unparam\
	;
