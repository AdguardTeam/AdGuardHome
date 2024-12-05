//go:build tools

// Package tools and its main module are a nested internal module containing our
// development tool dependencies.
//
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module.
package tools

import (
	_ "github.com/fzipp/gocyclo/cmd/gocyclo"
	_ "github.com/golangci/misspell/cmd/misspell"
	_ "github.com/gordonklaus/ineffassign"
	_ "github.com/jstemmer/go-junit-report/v2"
	_ "github.com/kisielk/errcheck"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "github.com/uudashr/gocognit/cmd/gocognit"
	_ "golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness"
	_ "golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "honnef.co/go/tools/cmd/staticcheck"
	_ "mvdan.cc/gofumpt"
	_ "mvdan.cc/sh/v3/cmd/shfmt"
	_ "mvdan.cc/unparam"
)
