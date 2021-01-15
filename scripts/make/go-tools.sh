#!/bin/sh

verbose="${VERBOSE:-0}"

if [ "$verbose" -gt '1' ]
then
	set -x
	readonly v_flags='-v'
	readonly x_flags='-x'
elif [ "$verbose" -gt '0' ]
then
	set -x
	readonly v_flags='-v'
	readonly x_flags=''
else
	set +x
	readonly v_flags=''
	readonly x_flags=''
fi

set -e -f -u

go="${GO:-go}"

# TODO(a.garipov): Add goconst?

# Reset GOARCH and GOOS to make sure we install the tools for the native
# architecture even when we're cross-compiling the main binary, and also
# to prevent the "cannot install cross-compiled binaries when GOBIN is
# set" error.
env\
	GOARCH=""\
	GOOS=""\
	GOBIN="${PWD}/bin"\
	"$go" install --modfile=./internal/tools/go.mod\
	$v_flags $x_flags\
	github.com/fzipp/gocyclo/cmd/gocyclo\
	github.com/golangci/misspell/cmd/misspell\
	github.com/gordonklaus/ineffassign\
	github.com/kisielk/errcheck\
	github.com/kyoh86/looppointer/cmd/looppointer\
	github.com/securego/gosec/v2/cmd/gosec\
	golang.org/x/lint/golint\
	golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness\
	golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow\
	honnef.co/go/tools/cmd/staticcheck\
	mvdan.cc/gofumpt\
	mvdan.cc/unparam\
	;
