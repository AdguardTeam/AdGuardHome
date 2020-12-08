#!/bin/sh

test "$VERBOSE" = '1' && set -x
set -e -f -u

# TODO(a.garipov): Add goconst?

env GOBIN="${PWD}/bin" "$GO" install --modfile=./internal/tools/go.mod\
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
